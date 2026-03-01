import SwiftUI

public struct PurpTapeInteractiveButtonStyle: ButtonStyle {
    public init() {}

    public func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.98 : 1.0)
            .animation(.spring(response: 0.25, dampingFraction: 0.8), value: configuration.isPressed)
    }
}

public struct PurpTapePrimaryButton: View {
    let title: String
    let isLoading: Bool
    let isDisabled: Bool
    let action: () -> Void

    public init(
        _ title: String,
        isLoading: Bool = false,
        isDisabled: Bool = false,
        action: @escaping () -> Void
    ) {
        self.title = title
        self.isLoading = isLoading
        self.isDisabled = isDisabled
        self.action = action
    }

    public var body: some View {
        Button(action: action) {
            HStack {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Text(title)
                        .font(PurpTapeTypography.labelLarge)
                }
            }
            .frame(maxWidth: .infinity)
            .paddingMD()
            .background(isDisabled ? PurpTapeColors.textSecondary.opacity(0.6) : PurpTapeColors.primary)
            .foregroundColor(.white)
            .cornerRadiusMD()
            .shadow(color: PurpTapeColors.shadowLight, radius: 8, x: 0, y: 3)
        }
        .buttonStyle(PurpTapeInteractiveButtonStyle())
        .disabled(isDisabled || isLoading)
    }
}

public struct PurpTapeSecondaryButton: View {
    let title: String
    let icon: String?
    let isDisabled: Bool
    let action: () -> Void

    public init(
        _ title: String,
        icon: String? = nil,
        isDisabled: Bool = false,
        action: @escaping () -> Void
    ) {
        self.title = title
        self.icon = icon
        self.isDisabled = isDisabled
        self.action = action
    }

    public var body: some View {
        Button(action: action) {
            HStack(spacing: Spacing.md) {
                if let icon {
                    Image(systemName: icon)
                        .font(.system(size: 18, weight: .semibold))
                }
                Text(title)
                    .font(PurpTapeTypography.labelLarge)
            }
            .frame(maxWidth: .infinity)
            .paddingMD()
            .background(PurpTapeColors.surface)
            .foregroundColor(PurpTapeColors.text)
            .overlay(
                RoundedRectangle(cornerRadius: CornerRadius.md)
                    .stroke(PurpTapeColors.border, lineWidth: 1)
            )
            .cornerRadiusMD()
            .shadow(color: PurpTapeColors.shadowLight, radius: 6, x: 0, y: 2)
        }
        .buttonStyle(PurpTapeInteractiveButtonStyle())
        .disabled(isDisabled)
    }
}
