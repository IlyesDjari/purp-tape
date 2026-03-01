import SwiftUI
import CryptoKit

// MARK: - Protected Route Guard

public struct ProtectedRouteView<Content: View>: View {
    @ObservedObject var authViewModel: AuthViewModel
    let content: (AuthSession) -> Content
    
    public init(
        authViewModel: AuthViewModel,
        @ViewBuilder content: @escaping (AuthSession) -> Content
    ) {
        self.authViewModel = authViewModel
        self.content = content
    }
    
    public var body: some View {
        if let session = authViewModel.currentSession, authViewModel.isAuthenticated {
            content(session)
        } else {
            VStack(spacing: Spacing.lg) {
                Image(systemName: "lock.fill")
                    .font(.system(size: 48, weight: .semibold))
                    .foregroundColor(PurpTapeColors.textSecondary)
                
                Text("Access Denied")
                    .titleLarge()
                    .foregroundColor(PurpTapeColors.text)
                
                Text("You need to be signed in to access this content")
                    .bodyMedium()
                    .foregroundColor(PurpTapeColors.textSecondary)
                    .multilineTextAlignment(.center)
            }
            .paddingLG()
        }
    }
}

// MARK: - Session Recovery View

public struct SessionRecoveryView: View {
    @ObservedObject var authViewModel: AuthViewModel
    
    public var body: some View {
        VStack(spacing: Spacing.lg) {
            VStack(spacing: Spacing.sm) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .font(.system(size: 40, weight: .semibold))
                    .foregroundColor(PurpTapeColors.warning)
                
                Text("Session Expired")
                    .headlineSmall()
                    .foregroundColor(PurpTapeColors.text)
                
                Text("Your session has expired. Please sign in again.")
                    .bodyMedium()
                    .foregroundColor(PurpTapeColors.textSecondary)
                    .multilineTextAlignment(.center)
            }
            .paddingVerticalXL()
            
            Button(action: {
                Task {
                    await authViewModel.signOut()
                }
            }) {
                Text("Sign In Again")
                    .font(PurpTapeTypography.labelLarge)
                    .frame(maxWidth: .infinity)
                    .paddingMD()
                    .background(PurpTapeColors.primary)
                    .foregroundColor(.white)
                    .cornerRadiusMD()
            }
        }
        .paddingLG()
        .background(PurpTapeColors.surface)
        .cornerRadiusLG()
    }
}

// MARK: - Auth State Indicator

public struct AuthStateIndicator: View {
    @ObservedObject var authViewModel: AuthViewModel
    
    public var body: some View {
        HStack(spacing: 8) {
            Group {
                if authViewModel.isLoading {
                    ProgressView()
                } else if authViewModel.currentSession != nil {
                    HStack(spacing: Spacing.xs) {
                        Circle()
                            .fill(PurpTapeColors.success)
                            .frame(width: 8, height: 8)
                        Text("Authenticated")
                    }
                } else {
                    HStack(spacing: Spacing.xs) {
                        Circle()
                            .fill(PurpTapeColors.textSecondary)
                            .frame(width: 8, height: 8)
                        Text("Not Authenticated")
                    }
                }
            }
            .font(PurpTapeTypography.labelSmall)
            
            Spacer()
            
            if let session = authViewModel.currentSession {
                Menu {
                    Section("User ID") {
                        Text(session.userID.uuidString)
                            .textSelection(.enabled)
                    }
                    
                    Section("Token Status") {
                        Text(session.isExpired ? "Expired" : "Valid")
                    }
                    
                    Section(header: Text("Expires At")) {
                        Text(session.expiresAt.formatted(date: .abbreviated, time: .standard))
                    }
                    
                    Button(role: .destructive, action: {
                        Task {
                            await authViewModel.signOut()
                        }
                    }) {
                        Label("Sign Out", systemImage: "arrow.backward.circle")
                    }
                } label: {
                    Image(systemName: "info.circle")
                        .foregroundColor(PurpTapeColors.text)
                }
                .frame(width: 24, height: 24)
            }
        }
        .paddingMD()
        .background(PurpTapeColors.surface)
        .cornerRadiusMD()
    }
}

// MARK: - Auth Error Banner

public struct AuthErrorBanner: View {
    let error: AuthError
    let action: (() -> Void)?
    
    public init(error: AuthError, action: (() -> Void)? = nil) {
        self.error = error
        self.action = action
    }
    
