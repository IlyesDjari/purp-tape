import SwiftUI

struct TrackCard: View {
    let track: Track
    let isPlaying: Bool
    let isBuffering: Bool
    let progress: Double
    let onTogglePlayback: () -> Void

    init(
        track: Track,
        isPlaying: Bool = false,
        isBuffering: Bool = false,
        progress: Double = 0,
        onTogglePlayback: @escaping () -> Void = {}
    ) {
        self.track = track
        self.isPlaying = isPlaying
        self.isBuffering = isBuffering
        self.progress = progress
        self.onTogglePlayback = onTogglePlayback
    }
    
    var body: some View {
        MusicPlayerRow(
            title: track.title,
            trailingText: formatDuration(track.durationSeconds),
            isPlaying: isPlaying,
            isBuffering: isBuffering,
            progress: progress,
            onTogglePlayback: onTogglePlayback
        )
    }
    
    private func formatDuration(_ seconds: Int) -> String {
        let minutes = seconds / 60
        let secs = seconds % 60
        return String(format: "%d:%02d", minutes, secs)
    }
}

#Preview {
    TrackCard(
        track: Track(
            id: UUID(),
            projectID: UUID(),
            title: "Midnight Groove",
            durationSeconds: 245
        )
    )
    .padding()
}
