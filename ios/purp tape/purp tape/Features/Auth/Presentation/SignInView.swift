import SwiftUI
import AuthenticationServices

public struct SignInView: View {
    @StateObject private var viewModel: AuthViewModel
    @State private var email = ""
    @State private var password = ""
    @State private var showPassword = false
    @State private var currentNonce = ""
    @State private var emailError: String?
    @State private var passwordError: String?
    @Environment(\.navigationManager) var navigationManager
    
    @FocusState private var focusedField: Field?
    
    enum Field {
        case email
        case password
    }
    
    public init(viewModel: AuthViewModel) {
        self._viewModel = StateObject(wrappedValue: viewModel)
    }
    
    public var body: some View {
        ZStack {
            VStack(spacing: 0) {
                    // Header
                    VStack(alignment: .leading, spacing: Spacing.sm) {
                        Text("Welcome Back")
                            .displayLarge()
                            .foregroundColor(PurpTapeColors.text)
                        Text("Sign in to access your projects")
                            .bodyLarge()
                            .foregroundColor(PurpTapeColors.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .paddingHorizontalLG()
                    .paddingTopLG()
                    .padding(.top, Spacing.xl)
                    .paddingBottomLG()
                    
                    // Form
                    ScrollView {
                        VStack(spacing: Spacing.xl) {
                            // Email Field
                            VStack(alignment: .leading, spacing: Spacing.sm) {
                                Label("Email", systemImage: "envelope.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                
                                TextField("Enter your email", text: $email)
                                    .textInputAutocapitalization(.never)
                                    .autocorrectionDisabled()
                                    .keyboardType(.emailAddress)
                                    .focused($focusedField, equals: .email)
                                    .onChange(of: email) { _, newValue in
                                        emailError = nil
                                    }
                                    .font(PurpTapeTypography.bodyMedium)
                                    .purpTapeInputContainer()
                                
                                if let emailError {
                                    Text(emailError)
                                        .captionSmall()
                                        .foregroundColor(PurpTapeColors.error)
                                }
                            }
                            
                            // Password Field
                            VStack(alignment: .leading, spacing: Spacing.sm) {
                                Label("Password", systemImage: "lock.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                
                                HStack {
                                    if showPassword {
                                        TextField("Enter your password", text: $password)
                                            .font(PurpTapeTypography.bodyMedium)
                                            .focused($focusedField, equals: .password)
                                    } else {
                                        SecureField("Enter your password", text: $password)
                                            .font(PurpTapeTypography.bodyMedium)
                                            .focused($focusedField, equals: .password)
                                    }
                                    
                                    Button(action: { showPassword.toggle() }) {
                                        Image(systemName: showPassword ? "eye.slash.fill" : "eye.fill")
                                            .foregroundColor(PurpTapeColors.textSecondary)
                                    }
                                }
                                .purpTapeInputContainer()
                                
                                if let passwordError {
                                    Text(passwordError)
                                        .captionSmall()
                                        .foregroundColor(PurpTapeColors.error)
                                }
                            }
                            
                            // Forgot Password Link
                            Button(action: {
                                navigationManager.goToForgotPassword()
                            }) {
                                Text("Forgot password?")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.primary)
                            }
                            .frame(maxWidth: .infinity, alignment: .trailing)
                            
                            Spacer()
                        }
                        .paddingHorizontalLG()
                        .paddingVerticalLG()
                    }
                    
                    // Error Message
                    if let error = viewModel.error {
                        VStack(spacing: Spacing.sm) {
                            HStack(spacing: Spacing.md) {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .foregroundColor(PurpTapeColors.error)
                                Text(error.errorDescription ?? "An error occurred")
                                    .bodySmall()
                                    .foregroundColor(PurpTapeColors.error)
                                Spacer()
                            }
                            .paddingMD()
                            .background(PurpTapeColors.error.opacity(0.1))
                            .cornerRadiusMD()
                        }
                        .paddingHorizontalLG()
                        .paddingBottomLG()
                    }
                    
                    // Sign In Button
                    PurpTapePrimaryButton(
                        "Sign In",
                        isLoading: viewModel.isLoading,
                        isDisabled: !isFormValid
                    ) {
                        Task {
                            await viewModel.signIn(email: email, password: password)
                        }
                    }
                    .paddingHorizontalLG()
                    .paddingBottomLG()
                    
                    // Social Auth
                    VStack(spacing: Spacing.md) {
                        Text("Or continue with")
                            .bodyMedium()
                            .foregroundColor(PurpTapeColors.textSecondary)

                        SignInWithAppleButton { request in
                            let nonce = randomNonceString()
                            currentNonce = nonce
                            request.requestedScopes = [.fullName, .email]
                            request.nonce = sha256(nonce)
                        } onCompletion: { result in
                            handleAppleSignIn(result)
                        }
                        .signInWithAppleButtonStyle(.white)
                        .frame(height: 48)
                        .cornerRadius(8)
                        .overlay(
                            RoundedRectangle(cornerRadius: CornerRadius.md)
                                .stroke(PurpTapeColors.border, lineWidth: 1)
                        )
                        .shadow(color: PurpTapeColors.shadowLight, radius: 6, x: 0, y: 2)
                    }
                    .paddingHorizontalLG()
                    .padding(.bottom, Spacing.xl)
                    
                    // Sign Up Link
                    HStack(spacing: Spacing.xs) {
                        Text("Don't have an account?")
                            .bodyMedium()
                            .foregroundColor(PurpTapeColors.textSecondary)
                        Button("Sign Up", action: {
                            navigationManager.goToSignUp()
                        })
                        .font(PurpTapeTypography.bodyMedium)
                        .fontWeight(.semibold)
                        .foregroundColor(PurpTapeColors.primary)
                    }
                    .padding(.bottom, 40)
            }
        }
    }
    
    private var isFormValid: Bool {
        !email.isEmpty && !password.isEmpty
    }

    private func handleAppleSignIn(_ result: Result<ASAuthorization, Error>) {
        switch result {
        case .failure:
            Task { @MainActor in
                viewModel.error = .socialAuthFailed("Apple")
            }
        case .success(let authorization):
            guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
                  let tokenData = credential.identityToken,
                  let idToken = String(data: tokenData, encoding: .utf8) else {
                Task { @MainActor in
                    viewModel.error = .socialAuthFailed("Apple")
                }
                return
            }

            Task {
                await viewModel.signInWithApple(token: idToken, nonce: currentNonce)
            }
        }
    }
    
}

#Preview {
    let mockRepository = MockAuthRepository()
    let mockAuthService = MockAuthService()
    let viewModel = AuthViewModel(authRepository: mockRepository, authService: mockAuthService)
    SignInView(viewModel: viewModel)
}

// MARK: - Mock for Preview

private actor MockAuthRepository: AuthRepository {
    func signUp(request: SignUpRequest) async throws -> AuthSession {
        throw AuthError.unknownError("Mock")
    }
    
    func signIn(request: SignInRequest) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    
    func signOut() async throws {}
    func currentSession() async throws -> AuthSession? { nil }
    func refreshSessionIfNeeded() async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func requestPasswordReset(email: String) async throws {}
    func resetPassword(token: String, newPassword: String) async throws {}
    func signInWithApple(token: String, nonce: String) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    nonisolated func validateEmail(_ email: String) throws {}
    nonisolated func validatePassword(_ password: String) throws {}
}

private actor MockAuthService: AuthService {
    func currentSession() async throws -> AuthSession? { nil }
    func signIn(email: String, password: String) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func signInWithApple(idToken: String, nonce: String?) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func signOut() async throws {}
    func refreshIfNeeded() async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
}
