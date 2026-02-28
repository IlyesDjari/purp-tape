import Foundation

public protocol KeychainVault: Sendable {
    func saveSession(_ session: AuthSession) async throws
    func loadSession() async throws -> AuthSession?
    func clearSession() async throws
}
