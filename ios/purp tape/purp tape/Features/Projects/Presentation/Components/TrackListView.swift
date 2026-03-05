import SwiftUI
import Foundation

struct TrackListView: View {
    let projectID: UUID
    let apiClient: APIClient
    let maxNonScrollableHeight: CGFloat
    let onOverflowChange: ((Bool) -> Void)?
    let playbackController: TrackPlaybackController
    let onActiveTrackChange: ((Track?) -> Void)?
    @State private var tracks: [Track] = []
    @State private var showAddTrackSheet = false
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var contentHeight: CGFloat = 0
    @State private var deletingTrackIDs: Set<UUID> = []
    
    private let dataStore: TracksDataStore
    private let logger = DebugLogger(category: "ui.track-list")
    
    init(
        projectID: UUID,
        apiClient: APIClient,
        maxNonScrollableHeight: CGFloat = 340,
        onOverflowChange: ((Bool) -> Void)? = nil,
        playbackController: TrackPlaybackController,
        onActiveTrackChange: ((Track?) -> Void)? = nil
    ) {
        self.projectID = projectID
        self.apiClient = apiClient
        self.maxNonScrollableHeight = maxNonScrollableHeight
        self.onOverflowChange = onOverflowChange
        self.playbackController = playbackController
        self.onActiveTrackChange = onActiveTrackChange
        self.dataStore = URLSessionTracksDataStore(apiClient: apiClient)
    }
    
    var body: some View {
        trackListBody
        .animation(.spring(response: 0.52, dampingFraction: 0.8), value: isOverflowing)
        .sheet(isPresented: $showAddTrackSheet) {
            AddTrackSheet(
                isPresented: $showAddTrackSheet,
                projectID: projectID,
                apiClient: apiClient,
                onTrackAdded: { newTrack in
                    tracks.insert(newTrack, at: 0)
                }
            )
        }
        .task {
            await loadTracks()
        }
        .onChange(of: isOverflowing) { _, newValue in
            onOverflowChange?(newValue)
        }
        .onAppear {
            onOverflowChange?(isOverflowing)
        }
        .onChange(of: playbackController.playbackErrorMessage) { _, newValue in
            if let newValue, !newValue.isEmpty {
                errorMessage = "Playback error: \(newValue)"
                onActiveTrackChange?(nil)
            }
        }
    }

    private var trackListBody: some View {
        VStack(alignment: .leading, spacing: Spacing.lg) {
            if tracks.isEmpty {
                VStack(spacing: Spacing.md) {
                    Image(systemName: "music.note")
                        .font(.system(size: 40))
                        .foregroundStyle(PurpTapeColors.textSecondary)
                    Text("No tracks yet")
                        .font(PurpTapeTypography.bodyMedium)
                        .foregroundStyle(PurpTapeColors.textSecondary)
                    Text("Add your first track to get started")
                        .font(PurpTapeTypography.bodySmall)
                        .foregroundStyle(PurpTapeColors.textSecondary)
                    Button(action: { showAddTrackSheet = true }) {
                        HStack {
                            Image(systemName: "plus.circle.fill")
                                .font(.system(size: 20))
                            Text("Add Track")
                                .font(PurpTapeTypography.bodyMedium)
                        }
                        .foregroundStyle(.white)
                        .padding(.vertical, 10)
                        .padding(.horizontal, 24)
                        .background(PurpTapeColors.primary)
                        .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                    .padding(.top, Spacing.md)
                }
                .frame(maxWidth: .infinity)
                .padding(Spacing.xl)
            } else {
                VStack(spacing: Spacing.md) {
                    ForEach(tracks, id: \.id) { track in
                        SwipeToDeleteTrackRow(
                            track: track,
                            isPlaying: playbackController.isActive(trackID: track.id) && playbackController.isPlaying,
                            isBuffering: playbackController.isActive(trackID: track.id) && playbackController.isBuffering,
                            progress: playbackController.isActive(trackID: track.id) ? playbackController.progress : 0,
                            isDeleting: deletingTrackIDs.contains(track.id),
                            onTogglePlayback: {
                                togglePlayback(for: track)
                            },
                            onDelete: {
                                deleteTrack(track)
                            }
                        )
                    }
                }
                Button(action: { showAddTrackSheet = true }) {
                    HStack(alignment: .center) {
                        Image(systemName: "plus.circle.fill")
                            .font(.system(size: 20))
                        Text("Add Track")
                            .font(PurpTapeTypography.bodyMedium)
                    }
                    .foregroundStyle(.white)
                    .padding(.vertical, 10)
                    .padding(.horizontal, 24)
                    .background(PurpTapeColors.primary)
                    .clipShape(Capsule())
                }
                .buttonStyle(.plain)
                .padding(.top, Spacing.md)
            }

            if let errorMessage, !errorMessage.isEmpty {
                HStack(alignment: .top, spacing: Spacing.sm) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                    Text(errorMessage)
                        .font(PurpTapeTypography.bodySmall)
                        .foregroundStyle(PurpTapeColors.text)
                }
                .padding(Spacing.md)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.orange.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            }
        }
        .background(
            GeometryReader { proxy in
                Color.clear
                    .preference(key: TrackListContentHeightKey.self, value: proxy.size.height)
            }
        )
        .onPreferenceChange(TrackListContentHeightKey.self) { newHeight in
            Task { @MainActor in
                contentHeight = newHeight
            }
        }
    }

