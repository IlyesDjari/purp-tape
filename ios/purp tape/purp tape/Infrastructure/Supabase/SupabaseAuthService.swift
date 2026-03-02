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
    private let vaultSyncInterval: TimeInterval = 300 // 5 minutes - refresh from vault periodically

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
            
            // OPTIMIZATION: Proactively refresh if expiring soon (< 30 seconds)
            if timeUntilExpiry < 30 && timeUntilExpiry > 0 {
                logger.warning("Token expiring soon, refreshing proactively")
                do {
                    return try await refreshIfNeeded()
                } catch {
                    logger.warning("Proactive refresh failed, using current token")
                    return cached
                }
            }
            
            // Periodically sync with vault in case app was backgrounded
            let shouldSyncVault = lastVaultSyncTime == nil || 
                                  Date().timeIntervalSince(lastVaultSyncTime!) > vaultSyncInterval
            if shouldSyncVault {
                Task { await syncVaultInBackground() }
            }
            
            return cached
        }
        
        // Load from vault only if memory cache is empty or expired
        if let cached = try await vault.loadSession(), !cached.isExpired {
            inMemorySession = cached
            lastVaultSyncTime = Date()
            let tokenPrefix = String(cached.accessToken.prefix(20))
            logger.auth("Using cached session - Token: \(tokenPrefix)...")
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
            return try await vault.loadSession()
        }
    }

    public func signIn(email: String, password: String) async throws -> AuthSession {
        logger.auth("Signing in user: \(email)")
        let session = try await client.auth.signIn(email: email, password: password)
        let mapped = map(session)
        let tokenPrefix = String(mapped.accessToken.prefix(20))
        logger.success("Sign in successful - Token: \(tokenPrefix)...")
        
        // PERFORMANCE: Update both memory and vault
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
        let tokenPrefix = String(mapped.accessToken.prefix(20))
        logger.success("Apple Sign In successful - Token: \(tokenPrefix)...")
        
        // PERFORMANCE: Update both memory and vault
        inMemorySession = mapped
        lastVaultSyncTime = Date()
        try await vault.saveSession(mapped)
        didRestorePersistedSession = true
        return mapped
    }

    public func signOut() async throws {
        logger.auth("Signing out user")
        // OPTIMIZATION: Skip restoration - we're about to clear anyway
        try await client.auth.signOut()
        try await vault.clearSession()
        inMemorySession = nil  // Clear memory cache
        lastVaultSyncTime = nil
        didRestorePersistedSession = false  // Reset for next sign in
        logger.success("User signed out successfully")
    }

    public func refreshIfNeeded() async throws -> AuthSession {
        logger.auth("Refreshing authentication session")
        try await restorePersistedSessionIfNeeded()

        do {
            let refreshed = try await client.auth.refreshSession()
            let mapped = map(refreshed)
            let tokenPrefix = String(mapped.accessToken.prefix(20))
            logger.success("Session refreshed - Token: \(tokenPrefix)...")
            
            // PERFORMANCE: Update both memory and vault  
            inMemorySession = mapped
            lastVaultSyncTime = Date()
            try await vault.saveSession(mapped)
            return mapped
        } catch {
            if let current = try await vault.loadSession(), !current.isExpired {
                logger.warning("Refresh failed, using cached session")
                inMemorySession = current  // Update memory cache
                lastVaultSyncTime = Date()
                return current
            }
            logger.error("Failed to refresh session")
            throw error
        }
    }

    private func restorePersistedSessionIfNeeded() async throws {
        guard !didRestorePersistedSession else { return }
        defer { didRestorePersistedSession = true }

        guard let cached = try await vault.loadSession() else { return }
        inMemorySession = cached  // Update memory cache
        lastVaultSyncTime = Date()
        _ = try await client.auth.setSession(
            accessToken: cached.accessToken,
            refreshToken: cached.refreshToken
        )
    }
    
    /// PERFORMANCE: Background sync with vault - avoids blocking currentSession()
    private func syncVaultInBackground() {
        Task {
            do {
                if let latestSession = inMemorySession,
                   !latestSession.isExpired {
                    // Quietly update vault in background
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
