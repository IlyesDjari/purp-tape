import SwiftUI

public struct ForgotPasswordView: View {
    @StateObject private var viewModel: AuthViewModel
    @State private var email = ""
    @State private var emailSent = false
    @Environment(\.dismiss) var dismiss
    @Environment(\.navigationManager) var navigationManager
    
    @FocusState private var emailFocused: Bool
    
    public init(viewModel: AuthViewModel) {
        self._viewModel = StateObject(wrappedValue: viewModel)
    }
    
    public var body: some View {
        ZStack {
            VStack(spacing: 0) {
                    // Header
                    VStack(alignment: .leading, spacing: Spacing.sm) {
                        Text("Reset Password")
                            .displayLarge()
                            .foregroundColor(PurpTapeColors.text)
                        Text("We'll send you instructions to reset your password")
                            .bodyLarge()
                            .foregroundColor(PurpTapeColors.textSecondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .paddingHorizontalLG()
                    .paddingTopLG()
                    .padding(.top, Spacing.xl)
                    .paddingBottomLG()
                    
                    if emailSent {
                        // Success State
                        VStack(spacing: Spacing.xl) {
                            VStack(spacing: Spacing.lg) {
                                Image(systemName: "envelope.open.fill")
                                    .font(.system(size: 48, weight: .semibold))
                                    .foregroundColor(PurpTapeColors.primary)
                                
                                VStack(spacing: Spacing.sm) {
                                    Text("Check Your Email")
                                        .headlineMedium()
                                        .foregroundColor(PurpTapeColors.text)
                                    Text("We've sent password reset instructions to \(email)")
                                        .bodyMedium()
                                        .foregroundColor(PurpTapeColors.textSecondary)
                                        .multilineTextAlignment(.center)
                                }
                            }
                            .paddingTopLG()
                            
                            Spacer()
                            
                            VStack(spacing: Spacing.md) {
                                Text("Didn't get the email?")
                                    .bodyMedium()
                                    .foregroundColor(PurpTapeColors.textSecondary)
                                
                                Button(action: {
                                    emailSent = false
                                    email = ""
                                }) {
                                    Text("Try Another Email")
                                        .font(PurpTapeTypography.labelLarge)
                                        .foregroundColor(PurpTapeColors.primary)
                                }
                            }
                        }
                        .paddingHorizontalLG()
                    } else {
                        // Input State
                        VStack(spacing: Spacing.xl) {
                            VStack(spacing: Spacing.sm) {
                                Label("Email Address", systemImage: "envelope.fill")
                                    .font(PurpTapeTypography.labelLarge)
                                    .foregroundColor(PurpTapeColors.text)
                                    .frame(maxWidth: CGFloat.infinity, alignment: .leading)
                                
                                TextField("Enter your email", text: $email)
                                    .textInputAutocapitalization(.never)
                                    .autocorrectionDisabled()
                                    .keyboardType(.emailAddress)
                                    .font(PurpTapeTypography.bodyMedium)
                                    .focused($emailFocused)
                                    .paddingMD()
                                    .background(PurpTapeColors.surface)
                                    .cornerRadiusMD()
                                    .border(PurpTapeColors.border, width: 1)
                                
                                if !email.isEmpty && !viewModel.validateEmail(email) {
                                    Text("Invalid email address")
                                        .captionSmall()
                                        .foregroundColor(PurpTapeColors.error)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                }
                            }
                            
                            VStack(spacing: Spacing.md) {
                                Text("Enter the email address associated with your account. We'll send you a link to reset your password.")
                                    .captionLarge()
                                    .foregroundColor(PurpTapeColors.textSecondary)
                                    .multilineTextAlignment(.leading)
                            }
                            
                            Spacer()
                        }
                        .paddingHorizontalLG()
                        .paddingVerticalLG()
                    }
                    
                    // Error Message
                    if let error = viewModel.error {
                        VStack(spacing: 8) {
                            HStack(spacing: 12) {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .foregroundColor(.red)
                                Text(error.errorDescription ?? "An error occurred")
                                    .font(.system(size: 14, weight: .regular))
                                Spacer()
                            }
                            .padding(12)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                        }
                        .padding(.horizontal, 24)
                        .padding(.bottom, 16)
                    }
                    
                    if !emailSent {
                        // Send Button
                        Button(action: {
                            Task {
                                await viewModel.requestPasswordReset(email: email)
                                emailSent = true
                            }
                        }) {
                            HStack {
                                if viewModel.isLoading {
                                    ProgressView()
                                        .foregroundColor(.white)
                                } else {
                                    Text("Send Reset Link")
                                        .font(PurpTapeTypography.labelLarge)
                                }
                            }
                            .frame(maxWidth: .infinity)
                            .paddingMD()
                            .background(isFormValid ? PurpTapeColors.primary : PurpTapeColors.textSecondary)
                            .foregroundColor(.white)
                            .cornerRadiusMD()
                        }
                        .disabled(!isFormValid || viewModel.isLoading)
                        .paddingHorizontalLG()
                        .paddingBottomLG()
                    }
                    
                    // Back Button
                    Button(action: { dismiss() }) {
                        Text("Back to Sign In")
                            .font(PurpTapeTypography.labelLarge)
                            .frame(maxWidth: .infinity)
                            .paddingMD()
                            .foregroundColor(PurpTapeColors.primary)
                            .cornerRadiusMD()
                    }
                    .paddingHorizontalLG()
                    .padding(.bottom, Spacing.xl)
        }
    }
    
    }
    
    private var isFormValid: Bool {
        !email.isEmpty && viewModel.validateEmail(email)
    }
}

// MARK: - Password Reset View

public struct PasswordResetView: View {
    @StateObject private var viewModel: AuthViewModel
    @State private var resetToken: String
    @State private var password = ""
    @State private var confirmPassword = ""
    @State private var showPassword = false
    @State private var resetSuccess = false
    @Environment(\.dismiss) var dismiss
    
    @FocusState private var focusedField: Field?
    
    enum Field {
        case password
        case confirmPassword
    }
    
    public init(viewModel: AuthViewModel, resetToken: String) {
        self._viewModel = StateObject(wrappedValue: viewModel)
        self.resetToken = resetToken
    }
    
    public var body: some View {
        ZStack {
            VStack(spacing: 0) {
                    // Header
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Create New Password")
                            .font(.system(size: 32, weight: .bold))
                        Text("Enter your new password below")
                            .font(.system(size: 16, weight: .regular))
                            .foregroundColor(.secondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 24)
                    .padding(.top, 40)
                    .padding(.bottom, 32)
                    
                    if resetSuccess {
                        // Success State
                        VStack(spacing: 24) {
                            VStack(spacing: 16) {
                                Image(systemName: "checkmark.circle.fill")
                                    .font(.system(size: 48, weight: .semibold))
                                    .foregroundColor(.green)
                                
                                VStack(spacing: 8) {
                                    Text("Password Reset Successful")
                                        .font(.system(size: 24, weight: .bold))
                                    Text("Your password has been updated successfully. You can now sign in with your new password.")
                                        .font(.system(size: 14, weight: .regular))
                                        .foregroundColor(.secondary)
                                        .multilineTextAlignment(.center)
                                }
                            }
                            .padding(.top, 40)
                            
                            Spacer()
                        }
                        .padding(.horizontal, 24)
                    } else {
                        // Form State
                        ScrollView {
                            VStack(spacing: 20) {
                                // Password Field
                                VStack(alignment: .leading, spacing: 8) {
                                    Label("New Password", systemImage: "lock.fill")
                                        .font(.system(size: 14, weight: .semibold))
                                        .foregroundColor(.primary)
                                    
                                    HStack {
                                        if showPassword {
                                            TextField("Enter new password", text: $password)
                                                .focused($focusedField, equals: .password)
                                        } else {
                                            SecureField("Enter new password", text: $password)
                                                .focused($focusedField, equals: .password)
                                        }
                                        
                                        Button(action: { showPassword.toggle() }) {
                                            Image(systemName: showPassword ? "eye.slash.fill" : "eye.fill")
                                                .foregroundColor(.secondary)
                                        }
                                    }
                                    .padding(12)
                                    .background(Color(.systemGray6))
                                    .cornerRadius(8)
                                    
                                    if !password.isEmpty {
                                        HStack(spacing: 4) {
                                            ForEach(0..<3, id: \.self) { index in
                                                Capsule()
                                                    .fill(passwordStrength > CGFloat(index) ? Color.green : Color.gray.opacity(0.2))
                                                    .frame(height: 4)
                                            }
                                        }
                                        
                                        Text(passwordStrengthText)
                                            .font(.system(size: 12, weight: .regular))
                                            .foregroundColor(passwordStrength > 2 ? Color.green : Color.orange)
                                    }
                                }
                                
                                // Confirm Password Field
                                VStack(alignment: .leading, spacing: 8) {
                                    Label("Confirm Password", systemImage: "lock.fill")
                                        .font(.system(size: 14, weight: .semibold))
                                        .foregroundColor(.primary)
                                    
                                    SecureField("Confirm new password", text: $confirmPassword)
                                        .focused($focusedField, equals: .confirmPassword)
                                        .padding(12)
                                        .background(Color(.systemGray6))
                                        .cornerRadius(8)
                                    
                                    if !confirmPassword.isEmpty && password != confirmPassword {
                                        Text("Passwords do not match")
                                            .font(.system(size: 12, weight: .regular))
                                            .foregroundColor(.red)
                                    }
                                }
                                
                                VStack(spacing: 12) {
                                    Text("Password requirements:")
                                        .font(.system(size: 13, weight: .semibold))
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                    
                                    HStack(spacing: 8) {
                                        Image(systemName: password.count >= 8 ? "checkmark.circle.fill" : "circle")
                                            .foregroundColor(password.count >= 8 ? .green : .gray)
                                        Text("At least 8 characters")
                                            .font(.system(size: 13, weight: .regular))
                                        Spacer()
                                    }
                                    
                                    HStack(spacing: 8) {
                                        Image(systemName: hasUppercase ? "checkmark.circle.fill" : "circle")
                                            .foregroundColor(hasUppercase ? .green : .gray)
                                        Text("One uppercase letter")
                                            .font(.system(size: 13, weight: .regular))
                                        Spacer()
                                    }
                                    
                                    HStack(spacing: 8) {
                                        Image(systemName: hasNumber ? "checkmark.circle.fill" : "circle")
                                            .foregroundColor(hasNumber ? .green : .gray)
                                        Text("One number")
                                            .font(.system(size: 13, weight: .regular))
                                        Spacer()
                                    }
                                }
                                .padding(12)
                                .background(Color(.systemGray6))
                                .cornerRadius(8)
                                
                                Spacer()
                            }
                            .padding(.horizontal, 24)
                            .padding(.vertical, 32)
                        }
                    }
                    
                    // Error Message
                    if let error = viewModel.error {
                        VStack(spacing: 8) {
                            HStack(spacing: 12) {
                                Image(systemName: "exclamationmark.circle.fill")
                                    .foregroundColor(.red)
                                Text(error.errorDescription ?? "An error occurred")
                                    .font(.system(size: 14, weight: .regular))
                                Spacer()
                            }
                            .padding(12)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                        }
                        .padding(.horizontal, 24)
                        .padding(.bottom, 16)
                    }
                    
                    if resetSuccess {
                        // Continue Button
                        Button(action: { dismiss() }) {
                            Text("Continue to Sign In")
                                .font(.system(size: 16, weight: .semibold))
                                .frame(maxWidth: .infinity)
                                .padding(12)
                                .background(Color.blue)
                                .foregroundColor(.white)
                                .cornerRadius(8)
                        }
                        .padding(.horizontal, 24)
                        .padding(.bottom, 16)
                    } else {
                        // Reset Button
                        Button(action: {
                            Task {
                                await viewModel.resetPassword(token: resetToken, newPassword: password)
                                resetSuccess = true
                            }
                        }) {
                            HStack {
                                if viewModel.isLoading {
                                    ProgressView()
                                        .foregroundColor(.white)
                                } else {
                                    Text("Reset Password")
                                        .font(.system(size: 16, weight: .semibold))
                                }
                            }
                            .frame(maxWidth: .infinity)
                            .padding(12)
                            .background(isFormValid ? Color.blue : Color.gray)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                        }
                        .disabled(!isFormValid || viewModel.isLoading)
                        .padding(.horizontal, 24)
                        .padding(.bottom, 16)
                    }
                    
                    // Back Button
                    Button(action: { dismiss() }) {
                        Text("Back")
                            .font(.system(size: 16, weight: .semibold))
                            .frame(maxWidth: .infinity)
                            .padding(12)
                            .foregroundColor(.blue)
                            .cornerRadius(8)
                    }
                    .padding(.horizontal, 24)
                    .padding(.bottom, 40)
        }
    }
    
    }
    
    private var isFormValid: Bool {
        !password.isEmpty &&
        !confirmPassword.isEmpty &&
        password == confirmPassword &&
        password.count >= 8 &&
        hasUppercase &&
        hasNumber
    }
    
    private var hasUppercase: Bool {
        password.range(of: "[A-Z]", options: .regularExpression) != nil
    }
    
    private var hasNumber: Bool {
        password.range(of: "[0-9]", options: .regularExpression) != nil
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
}

#Preview {
    let mockRepository = MockAuthRepository()
    let mockAuthService = MockAuthService()
    let viewModel = AuthViewModel(authRepository: mockRepository, authService: mockAuthService)
    ForgotPasswordView(viewModel: viewModel)
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
