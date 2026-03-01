import Foundation

public actor DefaultAuthRepository: AuthRepository {
    private let authService: AuthService
    
    public init(authService: AuthService, validator: AuthInputValidator? = nil) {
        self.authService = authService
        // validator parameter is for testing purposes but not stored
    }
    
    // MARK: - Email/Password Auth
    
    public func signUp(request: SignUpRequest) async throws -> AuthSession {
        try validateEmail(request.email)
        try validatePassword(request.password)
        
        // Note: The actual signup is handled by Supabase Auth
        // This would typically be called after user creation, but Supabase
        // handles the signup process directly through the auth service
        
        throw AuthError.unknownError("SignUp should be handled via AuthService.signUp()")
    }
    
    public func signIn(request: SignInRequest) async throws -> AuthSession {
        try validateEmail(request.email)
        
        do {
            return try await authService.signIn(email: request.email, password: request.password)
        } catch {
            throw mapError(error)
        }
    }
    
    public func signOut() async throws {
        try await authService.signOut()
    }
    
    // MARK: - Session Management
    
    public func currentSession() async throws -> AuthSession? {
        try await authService.currentSession()
    }
    
    public func refreshSessionIfNeeded() async throws -> AuthSession {
        try await authService.refreshIfNeeded()
    }
    
    // MARK: - Password Recovery
    
    public func requestPasswordReset(email: String) async throws {
        try validateEmail(email)
        // This would typically call a password reset endpoint
        // For now, delegated to Supabase's native functionality
        throw AuthError.passwordResetFailed
    }
    
    public func resetPassword(token: String, newPassword: String) async throws {
        try validatePassword(newPassword)
        // This would typically verify the reset token and update password
        throw AuthError.passwordResetFailed
    }
    
    // MARK: - Social Auth
    
    public func signInWithApple(token: String, nonce: String) async throws -> AuthSession {
        guard !token.isEmpty else {
            throw AuthError.invalidCredentials
        }

        do {
            return try await authService.signInWithApple(idToken: token, nonce: nonce)
        } catch {
            throw mapError(error)
        }
    }
    
    // MARK: - Validation
    
    nonisolated public func validateEmail(_ email: String) throws {
        try AuthInputValidator.validateEmail(email)
    }
    
    nonisolated public func validatePassword(_ password: String) throws {
        try AuthInputValidator.validatePassword(password)
    }
    
    // MARK: - Private
    
    private func mapError(_ error: Error) -> AuthError {
        if let authError = error as? AuthError {
            return authError
        }
        return .unknownError(error.localizedDescription)
    }
}

public struct AuthInputValidator {
    public init() {}
    
    public static func validateEmail(_ email: String) throws {
        let email = email.trimmingCharacters(in: .whitespaces)
        
        guard !email.isEmpty else {
            throw AuthError.invalidEmail
        }
        
        let emailPattern = "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
        let regex = try NSRegularExpression(pattern: emailPattern)
        let range = NSRange(email.startIndex..<email.endIndex, in: email)
        
        guard regex.firstMatch(in: email, range: range) != nil else {
            throw AuthError.invalidEmail
        }
    }
    
    public static func validatePassword(_ password: String) throws {
        guard password.count >= 8 else {
            throw AuthError.invalidPassword
        }
        
        let hasUppercase = password.range(of: "[A-Z]", options: .regularExpression) != nil
        let hasLowercase = password.range(of: "[a-z]", options: .regularExpression) != nil
        let hasNumber = password.range(of: "[0-9]", options: .regularExpression) != nil
        
        guard hasUppercase && hasLowercase && hasNumber else {
            throw AuthError.invalidPassword
        }
    }
}