    public var body: some View {
        VStack(spacing: Spacing.md) {
            HStack(spacing: Spacing.md) {
                Image(systemName: "exclamationmark.circle.fill")
                    .foregroundColor(PurpTapeColors.error)
                    .font(.system(size: 16, weight: .semibold))
                
                VStack(alignment: .leading, spacing: Spacing.xs) {
                    Text("Error")
                        .labelSmall()
                        .foregroundColor(PurpTapeColors.error)
                    Text(error.errorDescription ?? "Unknown error")
                        .captionLarge()
                        .foregroundColor(PurpTapeColors.text)
                }
                
                Spacer()
            }
            
            if let recoverySuggestion = error.recoverySuggestion {
                Text(recoverySuggestion)
                    .captionSmall()
                    .foregroundColor(PurpTapeColors.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            
            if let action = action {
                Button(action: action) {
                    Text("Try Again")
                        .captionLarge()
                        .fontWeight(.semibold)
                        .frame(maxWidth: .infinity)
                        .paddingSM()
                        .background(PurpTapeColors.error.opacity(0.1))
                        .foregroundColor(PurpTapeColors.error)
                        .cornerRadiusSM()
                }
            }
        }
        .paddingMD()
        .background(PurpTapeColors.error.opacity(0.1))
        .cornerRadiusMD()
        .overlay(
            RoundedRectangle(cornerRadius: CornerRadius.md)
                .stroke(PurpTapeColors.error.opacity(0.2), lineWidth: 1)
        )
    }
}

// MARK: - Loading State

public struct AuthLoadingView: View {
    let message: String
    
    public init(message: String = "Loading...") {
        self.message = message
    }
    
    public var body: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.2, anchor: .center)
            
            Text(message)
                .font(.system(size: 14, weight: .regular))
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}

// MARK: - View Extensions

public extension View {
    /// Apply this to protect a view from unauthorized access
    func withAuthProtection(
        _ authViewModel: AuthViewModel
    ) -> some View {
        ProtectedRouteView(authViewModel: authViewModel) { session in
            self
        }
    }
    
    /// Show error banner if auth has an error
    func withAuthErrorHandling(
        _ authViewModel: AuthViewModel
    ) -> some View {
        VStack(spacing: 0) {
            if let error = authViewModel.error {
                AuthErrorBanner(error: error, action: {
                    Task {
                        await authViewModel.refreshSessionIfNeeded()
                    }
                })
                .padding()
            }
            
            self
        }
    }
    
    /// Show loading overlay when auth is processing
    func withAuthLoadingOverlay(
        _ authViewModel: AuthViewModel
    ) -> some View {
        ZStack {
            self
            
            if authViewModel.isLoading {
                Color.black.opacity(0.3)
                    .ignoresSafeArea()
                AuthLoadingView(message: "Processing...")
            }
        }
    }
}

func randomNonceString(length: Int = 32) -> String {
    precondition(length > 0)
    let charset: [Character] = Array("0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._")
    var result = ""
    var remainingLength = length

    while remainingLength > 0 {
        var randoms: [UInt8] = Array(repeating: 0, count: 16)
        let status = SecRandomCopyBytes(kSecRandomDefault, randoms.count, &randoms)
        if status != errSecSuccess {
            break
        }

        randoms.forEach { random in
            if remainingLength == 0 {
                return
            }

            if random < charset.count {
                result.append(charset[Int(random)])
                remainingLength -= 1
            }
        }
    }

    return result
}

func sha256(_ input: String) -> String {
    let inputData = Data(input.utf8)
    let hashedData = SHA256.hash(data: inputData)
    return hashedData.map { String(format: "%02x", $0) }.joined()
}

// MARK: - Disclosure Helpers

public struct AuthDebugInfo: View {
    @ObservedObject var authViewModel: AuthViewModel
    
    public var body: some View {
        DisclosureGroup("Auth Debug Info") {
            VStack(spacing: 12) {
                Divider()
                
                InfoRow(label: "Is Authenticated", value: authViewModel.isAuthenticated ? "Yes" : "No")
                InfoRow(label: "Is Loading", value: authViewModel.isLoading ? "Yes" : "No")
                
                if let session = authViewModel.currentSession {
                    Divider()
                    
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Session Info")
                            .font(.system(size: 12, weight: .semibold))
                        
                        InfoRow(label: "User ID", value: session.userID.uuidString)
                        InfoRow(label: "Expires At", value: session.expiresAt.formatted())
                        InfoRow(label: "Is Expired", value: session.isExpired ? "Yes" : "No")
                        
                        Text("Access Token (first 20 chars)")
                            .font(.system(size: 11, weight: .regular))
                            .foregroundColor(.secondary)
                        Text(String(session.accessToken.prefix(20)) + "...")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundColor(.blue)
                            .textSelection(.enabled)
                    }
                }
                
                if let error = authViewModel.error {
                    Divider()
                    
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Last Error")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundColor(.red)
                        
                        Text(error.errorDescription ?? "Unknown")
                            .font(.system(size: 11, design: .monospaced))
                            .foregroundColor(.primary)
                        
                        if let suggestion = error.recoverySuggestion {
                            Text(suggestion)
                                .font(.system(size: 10, weight: .regular))
                                .foregroundColor(.secondary)
                        }
                    }
                }
            }
            .monospaced()
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.vertical, 8)
        }
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }
}

private struct InfoRow: View {
    let label: String
    let value: String
    
    var body: some View {
        HStack {
            Text(label)
                .font(.system(size: 11, weight: .semibold))
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .font(.system(size: 11, weight: .semibold))
                .foregroundColor(.primary)
                .lineLimit(1)
        }
    }
}
