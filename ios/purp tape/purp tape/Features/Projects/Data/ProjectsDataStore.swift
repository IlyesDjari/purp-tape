import Foundation
import os
import ImageIO
import UniformTypeIdentifiers

protocol ProjectsDataStore: Sendable {
    func fetchProjects(accessToken: String) async throws -> [Project]
    func createProject(draft: ProjectDraft, accessToken: String) async throws -> Project
    func deleteProject(projectID: UUID, accessToken: String) async throws
    func uploadProjectCover(projectID: UUID, imageData: Data, accessToken: String, altText: String?) async throws
}

actor URLSessionProjectsDataStore: ProjectsDataStore {
    private let baseURL: URL
    private let session: URLSession
    private let authService: AuthService?
    private let logger = Logger(subsystem: "ilyes.purp-tape", category: "projects.data")
    private let maxCoverUploadBytes = 5 * 1024 * 1024
    private let targetCoverUploadBytes = 4_800_000

    init(baseURL: URL, session: URLSession = .shared, authService: AuthService? = nil) {
        self.baseURL = baseURL
        self.session = session
        self.authService = authService
    }

    func fetchProjects(accessToken: String) async throws -> [Project] {
        // SECURITY: If auth service is available, get fresh token instead of using stale parameter
        let activeToken = try await getFreshToken(fallback: accessToken)
        
        logger.debug("Fetching projects with token")
        var request = URLRequest(url: baseURL.appending(path: "projects"))
        request.httpMethod = "GET"
        request.setValue("Bearer \(activeToken)", forHTTPHeaderField: "Authorization")

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
                let projects = try decodeProjectsList(from: data)
                logger.debug("Fetched projects count=\(projects.count, privacy: .public)")
                return projects
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
    
    /// Get fresh token from auth service if available, otherwise use fallback
    private func getFreshToken(fallback: String) async throws -> String {
        guard let authService = authService else {
            return fallback
        }
        
        do {
            if let session = try await authService.currentSession() {
                return session.accessToken
            }
        } catch {
            logger.warning("Failed to get fresh token from auth service: \(error.localizedDescription), using fallback")
        }
        
        return fallback
    }

    private func decodeProjectsList(from data: Data) throws -> [Project] {
        // Empty success response = no projects
        if data.isEmpty {
            return []
        }

        let decoder = JSONDecoder()

        // Primary shape: { data: [...] }
        if let envelope = try? decoder.decode(ProjectListResponseDTO.self, from: data) {
            return mapProjectItems(envelope.data ?? [])
        }

        // Alternate shape: { projects: [...] }
        if let altEnvelope = try? decoder.decode(AltProjectListResponseDTO.self, from: data) {
            return mapProjectItems(altEnvelope.projects ?? [])
        }

        // Raw array shape: [ ... ]
        if let items = try? decoder.decode([ProjectItemDTO].self, from: data) {
            return mapProjectItems(items)
        }

        throw ProjectsDataStoreError.decodingFailed("Unsupported projects response format")
    }

    private func mapProjectItems(_ items: [ProjectItemDTO]) -> [Project] {
        items.map {
            Project(
                id: $0.id,
                name: $0.name,
                description: $0.description,
                isPublic: $0.isPublic,
                coverImageURL: $0.coverImageURL
            )
        }
    }

    private func decodeCreatedProject(from data: Data, draft: ProjectDraft) throws -> Project {
        if data.isEmpty {
            throw ProjectsDataStoreError.decodingFailed("Empty create project response")
        }

        let decoder = JSONDecoder()

        if let envelope = try? decoder.decode(CreateProjectResponseDTO.self, from: data),
           let created = envelope.data {
            return Project(
                id: created.id,
                name: created.name,
                description: created.description ?? draft.description,
                isPublic: created.isPublic ?? draft.isPublic
            )
        }

        if let altEnvelope = try? decoder.decode(CreateProjectAltResponseDTO.self, from: data),
           let created = altEnvelope.project {
            return Project(
                id: created.id,
                name: created.name,
                description: created.description ?? draft.description,
                isPublic: created.isPublic ?? draft.isPublic
            )
        }

        if let created = try? decoder.decode(CreatedProjectDTO.self, from: data) {
            return Project(
                id: created.id,
                name: created.name,
                description: created.description ?? draft.description,
                isPublic: created.isPublic ?? draft.isPublic
            )
        }

        // Last-resort dynamic parsing for unexpected success payload shapes.
        if let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let project = parseCreatedProject(from: root, draft: draft) {
            return project
        }

        throw ProjectsDataStoreError.decodingFailed("Unsupported create project response format")
    }

    private func parseCreatedProject(from root: [String: Any], draft: ProjectDraft) -> Project? {
        let candidates: [[String: Any]] = {
            var values: [[String: Any]] = [root]
            if let data = root["data"] as? [String: Any] { values.append(data) }
            if let data = root["Data"] as? [String: Any] { values.append(data) }
            if let project = root["project"] as? [String: Any] { values.append(project) }
            if let project = root["Project"] as? [String: Any] { values.append(project) }
            return values
        }()

        for dict in candidates {
            let idString = (dict["id"] as? String) ?? (dict["ID"] as? String)

            if let idString,
               let id = UUID(uuidString: idString) {
                let name = (dict["name"] as? String) ?? (dict["Name"] as? String) ?? draft.name
                let description = (dict["description"] as? String) ?? (dict["Description"] as? String) ?? draft.description

                let isPublic: Bool = {
                    if let value = dict["is_public"] as? Bool { return value }
                    if let value = dict["isPrivate"] as? Bool { return !value }
                    if let value = dict["is_private"] as? Bool { return !value }
                    if let value = dict["IsPrivate"] as? Bool { return !value }
                    return draft.isPublic
                }()

                return Project(id: id, name: name, description: description, isPublic: isPublic)
            }
        }

        return nil
    }

    func createProject(draft: ProjectDraft, accessToken: String) async throws -> Project {
        // SECURITY: If auth service is available, get fresh token instead of using stale parameter
        let activeToken = try await getFreshToken(fallback: accessToken)
        
        logger.debug("Creating project name=\(draft.name, privacy: .public)")
        let payload = CreateProjectRequestDTO(
            name: draft.name,
            description: draft.description,
            isPublic: draft.isPublic
        )

        var request = URLRequest(url: baseURL.appending(path: "projects"))
        request.httpMethod = "POST"
        request.setValue("Bearer \(activeToken)", forHTTPHeaderField: "Authorization")
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
                let created = try decodeCreatedProject(from: data, draft: draft)
                logger.debug("Created project id=\(created.id.uuidString, privacy: .public)")
                return created
            } catch let error as ProjectsDataStoreError {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Create project decode failed: \(error.localizedDescription, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw error
            } catch {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Create project decode failed: \(error.localizedDescription, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw ProjectsDataStoreError.decodingFailed(error.localizedDescription)
            }
        } catch let error as ProjectsDataStoreError {
            throw error
        } catch {
            logger.error("Create project transport failed: \(error.localizedDescription, privacy: .public)")
            throw ProjectsDataStoreError.transportFailed(error.localizedDescription)
        }
    }

    func deleteProject(projectID: UUID, accessToken: String) async throws {
        let activeToken = try await getFreshToken(fallback: accessToken)

        logger.debug("Deleting project id=\(projectID.uuidString, privacy: .public)")
        var request = URLRequest(url: baseURL.appending(path: "projects/\(projectID.uuidString)"))
        request.httpMethod = "DELETE"
        request.setValue("Bearer \(activeToken)", forHTTPHeaderField: "Authorization")

        do {
            let (data, response) = try await session.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                throw ProjectsDataStoreError.transportFailed("Invalid response")
            }

            guard 200 ..< 300 ~= http.statusCode else {
                let body = String(data: data, encoding: .utf8) ?? ""
                logger.error("Delete project failed status=\(http.statusCode, privacy: .public) body=\(body.prefix(200), privacy: .public)")
                throw ProjectsDataStoreError.requestFailed(statusCode: http.statusCode, body: String(body.prefix(200)))
            }

            logger.debug("Deleted project id=\(projectID.uuidString, privacy: .public)")
        } catch let error as ProjectsDataStoreError {
            throw error
        } catch {
            logger.error("Delete project transport failed: \(error.localizedDescription, privacy: .public)")
            throw ProjectsDataStoreError.transportFailed(error.localizedDescription)
        }
    }

    func uploadProjectCover(projectID: UUID, imageData: Data, accessToken: String, altText: String?) async throws {
        // SECURITY: If auth service is available, get fresh token instead of using stale parameter
        let activeToken = try await getFreshToken(fallback: accessToken)

        // Production-safe upload size handling: compress/downscale before sending
        let optimizedImageData = try optimizeCoverImageForUpload(imageData)
        
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
        body.append(optimizedImageData)
        body.append("\r\n")
        body.append("--\(boundary)--\r\n")

        var request = URLRequest(url: baseURL.appending(path: "projects/\(projectID.uuidString)/cover"))
        request.httpMethod = "POST"
        request.setValue("Bearer \(activeToken)", forHTTPHeaderField: "Authorization")
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

    private func optimizeCoverImageForUpload(_ originalData: Data) throws -> Data {
        if originalData.count <= targetCoverUploadBytes {
            return originalData
        }

        guard let source = CGImageSourceCreateWithData(originalData as CFData, nil),
              let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any] else {
            throw ProjectsDataStoreError.decodingFailed("Invalid image data")
        }

        let width = (properties[kCGImagePropertyPixelWidth] as? NSNumber)?.intValue ?? 2048
        let height = (properties[kCGImagePropertyPixelHeight] as? NSNumber)?.intValue ?? 2048
        var maxPixel = max(width, height)

        var quality: CGFloat = 0.85
        var bestData: Data?

        for _ in 0..<16 {
            guard let cgImage = createThumbnail(from: source, maxPixel: maxPixel),
                  let data = encodeJPEG(cgImage, quality: quality) else {
                break
            }

            if data.count <= targetCoverUploadBytes {
                logger.debug("Cover image optimized from \(originalData.count, privacy: .public) bytes to \(data.count, privacy: .public) bytes")
                return data
            }

            if bestData == nil || data.count < (bestData?.count ?? Int.max) {
                bestData = data
            }

            if quality > 0.35 {
                quality -= 0.1
            } else {
                maxPixel = Int(Double(maxPixel) * 0.8)
                quality = 0.8
            }

            if maxPixel < 320 {
                break
            }
        }

        guard let compressedData = bestData else {
            throw ProjectsDataStoreError.decodingFailed("Unable to optimize image")
        }

        logger.debug("Cover image optimized from \(originalData.count, privacy: .public) bytes to \(compressedData.count, privacy: .public) bytes")

        if compressedData.count > maxCoverUploadBytes {
            throw ProjectsDataStoreError.requestFailed(
                statusCode: 400,
                body: "cover image too large after optimization: max 5MB"
            )
        }

        return compressedData
    }

    private func createThumbnail(from source: CGImageSource, maxPixel: Int) -> CGImage? {
        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceThumbnailMaxPixelSize: maxPixel,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: false
        ]

        return CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary)
    }

    private func encodeJPEG(_ image: CGImage, quality: CGFloat) -> Data? {
        let mutableData = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(
            mutableData,
            UTType.jpeg.identifier as CFString,
            1,
            nil
        ) else {
            return nil
        }

        let properties: [CFString: Any] = [
            kCGImageDestinationLossyCompressionQuality: quality
        ]

        CGImageDestinationAddImage(destination, image, properties as CFDictionary)
        guard CGImageDestinationFinalize(destination) else {
            return nil
        }

        return mutableData as Data
    }
}

