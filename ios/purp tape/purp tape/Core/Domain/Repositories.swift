import Foundation

public protocol ProjectRepository: Sendable {
    func fetchProjects() async throws -> [Project]
}

public protocol TrackRepository: Sendable {
    func fetchTracks(projectID: UUID) async throws -> [Track]
    func fetchVersions(trackID: UUID) async throws -> [TrackVersion]
}
