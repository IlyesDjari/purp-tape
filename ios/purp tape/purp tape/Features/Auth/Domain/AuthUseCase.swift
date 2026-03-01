import Foundation

public protocol AuthRepository: Sendable {
    // Email/Password Auth
    func signUp(request: SignUpRequest) async throws -> AuthSession
    func signIn(request: SignInRequest) async throws -> AuthSession
    func signOut() async throws
    
    // Session Management
    func currentSession() async throws -> AuthSession?
    func refreshSessionIfNeeded() async throws -> AuthSession
    
    // Password Recovery
    func requestPasswordReset(email: String) async throws
    func resetPassword(token: String, newPassword: String) async throws
    
    // Social Auth
    func signInWithApple(token: String, nonce: String) async throws -> AuthSession
    
    // Validation
    func validateEmail(_ email: String) throws
    func validatePassword(_ password: String) throws
}

public protocol AuthUseCase: Sendable {
    func execute() async throws
}

public protocol AuthUseCase2<Output>: Sendable {
    associatedtype Output
    associatedtype Input
    func execute(_ input: Input) async throws -> Output
}