enum ProjectsDataStoreError: Error, LocalizedError {
    case requestFailed(statusCode: Int, body: String)
    case decodingFailed(String)
    case transportFailed(String)

    var errorDescription: String? {
        switch self {
        case let .requestFailed(statusCode, body):
            return "HTTP \(statusCode): \(body)"
        case let .decodingFailed(message):
            return "Decoding failed: \(message)"
        case let .transportFailed(message):
            return "Transport failed: \(message)"
        }
    }
}

private extension Data {
    mutating func append(_ string: String) {
        if let data = string.data(using: .utf8) {
            append(data)
        }
    }
}

private struct ProjectListResponseDTO: Decodable {
    let data: [ProjectItemDTO]?
}

private struct AltProjectListResponseDTO: Decodable {
    let projects: [ProjectItemDTO]?
}

private struct ProjectItemDTO: Decodable {
    let id: UUID
    let name: String
    let description: String?
    let isPublic: Bool
    let coverImageURL: String?

    private enum CodingKeys: String, CodingKey {
        case id
        case idUpper = "ID"
        case name
        case nameUpper = "Name"
        case description
        case descriptionUpper = "Description"
        case isPublic = "is_public"
        case isPublicCamel = "isPublic"
        case isPrivate = "is_private"
        case isPrivateCamel = "isPrivate"
        case isPrivateUpper = "IsPrivate"
        case coverImageURL = "cover_image_url"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        if let value = try container.decodeIfPresent(UUID.self, forKey: .id) {
            id = value
        } else {
            id = try container.decode(UUID.self, forKey: .idUpper)
        }

