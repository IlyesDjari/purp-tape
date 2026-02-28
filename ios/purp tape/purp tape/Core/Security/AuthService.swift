import Foundation

public protocol AuthService: Sendable {
    func currentSession() async throws -> AuthSession?
    func signIn(email: String, password: String) async throws -> AuthSession
    func signOut() async throws
    func refreshIfNeeded() async throws -> AuthSession
}
