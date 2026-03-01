import Foundation
import os

protocol ProjectsDataStore: Sendable {
    func fetchProjects(accessToken: String) async throws -> [Project]
    func createProject(draft: ProjectDraft, accessToken: String) async throws -> Project
    func uploadProjectCover(projectID: UUID, imageData: Data, accessToken: String, altText: String?) async throws
}

actor URLSessionProjectsDataStore: ProjectsDataStore {
    private let baseURL: URL
    private let session: URLSession
    private let logger = Logger(subsystem: "ilyes.purp-tape", category: "projects.data")

    init(baseURL: URL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
    }

    func fetchProjects(accessToken: String) async throws -> [Project] {
        logger.debug("Fetching projects")
        var request = URLRequest(url: baseURL.appending(path: "projects"))
        request.httpMethod = "GET"
        request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")

        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                logger.error("Fetch projects received invalid response")
                throw ProjectsDataStoreError.transportFailed("Invalid response")
            }

            guard 200 ..< 300 ~= http.statusCode else {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Fetch projects failed status=\(http.statusCode, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw ProjectsDataStoreError.requestFailed(statusCode: http.statusCode, body: String(body.prefix(200)))
            }

            do {
                let decoded = try JSONDecoder().decode(ProjectListResponseDTO.self, from: data)
                logger.debug("Fetched projects count=\(decoded.data.count, privacy: .public)")
                return decoded.data.map {
                    Project(
                        id: $0.id,
                        name: $0.name,
                        description: $0.description,
                        isPublic: $0.isPublic
                    )
                }
            } catch {
                logger.error("Fetch projects decode failed: \(error.localizedDescription, privacy: .public)")
                throw ProjectsDataStoreError.decodingFailed(error.localizedDescription)
            }
        } catch let error as ProjectsDataStoreError {
            throw error
        } catch {
            logger.error("Fetch projects transport failed: \(error.localizedDescription, privacy: .public)")
            throw ProjectsDataStoreError.transportFailed(error.localizedDescription)
        }
    }

    func createProject(draft: ProjectDraft, accessToken: String) async throws -> Project {
        logger.debug("Creating project name=\(draft.name, privacy: .public)")
        let payload = CreateProjectRequestDTO(
            name: draft.name,
            description: draft.description,
            isPublic: draft.isPublic
        )

        var request = URLRequest(url: baseURL.appending(path: "projects"))
        request.httpMethod = "POST"
        request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(payload)

        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                logger.error("Create project received invalid response")
                throw ProjectsDataStoreError.transportFailed("Invalid response")
            }

            guard 200 ..< 300 ~= http.statusCode else {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Create project failed status=\(http.statusCode, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw ProjectsDataStoreError.requestFailed(statusCode: http.statusCode, body: String(body.prefix(200)))
            }

            do {
                let decoded = try JSONDecoder().decode(CreateProjectResponseDTO.self, from: data)
                logger.debug("Created project id=\(decoded.data.id.uuidString, privacy: .public)")
                return Project(
                    id: decoded.data.id,
                    name: decoded.data.name,
                    description: draft.description,
                    isPublic: draft.isPublic
                )
            } catch {
                logger.error("Create project decode failed: \(error.localizedDescription, privacy: .public)")
                throw ProjectsDataStoreError.decodingFailed(error.localizedDescription)
            }
        } catch let error as ProjectsDataStoreError {
            throw error
        } catch {
            logger.error("Create project transport failed: \(error.localizedDescription, privacy: .public)")
            throw ProjectsDataStoreError.transportFailed(error.localizedDescription)
        }
    }

    func uploadProjectCover(projectID: UUID, imageData: Data, accessToken: String, altText: String?) async throws {
        logger.debug("Uploading project cover projectID=\(projectID.uuidString, privacy: .public)")
        let boundary = "Boundary-\(UUID().uuidString)"
        var body = Data()

        if let altText, !altText.isEmpty {
            body.append("--\(boundary)\r\n")
            body.append("Content-Disposition: form-data; name=\"alt_text\"\r\n\r\n")
            body.append("\(altText)\r\n")
        }

        body.append("--\(boundary)\r\n")
        body.append("Content-Disposition: form-data; name=\"cover\"; filename=\"cover.jpg\"\r\n")
        body.append("Content-Type: image/jpeg\r\n\r\n")
        body.append(imageData)
        body.append("\r\n")
        body.append("--\(boundary)--\r\n")

        var request = URLRequest(url: baseURL.appending(path: "projects/\(projectID.uuidString)/cover"))
        request.httpMethod = "POST"
        request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        request.httpBody = body

        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                logger.error("Upload cover received invalid response")
                throw ProjectsDataStoreError.transportFailed("Invalid response")
            }

            guard 200 ..< 300 ~= http.statusCode else {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Upload cover failed status=\(http.statusCode, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw ProjectsDataStoreError.requestFailed(statusCode: http.statusCode, body: String(body.prefix(200)))
            }

            logger.debug("Uploaded project cover successfully")
        } catch let error as ProjectsDataStoreError {
            throw error
        } catch {
            logger.error("Upload cover transport failed: \(error.localizedDescription, privacy: .public)")
            throw ProjectsDataStoreError.transportFailed(error.localizedDescription)
        }
    }
}

enum ProjectsDataStoreError: Error {
    case requestFailed(statusCode: Int, body: String)
    case decodingFailed(String)
    case transportFailed(String)
}

private extension Data {
    mutating func append(_ string: String) {
        if let data = string.data(using: .utf8) {
            append(data)
        }
    }
}

private struct ProjectListResponseDTO: Decodable {
    let data: [ProjectItemDTO]
}

private struct ProjectItemDTO: Decodable {
    let id: UUID
    let name: String
    let description: String?
    let isPublic: Bool

    private enum CodingKeys: String, CodingKey {
        case id
        case name
        case description
        case isPublic = "is_public"
    }
}

private struct CreateProjectRequestDTO: Encodable {
    let name: String
    let description: String?
    let isPublic: Bool

    private enum CodingKeys: String, CodingKey {
        case name
        case description
        case isPublic = "is_public"
    }
}

private struct CreateProjectResponseDTO: Decodable {
    let data: CreatedProjectDTO
}

private struct CreatedProjectDTO: Decodable {
    let id: UUID
    let name: String
}
