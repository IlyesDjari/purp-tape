import Foundation

public struct Project: Sendable, Identifiable, Codable, Hashable {
    public let id: UUID
    public let name: String
    public let description: String?
    public let isPublic: Bool
    public let coverImageURL: String?

    public init(id: UUID, name: String, description: String?, isPublic: Bool, coverImageURL: String? = nil) {
        self.id = id
        self.name = name
        self.description = description
        self.isPublic = isPublic
        self.coverImageURL = coverImageURL
    }
}

public struct Track: Sendable, Identifiable, Codable, Hashable {
    public let id: UUID
    public let projectID: UUID
    public let title: String
    public let durationSeconds: Int

    public init(id: UUID, projectID: UUID, title: String, durationSeconds: Int) {
        self.id = id
        self.projectID = projectID
        self.title = title
        self.durationSeconds = durationSeconds
    }
}

public struct TrackVersion: Sendable, Identifiable, Codable, Hashable {
    public let id: UUID
    public let trackID: UUID
    public let versionNumber: Int
    public let fileURL: URL
    public let fileSizeBytes: Int64

    public init(id: UUID, trackID: UUID, versionNumber: Int, fileURL: URL, fileSizeBytes: Int64) {
        self.id = id
        self.trackID = trackID
        self.versionNumber = versionNumber
        self.fileURL = fileURL
        self.fileSizeBytes = fileSizeBytes
    }
}
