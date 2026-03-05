import Foundation
import os

@MainActor
final class ProjectsTabViewModel: ObservableObject {
    @Published var projects: [Project] = []
    @Published var cachedProjectsCount: Int = 0
    @Published var isLoading = false
    @Published var isCreating = false
    @Published var errorMessage: String?
    
    private let dataStore: ProjectsDataStore
    private let appDefaults: AppUserDefaults
    private let logger = Logger(subsystem: "ilyes.purp-tape", category: "projects.vm")
    
    init(dataStore: ProjectsDataStore? = nil, authService: AuthService? = nil, appDefaults: AppUserDefaults = .shared) {
        self.appDefaults = appDefaults
        self.cachedProjectsCount = max(1, appDefaults.int(for: .cachedProjectsCount, default: 1))
        
        if let dataStore {
            self.dataStore = dataStore
            return
        }
        
        if let raw = Bundle.main.object(forInfoDictionaryKey: "PURPTAPE_API_BASE_URL") as? String,
           let baseURL = URL(string: raw) {
            self.dataStore = URLSessionProjectsDataStore(baseURL: baseURL, authService: authService)
        } else {
            self.dataStore = FallbackProjectsDataStore()
        }
    }
    
    func loadProjects(accessToken: String?) async {
        guard !isLoading else { return }
        guard let accessToken else {
            logger.error("Load projects aborted: missing access token")
            errorMessage = "Authentication required"
            return
        }
        
        logger.debug("Load projects started")
        isLoading = true
        defer { isLoading = false }
        
        do {
            projects = try await dataStore.fetchProjects(accessToken: accessToken)
            cachedProjectsCount = max(1, projects.count)
            appDefaults.setInt(projects.count, for: .cachedProjectsCount)
            errorMessage = nil
            logger.debug("Load projects succeeded count=\(self.projects.count, privacy: .public)")
            for project in projects {
                print("[DEBUG] Project \(project.id) coverImageURL: \(project.coverImageURL ?? "nil")")
            }
        } catch let error as ProjectsDataStoreError {
            let mapped = mapDataStoreError(error, fallback: "Failed to load projects")
            logger.error("Load projects failed: \(mapped, privacy: .public)")
            errorMessage = mapped
        } catch {
            logger.error("Load projects unexpected error: \(error.localizedDescription, privacy: .public)")
            errorMessage = "Failed to load projects"
        }
    }
    
    func createProject(
        name: String,
        description: String,
        isPublic: Bool,
        artworkData: Data?,
        accessToken: String?
    ) async -> Bool {
        guard !isCreating else { return false }
        guard let accessToken else {
            logger.error("Create project aborted: missing access token")
            errorMessage = "Authentication required"
            return false
        }
        
        logger.debug("Create project started name=\(name, privacy: .public)")
        isCreating = true
        defer { isCreating = false }
        
        do {
            let draft = ProjectDraft(
                name: name,
                description: description.isEmpty ? nil : description,
                isPublic: isPublic
            )
            let created = try await dataStore.createProject(draft: draft, accessToken: accessToken)
            projects.insert(created, at: 0)
            cachedProjectsCount = max(1, projects.count)
            appDefaults.setInt(projects.count, for: .cachedProjectsCount)
            
            if let artworkData {
                do {
                    try await dataStore.uploadProjectCover(
                        projectID: created.id,
                        imageData: artworkData,
                        accessToken: accessToken,
                        altText: "Cover for \(created.name)"
                    )
                } catch {
                    logger.error("Create project cover upload failed: \(error.localizedDescription, privacy: .public)")
                    errorMessage = "Project created, but cover upload failed"
                    return true
                }
            }
            
            errorMessage = nil
            logger.debug("Create project succeeded id=\(created.id.uuidString, privacy: .public)")
            return true
        } catch let error as ProjectsDataStoreError {
            let mapped = mapDataStoreError(error, fallback: "Failed to create project")
            logger.error("Create project failed: \(mapped, privacy: .public)")
            errorMessage = mapped
            return false
        } catch {
            logger.error("Create project unexpected error: \(error.localizedDescription, privacy: .public)")
            errorMessage = "Failed to create project"
            return false
        }
    }
    
    func deleteProject(projectID: UUID, accessToken: String?) async -> Bool {
        guard let accessToken else {
            logger.error("Delete project aborted: missing access token")
            errorMessage = "Authentication required"
            return false
        }
        
        do {
            try await dataStore.deleteProject(projectID: projectID, accessToken: accessToken)
            projects.removeAll { $0.id == projectID }
            // No cover data to clear; handled by backend and image component
            cachedProjectsCount = max(1, projects.count)
            appDefaults.setInt(projects.count, for: .cachedProjectsCount)
            errorMessage = nil
            logger.debug("Delete project succeeded id=\(projectID.uuidString, privacy: .public)")
            return true
        } catch let error as ProjectsDataStoreError {
            let mapped = mapDataStoreError(error, fallback: "Failed to delete project")
            logger.error("Delete project failed: \(mapped, privacy: .public)")
            errorMessage = mapped
            return false
        } catch {
            logger.error("Delete project unexpected error: \(error.localizedDescription, privacy: .public)")
            errorMessage = "Failed to delete project"
            return false
        }
    }
    
    private func mapDataStoreError(_ error: ProjectsDataStoreError, fallback: String) -> String {
        switch error {
        case .requestFailed(let statusCode, let body):
            if body.isEmpty {
                return "\(fallback) (HTTP \(statusCode))"
            }
            return "\(fallback) (HTTP \(statusCode)): \(body)"
        case .decodingFailed(let details):
            return "\(fallback) (decode): \(details)"
        case .transportFailed(let details):
            return "\(fallback) (network): \(details)"
        }
    }
}

private actor FallbackProjectsDataStore: ProjectsDataStore {
    func fetchProjects(accessToken: String) async throws -> [Project] {
        []
    }
    
    func createProject(draft: ProjectDraft, accessToken: String) async throws -> Project {
        Project(id: UUID(), name: draft.name, description: draft.description, isPublic: draft.isPublic)
    }
    
    func deleteProject(projectID: UUID, accessToken: String) async throws {}
    
    func uploadProjectCover(projectID: UUID, imageData: Data, accessToken: String, altText: String?) async throws {}
}
