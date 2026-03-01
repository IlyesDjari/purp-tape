import Foundation

public struct AuthContainer {
    public let authViewModel: AuthViewModel
    public let authService: AuthService
    public let authRepository: AuthRepository
    
    public init(
        authService: AuthService,
        vault: KeychainVault? = nil
    ) {
        self.authService = authService
        self.authRepository = DefaultAuthRepository(
            authService: authService
        )
        self.authViewModel = AuthViewModel(
            authRepository: self.authRepository,
            authService: authService
        )
    }
}