    private var isOverflowing: Bool {
        contentHeight > maxNonScrollableHeight
    }
    
    private func loadTracks() async {
        isLoading = true
        
        do {
            // APIClient handles auth internally
            tracks = try await dataStore.fetchTracks(projectID: projectID, accessToken: "")

            if let currentTrackID = playbackController.currentTrackID,
               tracks.contains(where: { $0.id == currentTrackID }) == false {
                playbackController.stop()
                onActiveTrackChange?(nil)
            }

            errorMessage = nil
        } catch {
            errorMessage = "Failed to load tracks: \(error.localizedDescription)"
        }
        
        isLoading = false
    }
    
    private func togglePlayback(for track: Track) {
        errorMessage = nil
        logger.debug("Play toggle tapped track=\(track.id.uuidString) title=\(track.title)")

        if playbackController.isActive(trackID: track.id) {
            playbackController.toggleCurrentTrackPlayback()
            onActiveTrackChange?(track)
            return
        }
        
        Task {
            do {
                logger.network("Requesting signed playback URL for track \(track.id.uuidString)")
                let streamURL = try await dataStore.fetchSignedPlaybackURL(trackID: track.id, accessToken: "")
                await MainActor.run {
                    logger.success("Signed URL received for track \(track.id.uuidString)")
                    playbackController.play(trackID: track.id, streamURL: streamURL)
                    onActiveTrackChange?(track)
                }
            } catch {
                await MainActor.run {
                    let message = error.localizedDescription
                    if message.localizedCaseInsensitiveContains("no versions available") {
                        errorMessage = "This track has no uploaded version yet. Upload a new track or re-upload this one."
                    } else {
                        errorMessage = "Failed to play track: \(message)"
                    }
                    logger.error("Playback start failed for track \(track.id.uuidString): \(message)")
                }
            }
        }
    }

    private func deleteTrack(_ track: Track) {
        guard deletingTrackIDs.contains(track.id) == false else { return }
        deletingTrackIDs.insert(track.id)

        Task {
            do {
                try await dataStore.deleteTrack(trackID: track.id, accessToken: "")
                await MainActor.run {
                    withAnimation(.spring(response: 0.28, dampingFraction: 0.86)) {
                        tracks.removeAll { $0.id == track.id }
                    }

                    if playbackController.isActive(trackID: track.id) {
                        playbackController.stop()
                        onActiveTrackChange?(nil)
                    }

                    deletingTrackIDs.remove(track.id)
                }
            } catch {
                await MainActor.run {
                    deletingTrackIDs.remove(track.id)
                    errorMessage = "Failed to delete track: \(error.localizedDescription)"
                    logger.error("Delete track failed \(track.id.uuidString): \(error.localizedDescription)")
                }
            }
        }
    }
}

private struct TrackListContentHeightKey: PreferenceKey {
    static let defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}

#Preview {
    let playbackController = TrackPlaybackController()

    TrackListView(
        projectID: UUID(),
        apiClient: MockAPIClient(),
        playbackController: playbackController
    )
    .padding()
}

// Mock for preview
private actor MockAPIClient: APIClient {
    func send<T>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T where T : Decodable, T : Sendable {
        throw APIClientError.invalidResponse
    }
    
    func upload<T>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T where T : Decodable, T : Sendable {
        throw APIClientError.invalidResponse
    }
}
