import SwiftUI

struct MusicPlayerRow: View {
    let title: String
    let trailingText: String
    let isPlaying: Bool
    let isBuffering: Bool
    let progress: Double
    let onTogglePlayback: () -> Void

    var body: some View {
        HStack(spacing: Spacing.md) {
            Button(action: onTogglePlayback) {
                ZStack {
                    Circle()
                        .fill(PurpTapeColors.primary)
                        .frame(width: 40, height: 40)

                    if isBuffering {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                            .scaleEffect(0.8)
                    } else {
                        Image(systemName: isPlaying ? "pause.fill" : "play.fill")
                            .font(.system(size: 13, weight: .bold))
                            .foregroundStyle(.white)
                    }
                }
            }
            .buttonStyle(.plain)

            VStack(alignment: .leading, spacing: 6) {
                Text(title)
                    .font(PurpTapeTypography.bodyMedium)
                    .foregroundStyle(PurpTapeColors.text)
                    .lineLimit(1)

                GeometryReader { proxy in
                    ZStack(alignment: .leading) {
                        Capsule()
                            .fill(PurpTapeColors.textSecondary.opacity(0.2))
                        Capsule()
                            .fill(PurpTapeColors.primary)
                            .frame(width: max(2, proxy.size.width * CGFloat(min(max(progress, 0), 1))))
                    }
                }
                .frame(height: 4)
            }

            Spacer(minLength: Spacing.sm)

            Text(trailingText)
                .font(PurpTapeTypography.bodySmall)
                .foregroundStyle(PurpTapeColors.textSecondary)
        }
        .padding(Spacing.md)
        .background(Color(.systemGray6))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
