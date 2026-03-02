import Foundation
import Supabase

public actor SupabaseAuthService: AuthService {
    private let client: SupabaseClient
    private let vault: KeychainVault
    private var didRestorePersistedSession = false
    private var authStateTask: Task<Void, Never>?
    private let logger = DebugLogger(category: "auth.supabase")
    
    // PERFORMANCE: In-memory cache avoids keychain reads on every API request
    private var inMemorySession: AuthSession?
    private var lastVaultSyncTime: Date?
    private let vaultSyncInterval: TimeInterval = 300 // 5 minutes
    
    // SECURITY: Token refresh synchronization - prevents concurrent refresh attempts
    private var activeRefreshTask: Task<AuthSession, Error>?
    private var lastRefreshAttempt: Date?
    private let minRefreshInterval: TimeInterval = 1.0

    public init(supabaseURL: URL, supabaseAnonKey: String, vault: KeychainVault) {
        self.client = SupabaseClient(
            supabaseURL: supabaseURL,
            supabaseKey: supabaseAnonKey,
            options: SupabaseClientOptions(
                auth: .init(emitLocalSessionAsInitialSession: true)
            )
        )
        self.vault = vault

        authStateTask = Task { [client, vault] in
            for await (_, session) in client.auth.authStateChanges {
                do {
                    if let session {
                        let mapped = AuthSession(
                            accessToken: session.accessToken,
                            refreshToken: session.refreshToken,
                            userID: session.user.id,
                            expiresAt: Date(timeIntervalSince1970: session.expiresAt)
                        )
                        try await vault.saveSession(mapped)
                    } else {
                        try await vault.clearSession()
                    }
                } catch {
                    continue
                }
            }
        }
    }

    deinit {
        authStateTask?.cancel()
    }

    public func currentSession() async throws -> AuthSession? {
        // PERFORMANCE: Check memory cache first (< 1ms)
        if let cached = inMemorySession, !cached.isExpired {
            let timeUntilExpiry = cached.expiresAt.timeIntervalSince(Date())
            
            // Proactively refresh if expiring soon (< 30 seconds)
            if timeUntilExpiry < 30 && timeUntilExpiry > 0 {
                logger.warning("Token expiring soon (\(Int(timeUntilExpiry))s), refreshing proactively")
                do {
                    return try await refreshSessionDeduped()
                } catch {
                    logger.warning("Proactive refresh failed, using current token")
                    return cached
                }
            }
            
            // Periodically sync with vault in case app was backgrounded
            let shouldSyncVault = lastVaultSyncTime == nil || 
                                  Date().timeIntervalSince(lastVaultSyncTime!) > vaultSyncInterval
            if shouldSyncVault {
                Task { syncVaultInBackground() }
            }
            
            return cached
        }
        
        // Load from vault only if memory cache is empty or expired
        if let cached = try await vault.loadSession(), !cached.isExpired {
            inMemorySession = cached
            lastVaultSyncTime = Date()
            let tokenPrefix = String(cached.accessToken.prefix(20))
            logger.auth("Using vault token: \(tokenPrefix)...")
            return cached
        }
        
        // Re-establish from Supabase if cache is depleted
        try await restorePersistedSessionIfNeeded()
        
        do {
            let session = try await client.auth.session
            let mapped = map(session)
            inMemorySession = mapped
            lastVaultSyncTime = Date()
            try await vault.saveSession(mapped)
            return mapped
        } catch {
            let fallback = try await vault.loadSession()
            if fallback != nil {
                logger.warning("Supabase session retrieval failed, using vault fallback")
            }
            return fallback
        }
    }

    public func signIn(email: String, password: String) async throws -> AuthSession {
        logger.auth("Signing in user: \(email)")
        let session = try await client.auth.signIn(email: email, password: password)
        let mapped = map(session)
        logger.success("Sign in successful")
        
        inMemorySession = mapped
        lastVaultSyncTime = Date()
        try await vault.saveSession(mapped)
        didRestorePersistedSession = true
        return mapped
    }

    public func signInWithApple(idToken: String, nonce: String?) async throws -> AuthSession {
        logger.auth("Initiating Apple Sign In")
        let session = try await client.auth.signInWithIdToken(
            credentials: OpenIDConnectCredentials(
                provider: .apple,
                idToken: idToken,
                nonce: nonce
            )
        )

        let mapped = map(session)
        logger.success("Apple Sign In successful")
        
        inMemorySession = mapped
        lastVaultSyncTime = Date()
        try await vault.saveSession(mapped)
        didRestorePersistedSession = true
        return mapped
    }

    public func signOut() async throws {
        logger.auth("Signing out user")
        try await client.auth.signOut()
        try await vault.clearSession()
        inMemorySession = nil
        lastVaultSyncTime = nil
        didRestorePersistedSession = false
        logger.success("User signed out successfully")
    }

    public func refreshIfNeeded() async throws -> AuthSession {
        return try await refreshSessionDeduped()
    }
    
    /// Deduplicated refresh to prevent concurrent refresh storms
    private func refreshSessionDeduped() async throws -> AuthSession {
        // Check if we've already refreshed very recently
        if let lastAttempt = lastRefreshAttempt,
           Date().timeIntervalSince(lastAttempt) < minRefreshInterval {
            if let activeTask = activeRefreshTask {
                return try await activeTask.value
            }
        }
        
        // If there's an active refresh task, wait for it
        if let activeTask = activeRefreshTask {
            return try await activeTask.value
        }
        
        // Start new refresh
        let task = Task {
            try await performTokenRefresh()
        }
        
        activeRefreshTask = task
        lastRefreshAttempt = Date()
        
        defer {
            activeRefreshTask = nil
        }
        
        return try await task.value
    }
    
    /// Performs the actual token refresh
    private func performTokenRefresh() async throws -> AuthSession {
        logger.auth("Refreshing authentication session")
        try await restorePersistedSessionIfNeeded()

        do {
            let refreshed = try await client.auth.refreshSession()
            let mapped = map(refreshed)
            
            // Validate access token is present
            guard !mapped.accessToken.isEmpty else {
                logger.error("Token refresh returned empty access token")
                throw AuthError.unknownError("Token refresh returned invalid token")
            }
            
            // Update both memory and vault
            inMemorySession = mapped
            lastVaultSyncTime = Date()
            try await vault.saveSession(mapped)
            
            let expiresIn = Int(mapped.expiresAt.timeIntervalSince(Date()))
            logger.success("Session refreshed (expires in \(expiresIn)s)")
            return mapped
        } catch {
            logger.error("Token refresh failed: \(error.localizedDescription)")
            
            // Fallback: use valid cached session if refresh fails
            if let current = try await vault.loadSession(), !current.isExpired {
                let timeLeft = Int(current.expiresAt.timeIntervalSince(Date()))
                if timeLeft > 0 {
                    logger.warning("Refresh failed, using cached session (expires in \(timeLeft)s)")
                    inMemorySession = current
                    lastVaultSyncTime = Date()
                    return current
                }
            }
            
            logger.error("No valid cached session available after refresh failure")
            throw error
        }
    }

    private func restorePersistedSessionIfNeeded() async throws {
        guard !didRestorePersistedSession else { return }
        defer { didRestorePersistedSession = true }

        guard let cached = try await vault.loadSession() else { return }
        inMemorySession = cached
        lastVaultSyncTime = Date()
        _ = try await client.auth.setSession(
            accessToken: cached.accessToken,
            refreshToken: cached.refreshToken
        )
    }
    
    /// Background sync with vault
    private func syncVaultInBackground() {
        Task {
            do {
                if let latestSession = inMemorySession,
                   !latestSession.isExpired {
                    try await vault.saveSession(latestSession)
                    lastVaultSyncTime = Date()
                }
            } catch {
                logger.warning("Background vault sync failed: \(error.localizedDescription)")
            }
        }
    }

    private func map(_ session: Session) -> AuthSession {
        AuthSession(
            accessToken: session.accessToken,
            refreshToken: session.refreshToken,
            userID: session.user.id,
            expiresAt: Date(timeIntervalSince1970: session.expiresAt)
        )
    }
}
