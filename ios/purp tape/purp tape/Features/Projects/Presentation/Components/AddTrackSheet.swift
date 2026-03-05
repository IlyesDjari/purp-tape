import SwiftUI
import AVFoundation
import UniformTypeIdentifiers

struct AddTrackSheet: View {
    @Binding var isPresented: Bool
    let projectID: UUID
    let apiClient: APIClient
    let onTrackAdded: (Track) -> Void
    
    @State private var selectedAudioURL: URL?
    @State private var audioData: Data?
    @State private var selectedAudioFileName: String = "track.m4a"
    @State private var selectedAudioMimeType: String?
    @State private var trackTitle: String = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    
    private let dataStore: TracksDataStore
    
    init(
        isPresented: Binding<Bool>,
        projectID: UUID,
        apiClient: APIClient,
        onTrackAdded: @escaping (Track) -> Void
    ) {
        self._isPresented = isPresented
        self.projectID = projectID
        self.apiClient = apiClient
        self.onTrackAdded = onTrackAdded
        self.dataStore = URLSessionTracksDataStore(apiClient: apiClient)
    }
    
    var body: some View {
        NavigationStack {
            VStack(spacing: Spacing.lg) {
                // File Picker Section
                VStack(spacing: Spacing.md) {
                    if selectedAudioURL != nil {
                        VStack(spacing: Spacing.sm) {
                            Image(systemName: "checkmark.circle.fill")
                                .font(.system(size: 40))
                                .foregroundStyle(.green)
                            
                            Text("File selected")
                                .font(PurpTapeTypography.bodyMedium)
                                .foregroundStyle(PurpTapeColors.text)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(Spacing.lg)
                    } else {
                        FilePickerButton { url in
                            selectedAudioURL = url
                            do {
                                audioData = try readData(from: url)
                                selectedAudioFileName = url.lastPathComponent
                                selectedAudioMimeType = mimeType(for: url)
                                extractTrackMetadata(from: url)
                            } catch {
                                errorMessage = "Failed to read audio file: \(error.localizedDescription)"
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding(Spacing.lg)
                    }
                }
                .background(Color(.systemGray6))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                
                // Track Title Input
                VStack(alignment: .leading, spacing: Spacing.sm) {
                    Text("Track Title")
                        .font(PurpTapeTypography.labelMedium)
                        .foregroundStyle(PurpTapeColors.text)
                    
                    TextField("Enter track title", text: $trackTitle)
                        .textFieldStyle(.roundedBorder)
                        .font(PurpTapeTypography.bodyMedium)
                }
                
                // Error Message
                if let errorMessage {
                    VStack(alignment: .leading, spacing: Spacing.sm) {
                        HStack(spacing: Spacing.sm) {
                            Image(systemName: "exclamationmark.circle.fill")
                                .foregroundStyle(.red)
                            Text(errorMessage)
                                .font(PurpTapeTypography.bodySmall)
                                .foregroundStyle(.red)
                        }
                    }
                    .padding(Spacing.md)
                    .background(Color.red.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                }
                
                Spacer()
                
                // Upload Button
                Button(action: uploadTrack) {
                    if isLoading {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                    } else {
                        Text("Upload Track")
                    }
                }
                .frame(maxWidth: .infinity)
                .padding(Spacing.md)
                .background(selectedAudioURL != nil && !trackTitle.isEmpty ? PurpTapeColors.primary : Color.gray)
                .foregroundStyle(.white)
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .disabled(selectedAudioURL == nil || trackTitle.isEmpty || isLoading)
            }
            .padding(Spacing.lg)
            .navigationTitle("Add Track")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") {
                        isPresented = false
                    }
                }
            }
        }
    }
    
    private func extractTrackMetadata(from url: URL) {
        // Extract filename as default title
        let title = url.deletingPathExtension().lastPathComponent
        
        trackTitle = title
        errorMessage = nil
    }

    private func readData(from url: URL) throws -> Data {
        let didAccessSecurityScope = url.startAccessingSecurityScopedResource()
        defer {
            if didAccessSecurityScope {
                url.stopAccessingSecurityScopedResource()
            }
        }

        return try Data(contentsOf: url)
    }

    private func mimeType(for url: URL) -> String? {
        let fileExtension = url.pathExtension
        guard !fileExtension.isEmpty,
              let type = UTType(filenameExtension: fileExtension) else {
            return nil
        }
        return type.preferredMIMEType
    }
    
    private func uploadTrack() {
        guard let fileData = audioData, !trackTitle.isEmpty else {
            errorMessage = "Please select a file and enter a title"
            return
        }
        
        isLoading = true
        
        Task {
            do {
                // Pass empty string for accessToken - APIClient handles auth internally
                let track = try await dataStore.createTrack(
                    projectID: projectID,
                    title: trackTitle,
                    audioData: fileData,
                    fileName: selectedAudioFileName,
                    mimeType: selectedAudioMimeType,
                    accessToken: ""
                )
                
                await MainActor.run {
                    onTrackAdded(track)
                    isLoading = false
                    isPresented = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = "Failed to upload track: \(error.localizedDescription)"
                    isLoading = false
                }
            }
        }
    }
}

struct FilePickerButton: View {
    let onFilePicked: (URL) -> Void
    @State private var showFilePicker = false
    
    var body: some View {
        Button(action: { showFilePicker = true }) {
            VStack(spacing: Spacing.md) {
                Image(systemName: "music.note.list")
                    .font(.system(size: 40))
                    .foregroundStyle(PurpTapeColors.primary)
                
                VStack(spacing: Spacing.sm) {
                    Text("Select Audio File")
                        .font(PurpTapeTypography.bodyMedium)
                        .foregroundStyle(PurpTapeColors.text)
                    
                    Text("MP3, WAV, or M4A")
                        .font(PurpTapeTypography.bodySmall)
                        .foregroundStyle(PurpTapeColors.textSecondary)
                }
            }
        }
        .buttonStyle(.plain)
        .fileImporter(
            isPresented: $showFilePicker,
            allowedContentTypes: [.audio],
            onCompletion: { result in
                if case .success(let url) = result {
                    onFilePicked(url)
                }
            }
        )
    }
}

#Preview {
    AddTrackSheet(
        isPresented: .constant(true),
        projectID: UUID(),
        apiClient: MockAPIClient(),
        onTrackAdded: { _ in }
    )
}

private actor MockAPIClient: APIClient {
    func send<T>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T where T : Decodable, T : Sendable {
        throw APIClientError.invalidResponse
    }
    func upload<T>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T where T : Decodable, T : Sendable {
        throw APIClientError.invalidResponse
    }
}
