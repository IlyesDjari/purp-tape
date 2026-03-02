import Foundation

public final class AuthViewModel: ObservableObject {
    @Published public var currentSession: AuthSession?
    @Published public var isLoading = false
    @Published public var error: AuthError?
    @Published public var isAuthenticated = false
    @Published public var didResolveInitialSession = false
    
    private let authRepository: AuthRepository
    private let authService: AuthService
    private var sessionCheckTask: Task<Void, Never>?
    
    public init(authRepository: AuthRepository, authService: AuthService) {
        self.authRepository = authRepository
        self.authService = authService
    }
    
    // MARK: - Accessors
    
    /// Exposed for dependency injection into other services
    public var service: AuthService {
        authService
    }
    
    // MARK: - Session Management
    
    @MainActor
    public func startSessionMonitoring() {
        guard sessionCheckTask == nil else { return }
        sessionCheckTask = Task {
            defer { self.didResolveInitialSession = true }
            do {
                // Initial check
                self.currentSession = try await authService.currentSession()
                self.isAuthenticated = self.currentSession != nil
            } catch {
                self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
                self.isAuthenticated = false
            }
        }
    }
    
    @MainActor
    public func stopSessionMonitoring() {
        sessionCheckTask?.cancel()
        sessionCheckTask = nil
    }
    
    @MainActor
    public func refreshSessionIfNeeded() async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            let session = try await authRepository.refreshSessionIfNeeded()
            self.currentSession = session
            self.isAuthenticated = true
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
            self.isAuthenticated = false
        }
    }
    
    // MARK: - Sign In
    
    @MainActor
    public func signIn(email: String, password: String) async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            let request = SignInRequest(email: email, password: password)
            let session = try await authRepository.signIn(request: request)
            self.currentSession = session
            self.isAuthenticated = true
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
            self.isAuthenticated = false
        }
    }
    
    // MARK: - Sign Out
    
    @MainActor
    public func signOut() async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            try await authRepository.signOut()
            self.currentSession = nil
            self.isAuthenticated = false
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
        }
    }
    
    // MARK: - Social Auth
    
    @MainActor
    public func signInWithApple(token: String, nonce: String) async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            let session = try await authRepository.signInWithApple(token: token, nonce: nonce)
            self.currentSession = session
            self.isAuthenticated = true
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
        }
    }
    
    // MARK: - Password Reset
    
    @MainActor
    public func requestPasswordReset(email: String) async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            try await authRepository.requestPasswordReset(email: email)
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
        }
    }
    
    @MainActor
    public func resetPassword(token: String, newPassword: String) async {
        isLoading = true
        defer { isLoading = false }
        
        do {
            try await authRepository.resetPassword(token: token, newPassword: newPassword)
            self.error = nil
        } catch {
            self.error = error as? AuthError ?? AuthError.unknownError(error.localizedDescription)
        }
    }
    
    // MARK: - Validation
    
    public func validateEmail(_ email: String) -> Bool {
        do {
            try authRepository.validateEmail(email)
            return true
        } catch {
            return false
        }
    }
    
    public func validatePassword(_ password: String) -> Bool {
        do {
            try authRepository.validatePassword(password)
            return true
        } catch {
            return false
        }
    }
}
