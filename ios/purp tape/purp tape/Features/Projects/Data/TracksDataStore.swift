import Foundation
import UniformTypeIdentifiers

protocol TracksDataStore: Sendable {
    func fetchTracks(projectID: UUID, accessToken: String) async throws -> [Track]
    func fetchSignedPlaybackURL(trackID: UUID, accessToken: String) async throws -> URL
    func deleteTrack(trackID: UUID, accessToken: String) async throws
    func createTrack(
        projectID: UUID,
        title: String,
        audioData: Data,
        fileName: String,
        mimeType: String?,
        accessToken: String
    ) async throws -> Track
}

actor URLSessionTracksDataStore: TracksDataStore {
    private let apiClient: APIClient
    private let logger = DebugLogger(category: "data.tracks")
    
    init(apiClient: APIClient) {
        self.apiClient = apiClient
    }
    
    func fetchTracks(projectID: UUID, accessToken: String) async throws -> [Track] {
        let endpoint = Endpoint(
            path: "projects/\(projectID.uuidString)/tracks",
            method: "GET"
        )
        
        let response: TracksListResponse = try await apiClient.send(endpoint, decode: TracksListResponse.self)
        return response.data.map { $0.toTrack() }
    }

    func fetchSignedPlaybackURL(trackID: UUID, accessToken: String) async throws -> URL {
        logger.network("Fetching signed playback URL for track \(trackID.uuidString)")

        let endpoint = Endpoint(
            path: "tracks/\(trackID.uuidString)/play",
            method: "GET"
        )

        do {
            let response: SignedPlaybackURLResponse = try await apiClient.send(endpoint, decode: SignedPlaybackURLResponse.self)

            guard let url = URL(string: response.url) else {
                logger.error("Signed playback URL invalid for track \(trackID.uuidString)")
                throw APIClientError.transportError("Invalid signed playback URL")
            }

            logger.success("Signed playback URL ready for track \(trackID.uuidString)")
            return url
        } catch {
            logger.error("Signed playback URL fetch failed for track \(trackID.uuidString): \(error.localizedDescription)")
            throw error
        }
    }

    func deleteTrack(trackID: UUID, accessToken: String) async throws {
        logger.network("Deleting track \(trackID.uuidString)")

        let endpoint = Endpoint(
            path: "tracks/\(trackID.uuidString)",
            method: "DELETE"
        )

        _ = try await apiClient.send(endpoint, decode: DeleteTrackResponse.self)
        logger.success("Track deleted \(trackID.uuidString)")
    }
    
    func createTrack(
        projectID: UUID,
        title: String,
        audioData: Data,
        fileName: String,
        mimeType: String?,
        accessToken: String
    ) async throws -> Track {
        if audioData.count > 100 * 1024 * 1024 {
            throw APIClientError.transportError("Audio file too large (max 100MB)")
        }

        let createEndpoint = Endpoint(
            path: "projects/\(projectID.uuidString)/tracks",
            method: "POST",
            headers: ["Content-Type": "application/json"],
            body: try JSONEncoder().encode(CreateTrackRequest(name: title))
        )

        let createdTrackResponse: CreateTrackResponse = try await apiClient.send(createEndpoint, decode: CreateTrackResponse.self)
        let createdTrack = createdTrackResponse.track

        let boundary = UUID().uuidString
        var body = Data()

        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"file\"; filename=\"\(sanitizeFileName(fileName))\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(resolvedMimeType(for: fileName, explicitMimeType: mimeType))\r\n\r\n".data(using: .utf8)!)
        body.append(audioData)
        body.append("\r\n".data(using: .utf8)!)
        body.append("--\(boundary)--\r\n".data(using: .utf8)!)

        let uploadEndpoint = Endpoint(
            path: "tracks/\(createdTrack.id.uuidString)/versions",
            method: "POST",
            headers: ["Content-Type": "multipart/form-data; boundary=\(boundary)"],
            body: body
        )

        _ = try await apiClient.send(uploadEndpoint, decode: TrackVersionUploadResponse.self)
        return createdTrack.toTrack()
    }

    private func sanitizeFileName(_ value: String) -> String {
        let cleaned = value
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "\"", with: "")
            .replacingOccurrences(of: "\r", with: "")
            .replacingOccurrences(of: "\n", with: "")
        return cleaned.isEmpty ? "track.m4a" : cleaned
    }

    private func resolvedMimeType(for fileName: String, explicitMimeType: String?) -> String {
        if let explicitMimeType, !explicitMimeType.isEmpty {
            return explicitMimeType
        }

        let fileExtension = URL(fileURLWithPath: fileName).pathExtension
        if !fileExtension.isEmpty,
           let type = UTType(filenameExtension: fileExtension),
           let mimeType = type.preferredMIMEType {
            return mimeType
        }

        return "application/octet-stream"
    }
}

private struct TracksListResponse: Decodable {
    let data: [TrackDTO]
}

private struct SignedPlaybackURLResponse: Decodable {
    let url: String
    let expiresInSeconds: Int?

    private enum CodingKeys: String, CodingKey {
        case url
        case expiresInSeconds = "expires_in_seconds"
    }
}

private struct CreateTrackRequest: Encodable {
    let name: String
}

private struct DeleteTrackResponse: Decodable {
    let deleted: Bool
}

private struct CreateTrackResponse: Decodable {
    let track: TrackDTO

    private enum CodingKeys: String, CodingKey {
        case data
    }

    init(from decoder: Decoder) throws {
        if let direct = try? TrackDTO(from: decoder) {
            track = direct
            return
        }

        let container = try decoder.container(keyedBy: CodingKeys.self)
        track = try container.decode(TrackDTO.self, forKey: .data)
    }
}

private struct TrackVersionUploadResponse: Decodable {}

private struct TrackDTO: Decodable {
    let id: UUID
    let projectID: UUID
    let title: String
    let durationSeconds: Int

    private enum CodingKeys: String, CodingKey {
        case id
        case idUpper = "ID"
        case projectID
        case projectIDUpper = "ProjectID"
        case projectIDSnake = "project_id"
        case title
        case name
        case nameUpper = "Name"
        case durationSeconds
        case duration
        case durationUpper = "Duration"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try (container.decodeIfPresent(UUID.self, forKey: .id)
            ?? container.decode(UUID.self, forKey: .idUpper))
        projectID = (try? container.decode(UUID.self, forKey: .projectIDSnake))
            ?? (try? container.decode(UUID.self, forKey: .projectID))
            ?? (try? container.decode(UUID.self, forKey: .projectIDUpper))
            ?? id

        title = (try? container.decode(String.self, forKey: .title))
            ?? (try? container.decode(String.self, forKey: .name))
            ?? (try? container.decode(String.self, forKey: .nameUpper))
            ?? "Untitled"

        durationSeconds = (try? container.decode(Int.self, forKey: .duration))
            ?? (try? container.decode(Int.self, forKey: .durationSeconds))
            ?? (try? container.decode(Int.self, forKey: .durationUpper))
            ?? 0
    }

    func toTrack() -> Track {
        Track(
            id: id,
            projectID: projectID,
            title: title,
            durationSeconds: durationSeconds
        )
    }
}
