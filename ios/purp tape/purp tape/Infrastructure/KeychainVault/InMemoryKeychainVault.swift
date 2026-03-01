import Foundation

public actor InMemoryKeychainVault: KeychainVault {
    private var cachedSession: AuthSession?

    public init() {}

    public func saveSession(_ session: AuthSession) async throws {
        cachedSession = session
    }

    public func loadSession() async throws -> AuthSession? {
        cachedSession
    }

    public func clearSession() async throws {
        cachedSession = nil
    }
}
