import SwiftUI

struct ProfileTabContent: View {
    @ObservedObject var authViewModel: AuthViewModel
    @Environment(\.navigationManager) var navigationManager

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.lg) {
            Text("Profile")
                .titleLarge()
                .foregroundColor(PurpTapeColors.text)

            if let session = authViewModel.currentSession {
                VStack(alignment: .leading, spacing: Spacing.sm) {
                    Text("User ID")
                        .labelSmall()
                        .foregroundColor(PurpTapeColors.textSecondary)
                    Text(session.userID.uuidString)
                        .monospaceMedium()
                        .foregroundColor(PurpTapeColors.primary)
                        .lineLimit(1)
                        .textSelection(.enabled)
                }
                .paddingMD()
                .background(PurpTapeColors.surface)
                .cornerRadiusMD()
            }

            Spacer()

            Button(role: .destructive, action: {
                Task {
                    await authViewModel.signOut()
                }
            }) {
                Label("Sign Out", systemImage: "arrow.backward.circle")
                    .frame(maxWidth: .infinity)
                    .font(PurpTapeTypography.labelLarge)
            }
            .buttonStyle(.bordered)
            .tint(PurpTapeColors.error)
        }
        .paddingLG()
        .purpTapeBackground()
    }
}
