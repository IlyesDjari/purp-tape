import SwiftUI

struct SwipeToDeleteTrackRow: View {
    let track: Track
    let isPlaying: Bool
    let isBuffering: Bool
    let progress: Double
    let isDeleting: Bool
    let onTogglePlayback: () -> Void
    let onDelete: () -> Void

    @State private var offsetX: CGFloat = 0
    @GestureState private var dragTranslation: CGFloat = 0

    private let revealWidth: CGFloat = 86
    private let deleteThreshold: CGFloat = 66

    var body: some View {
        ZStack(alignment: .trailing) {
            deleteActionBackground

            TrackCard(
                track: track,
                isPlaying: isPlaying,
                isBuffering: isBuffering,
                progress: progress,
                onTogglePlayback: onTogglePlayback
            )
            .offset(x: clampedOffset)
            .simultaneousGesture(dragGesture)
            .animation(.spring(response: 0.28, dampingFraction: 0.86), value: offsetX)
        }
        .contentShape(Rectangle())
        .onTapGesture {
            if offsetX != 0 {
                withAnimation(.spring(response: 0.25, dampingFraction: 0.9)) {
                    offsetX = 0
                }
            }
        }
    }

    private var clampedOffset: CGFloat {
        let raw = offsetX + dragTranslation
        return min(0, max(-revealWidth, raw))
    }

    private var dragGesture: some Gesture {
        DragGesture(minimumDistance: 8)
            .updating($dragTranslation) { value, state, _ in
                state = value.translation.width
            }
            .onEnded { value in
                let projected = offsetX + value.translation.width
                withAnimation(.spring(response: 0.24, dampingFraction: 0.9)) {
                    if projected <= -deleteThreshold {
                        offsetX = -revealWidth
                    } else {
                        offsetX = 0
                    }
                }
            }
    }

    private var deleteActionBackground: some View {
        HStack {
            Spacer()
            Button {
                onDelete()
            } label: {
                ZStack {
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .fill(Color.red)

                    if isDeleting {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                    } else {
                        Image(systemName: "trash.fill")
                            .font(.system(size: 16, weight: .bold))
                            .foregroundStyle(.white)
                    }
                }
                .frame(width: revealWidth - 10, height: 56)
            }
            .buttonStyle(.plain)
            .disabled(isDeleting)
            .padding(.trailing, 4)
        }
    }
}
