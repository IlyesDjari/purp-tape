import Foundation
import Supabase

public actor SupabaseAuthService: AuthService {
    private let client: SupabaseClient
    private let vault: KeychainVault
    private var didRestorePersistedSession = false
    private var authStateTask: Task<Void, Never>?

    public init(supabaseURL: URL, supabaseAnonKey: String, vault: KeychainVault) {
        self.client = SupabaseClient(supabaseURL: supabaseURL, supabaseKey: supabaseAnonKey)
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
        try await restorePersistedSessionIfNeeded()

        do {
            let session = try await client.auth.session
            let mapped = map(session)
            try await vault.saveSession(mapped)
            return mapped
        } catch {
            return try await vault.loadSession()
        }
    }

    public func signIn(email: String, password: String) async throws -> AuthSession {
        let session = try await client.auth.signIn(email: email, password: password)
        let mapped = map(session)
        try await vault.saveSession(mapped)
        didRestorePersistedSession = true
        return mapped
    }

    public func signOut() async throws {
        try await restorePersistedSessionIfNeeded()
        try await client.auth.signOut()
        try await vault.clearSession()
    }

    public func refreshIfNeeded() async throws -> AuthSession {
        try await restorePersistedSessionIfNeeded()

        if let current = try await vault.loadSession(), !current.isExpired {
            return current
        }

        let refreshed = try await client.auth.refreshSession()
        let mapped = map(refreshed)
        try await vault.saveSession(mapped)
        return mapped
    }

    private func restorePersistedSessionIfNeeded() async throws {
        guard !didRestorePersistedSession else { return }
        defer { didRestorePersistedSession = true }

        guard let cached = try await vault.loadSession() else { return }
        _ = try await client.auth.setSession(
            accessToken: cached.accessToken,
            refreshToken: cached.refreshToken
        )
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
