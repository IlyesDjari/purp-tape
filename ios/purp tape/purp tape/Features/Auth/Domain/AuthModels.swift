import Foundation

// MARK: - Authentication Models

public struct SignUpRequest: Sendable, Codable {
    public let email: String
    public let password: String
    public let displayName: String

    public init(email: String, password: String, displayName: String) {
        self.email = email
        self.password = password
        self.displayName = displayName
    }
}

public struct SignInRequest: Sendable, Codable {
    public let email: String
    public let password: String

    public init(email: String, password: String) {
        self.email = email
        self.password = password
    }
}

public struct PasswordResetRequest: Sendable, Codable {
    public let email: String

    public init(email: String) {
        self.email = email
    }
}

public enum AuthError: LocalizedError, Sendable {
    case invalidEmail
    case invalidPassword
    case emailAlreadyExists
    case userNotFound
    case invalidCredentials
    case sessionExpired
    case networkError(String)
    case unknownError(String)
    case socialAuthFailed(String)
    case passwordResetFailed

    public var errorDescription: String? {
        switch self {
        case .invalidEmail:
            return "Invalid email address"
        case .invalidPassword:
            return "Password must be at least 8 characters"
        case .emailAlreadyExists:
            return "Email already registered"
        case .userNotFound:
            return "User not found"
        case .invalidCredentials:
            return "Invalid email or password"
        case .sessionExpired:
            return "Your session has expired. Please sign in again"
        case let .networkError(message):
            return "Network error: \(message)"
        case let .unknownError(message):
            return "Error: \(message)"
        case let .socialAuthFailed(provider):
            return "\(provider) sign in failed"
        case .passwordResetFailed:
            return "Password reset failed. Please try again"
        }
    }

    public var recoverySuggestion: String? {
        switch self {
        case .invalidEmail:
            return "Please check your email format"
        case .invalidPassword:
            return "Use a stronger password with mixed characters"
        case .emailAlreadyExists:
            return "Try signing in instead"
        case .invalidCredentials:
            return "Check your credentials and try again"
        case .sessionExpired:
            return "Please authenticate again"
        default:
            return "Please try again"
        }
    }
}

// MARK: - Social Auth Provider

public enum SocialAuthProvider: String, Sendable, CaseIterable {
    case apple = "apple"

    public var displayName: String {
        switch self {
        case .apple:
            return "Apple"
        }
    }
}
