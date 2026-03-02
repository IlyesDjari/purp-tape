import AVFoundation
import Foundation

@MainActor
final class TrackPlaybackController: ObservableObject {
    @Published private(set) var currentTrackID: UUID?
    @Published private(set) var isPlaying = false
    @Published private(set) var isBuffering = false
    @Published private(set) var progress: Double = 0
    @Published private(set) var currentTimeSeconds: Double = 0
    @Published private(set) var durationSeconds: Double = 0
    @Published private(set) var playbackErrorMessage: String?

    private var player: AVPlayer?
    private var stateTimer: Timer?
    private let logger = DebugLogger(category: "player.track")

    func isActive(trackID: UUID) -> Bool {
        currentTrackID == trackID
    }

    func toggleCurrentTrackPlayback() {
        guard let player else { return }

        let shouldPause = player.timeControlStatus == .playing ||
            player.timeControlStatus == .waitingToPlayAtSpecifiedRate ||
            isBuffering

        if shouldPause {
            pause()
        } else {
            resume()
        }
    }

    func play(trackID: UUID, streamURL: URL) {
        logger.info("Starting playback for track \(trackID.uuidString)")
        logger.network("Signed playback URL host=\(streamURL.host() ?? "unknown")")

        if currentTrackID != trackID {
            teardownPlayer(resetTrack: false)
            currentTrackID = trackID
            player = AVPlayer(url: streamURL)
        }

        playbackErrorMessage = nil
        resume()
    }

    func stop() {
        logger.info("Stopping playback")
        teardownPlayer(resetTrack: true)
    }

    private func resume() {
        guard let player else { return }
        logger.debug("Resuming playback")

        if durationSeconds > 0,
           currentTimeSeconds >= (durationSeconds - 0.2) {
            player.seek(to: .zero)
            currentTimeSeconds = 0
            progress = 0
        }

        isBuffering = true
        ensureStateTimer()
        player.play()
        updatePlaybackState()
    }

    private func pause() {
        logger.debug("Pausing playback")
        player?.pause()
        invalidateStateTimer()
        isPlaying = false
        isBuffering = false
    }

    private func ensureStateTimer() {
        guard stateTimer == nil else { return }
        stateTimer = Timer.scheduledTimer(withTimeInterval: 0.25, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.updatePlaybackState()
            }
        }
        RunLoop.main.add(stateTimer!, forMode: .common)
    }

    private func invalidateStateTimer() {
        stateTimer?.invalidate()
        stateTimer = nil
    }

    private func updatePlaybackState() {
        guard let player else {
            isPlaying = false
            isBuffering = false
            return
        }

        let timeStatus = player.timeControlStatus
        isPlaying = timeStatus == .playing
        isBuffering = timeStatus == .waitingToPlayAtSpecifiedRate

        let time = player.currentTime().seconds
        if time.isFinite && time >= 0 {
            currentTimeSeconds = time
        }

        let duration = player.currentItem?.duration.seconds ?? 0
        if duration.isFinite, duration > 0 {
            durationSeconds = duration
            progress = min(1, max(0, currentTimeSeconds / duration))
        } else {
            progress = 0
        }

        if let item = player.currentItem {
            if item.status == .failed {
                let itemError = item.error?.localizedDescription ?? "Playback failed"
                logger.error("Player item failed: \(itemError)")
                playbackErrorMessage = itemError
                pause()
            } else if let errorLogEvents = item.errorLog()?.events, let lastEvent = errorLogEvents.last,
                      let errorComment = lastEvent.errorComment, !errorComment.isEmpty {
                logger.warning("Player error event: \(errorComment)")
                playbackErrorMessage = errorComment
            }
        }
    }

    private func teardownPlayer(resetTrack: Bool) {
        invalidateStateTimer()
        player?.pause()
        player = nil

        isPlaying = false
        isBuffering = false
        progress = 0
        currentTimeSeconds = 0
        durationSeconds = 0
        playbackErrorMessage = nil

        if resetTrack {
            currentTrackID = nil
        }
    }
}
