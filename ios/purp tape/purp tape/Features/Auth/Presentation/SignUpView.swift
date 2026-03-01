import SwiftUI
import AuthenticationServices

public struct SignUpView: View {
    @StateObject private var viewModel: AuthViewModel
    @State private var email = ""
    @State private var password = ""
    @State private var confirmPassword = ""
    @State private var displayName = ""
    @State private var showPassword = false
    @State private var currentNonce = ""
    @State private var agreeToTerms = false
    @State private var signUpSuccess = false
    @Environment(\.dismiss) var dismiss
    @Environment(\.navigationManager) var navigationManager
    
    @FocusState private var focusedField: Field?
    
    enum Field {
        case displayName
        case email
        case password
        case confirmPassword
    }
    
    public init(viewModel: AuthViewModel) {
        self._viewModel = StateObject(wrappedValue: viewModel)
    }
    
    public var body: some View {
        ZStack {
            VStack(spacing: 0) {
                    // Header
                    VStack(alignment: .leading, spacing: Spacing.sm) {
                        Text("Create Account")
                            .displayLarge()
                            .foregroundColor(PurpTapeColors.text)
                        Text("Join the PurpTape community")
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
                            // Display Name Field
                            VStack(alignment: .leading, spacing: Spacing.sm) {
                                Label("Full Name", systemImage: "person.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                
                                TextField("Enter your full name", text: $displayName)
                                    .font(PurpTapeTypography.bodyMedium)
                                    .focused($focusedField, equals: .displayName)
                                    .purpTapeInputContainer()
                            }
                            
                            // Email Field
                            VStack(alignment: .leading, spacing: Spacing.sm) {
                                Label("Email", systemImage: "envelope.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                
                                TextField("Enter your email", text: $email)
                                    .textInputAutocapitalization(.never)
                                    .autocorrectionDisabled()
                                    .keyboardType(.emailAddress)
                                    .font(PurpTapeTypography.bodyMedium)
                                    .focused($focusedField, equals: .email)
                                    .purpTapeInputContainer()
                                
                                if !email.isEmpty && !viewModel.validateEmail(email) {
                                    Text("Invalid email address")
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
                                
                                // Password strength indicator
                                if !password.isEmpty {
                                    HStack(spacing: Spacing.xs) {
                                        ForEach(0..<3, id: \.self) { index in
                                            Capsule()
                                                .fill(passwordStrength > CGFloat(index) ? PurpTapeColors.success : PurpTapeColors.textSecondary.opacity(0.2))
                                                .frame(height: 4)
                                        }
                                    }
                                    
                                    Text(passwordStrengthText)
                                        .captionSmall()
                                        .foregroundColor(passwordStrength > 2 ? PurpTapeColors.success : PurpTapeColors.warning)
                                }
                            }
                            
                            // Confirm Password Field
                            VStack(alignment: .leading, spacing: Spacing.sm) {
                                Label("Confirm Password", systemImage: "lock.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                
                                SecureField("Confirm your password", text: $confirmPassword)
                                    .font(PurpTapeTypography.bodyMedium)
                                    .focused($focusedField, equals: .confirmPassword)
                                    .purpTapeInputContainer()
                                
                                if !confirmPassword.isEmpty && password != confirmPassword {
                                    Text("Passwords do not match")
                                        .captionSmall()
                                        .foregroundColor(PurpTapeColors.error)
                                }
                            }
                            
                            // Terms Agreement
                            HStack(spacing: Spacing.md) {
                                Image(systemName: agreeToTerms ? "checkmark.square.fill" : "square")
                                    .font(.system(size: 18, weight: .semibold))
                                    .foregroundColor(agreeToTerms ? PurpTapeColors.primary : PurpTapeColors.textSecondary)
                                    .onTapGesture {
                                        agreeToTerms.toggle()
                                    }
                                
                                VStack(alignment: .leading, spacing: Spacing.xs) {
                                    (Text("I agree to the") + Text(" Terms of Service")
                                        .fontWeight(.semibold)
                                        .foregroundColor(PurpTapeColors.primary))
                                        .captionLarge()
                                }
                            }
                            .contentShape(Rectangle())
                            .onTapGesture {
                                agreeToTerms.toggle()
                            }
                            
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
                    
                    // Sign Up Button
                    PurpTapePrimaryButton(
                        "Create Account",
                        isLoading: viewModel.isLoading,
                        isDisabled: !isFormValid
                    ) {
                        Task {
                            _ = SignUpRequest(
                                email: email,
                                password: password,
                                displayName: displayName
                            )
                        }
                    }
                    .paddingHorizontalLG()
                    .paddingBottomLG()
                    
                    // Social Auth
                    VStack(spacing: Spacing.md) {
                        Text("Or sign up with")
                            .bodyMedium()
                            .foregroundColor(PurpTapeColors.textSecondary)

                        SignInWithAppleButton { request in
                            let nonce = randomNonceString()
                            currentNonce = nonce
                            request.requestedScopes = [.fullName, .email]
                            request.nonce = sha256(nonce)
                        } onCompletion: { result in
                            handleAppleSignUp(result)
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
                    .padding(.horizontal, 24)
                    .padding(.bottom, 24)
                    
                    // Sign In Link
                    HStack(spacing: 4) {
                        Text("Already have an account?")
                            .font(.system(size: 14, weight: .regular))
                            .foregroundColor(.secondary)
                        Button("Sign In") {
                            dismiss()
                        }
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundColor(.blue)
                    }
                    .padding(.bottom, 40)
            }
        }
    }
    
    private var isFormValid: Bool {
        !displayName.isEmpty &&
        !email.isEmpty &&
        !password.isEmpty &&
        !confirmPassword.isEmpty &&
        password == confirmPassword &&
        viewModel.validatePassword(password) &&
        viewModel.validateEmail(email) &&
        agreeToTerms
    }
    
    private var passwordStrength: CGFloat {
        var strength: CGFloat = 0
        if password.count >= 8 { strength += 1 }
        if password.range(of: "[A-Z]", options: .regularExpression) != nil { strength += 1 }
        if password.range(of: "[0-9]", options: .regularExpression) != nil { strength += 1 }
        return strength
    }
    
    private var passwordStrengthText: String {
        switch passwordStrength {
        case 0...1:
            return "Weak"
        case 1...2:
            return "Fair"
        default:
            return "Strong"
        }
    }

    private func handleAppleSignUp(_ result: Result<ASAuthorization, Error>) {
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
    SignUpView(viewModel: viewModel)
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
