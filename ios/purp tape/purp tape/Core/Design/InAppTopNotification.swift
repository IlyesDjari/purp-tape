import SwiftUI

struct InAppTopNotification: Equatable {
    enum Kind: Equatable {
        case success
        case error
        case info
    }

    let id = UUID()
    let kind: Kind
    let message: String

    static func success(_ message: String) -> InAppTopNotification {
        InAppTopNotification(kind: .success, message: message)
    }

    static func error(_ message: String) -> InAppTopNotification {
        InAppTopNotification(kind: .error, message: message)
    }

    static func info(_ message: String) -> InAppTopNotification {
        InAppTopNotification(kind: .info, message: message)
    }
}

private struct InAppTopNotificationBanner: View {
    let notification: InAppTopNotification

    private var icon: String {
        switch notification.kind {
        case .success:
            return "checkmark.circle.fill"
        case .error:
            return "xmark.octagon.fill"
        case .info:
            return "info.circle.fill"
        }
    }

    private var iconColor: Color {
        switch notification.kind {
        case .success:
            return PurpTapeColors.success
        case .error:
            return PurpTapeColors.error
        case .info:
            return PurpTapeColors.primary
        }
    }

    var body: some View {
        HStack(spacing: Spacing.sm) {
            Image(systemName: icon)
                .foregroundColor(iconColor)
            Text(notification.message)
                .font(PurpTapeTypography.labelLarge)
                .foregroundColor(PurpTapeColors.text)
                .lineLimit(2)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, Spacing.lg)
        .padding(.vertical, Spacing.md)
        .background(PurpTapeColors.surface)
        .clipShape(Capsule())
        .shadow(color: PurpTapeColors.shadowLight, radius: 12, x: 0, y: 6)
        .overlay(
            Capsule()
                .stroke(PurpTapeColors.border, lineWidth: 1)
        )
    }
}

private struct InAppTopNotificationModifier: ViewModifier {
    @Binding var notification: InAppTopNotification?
    let duration: Double
    @State private var dismissTask: Task<Void, Never>?

    func body(content: Content) -> some View {
        content
            .overlay(alignment: .top) {
                if let notification {
                    InAppTopNotificationBanner(notification: notification)
                        .padding(.top, 8)
                        .padding(.horizontal, Spacing.lg)
                        .transition(.move(edge: .top).combined(with: .opacity))
                }
            }
            .onChange(of: notification?.id) { _, newID in
                dismissTask?.cancel()
                guard newID != nil else { return }
                dismissTask = Task {
                    try? await Task.sleep(nanoseconds: UInt64(duration * 1_000_000_000))
                    guard !Task.isCancelled else { return }
                    await MainActor.run {
                        withAnimation(.spring(response: 0.28, dampingFraction: 0.9)) {
                            notification = nil
                        }
                    }
                }
            }
            .onDisappear {
                dismissTask?.cancel()
            }
    }
}

extension View {
    func inAppTopNotification(notification: Binding<InAppTopNotification?>, duration: Double = 3) -> some View {
        modifier(InAppTopNotificationModifier(notification: notification, duration: duration))
    }
}