        if let value = try container.decodeIfPresent(String.self, forKey: .name) {
            name = value
        } else {
            name = try container.decode(String.self, forKey: .nameUpper)
        }

        description = try container.decodeIfPresent(String.self, forKey: .description)
            ?? container.decodeIfPresent(String.self, forKey: .descriptionUpper)

        let publicBool = try container.decodeIfPresent(Bool.self, forKey: .isPublic)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPublicCamel)

        let privateBool = try container.decodeIfPresent(Bool.self, forKey: .isPrivate)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPrivateCamel)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPrivateUpper)

        if let publicBool {
            isPublic = publicBool
        } else if let privateBool {
            isPublic = !privateBool
        } else {
            isPublic = true
        }

        coverImageURL = try container.decodeIfPresent(String.self, forKey: .coverImageURL)
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
    let data: CreatedProjectDTO?
}

private struct CreateProjectAltResponseDTO: Decodable {
    let project: CreatedProjectDTO?
}

private struct CreatedProjectDTO: Decodable {
    let id: UUID
    let name: String
    let description: String?
    let isPublic: Bool?

    private enum CodingKeys: String, CodingKey {
        case id
        case idUpper = "ID"
        case name
        case nameUpper = "Name"
        case description
        case descriptionUpper = "Description"
        case isPublic = "is_public"
        case isPublicCamel = "isPublic"
        case isPrivate = "is_private"
        case isPrivateCamel = "isPrivate"
        case isPrivateUpper = "IsPrivate"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        if let value = try container.decodeIfPresent(UUID.self, forKey: .id) {
            id = value
        } else {
            id = try container.decode(UUID.self, forKey: .idUpper)
        }

        if let value = try container.decodeIfPresent(String.self, forKey: .name) {
            name = value
        } else {
            name = try container.decode(String.self, forKey: .nameUpper)
        }

        description = try container.decodeIfPresent(String.self, forKey: .description)
            ?? container.decodeIfPresent(String.self, forKey: .descriptionUpper)

        let publicBool = try container.decodeIfPresent(Bool.self, forKey: .isPublic)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPublicCamel)

        let privateBool = try container.decodeIfPresent(Bool.self, forKey: .isPrivate)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPrivateCamel)
            ?? container.decodeIfPresent(Bool.self, forKey: .isPrivateUpper)

        if let publicBool {
            isPublic = publicBool
        } else if let privateBool {
            isPublic = !privateBool
        } else {
            isPublic = nil
        }
    }
}
