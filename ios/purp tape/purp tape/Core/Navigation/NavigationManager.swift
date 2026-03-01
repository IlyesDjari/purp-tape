import Foundation
import SwiftUI

// MARK: - Navigation Routes

public enum NavigationRoute: Hashable, Codable, Identifiable {
    // Auth routes
    case signIn
    case signUp
    case forgotPassword
    case resetPassword(token: String)
    
    // Main app routes
    case projects
    case projectDetail(id: UUID)
    case discover
    case profile
    case settings
    
    // Modal routes
    case createProject
    case uploadTrack(projectID: UUID)
    case sharing(projectID: UUID)
    
    public var id: String {
        switch self {
        case .signIn:
            return "signIn"
        case .signUp:
            return "signUp"
        case .forgotPassword:
            return "forgotPassword"
        case .resetPassword(let token):
            return "resetPassword-\(token)"
        case .projects:
            return "projects"
        case .projectDetail(let id):
            return "projectDetail-\(id)"
        case .discover:
            return "discover"
        case .profile:
            return "profile"
        case .settings:
            return "settings"
        case .createProject:
            return "createProject"
        case .uploadTrack(let projectID):
            return "uploadTrack-\(projectID)"
        case .sharing(let projectID):
            return "sharing-\(projectID)"
        }
    }
    
    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .signIn:
            try container.encode("signIn", forKey: .type)
        case .signUp:
            try container.encode("signUp", forKey: .type)
        case .forgotPassword:
            try container.encode("forgotPassword", forKey: .type)
        case .resetPassword(let token):
            try container.encode("resetPassword", forKey: .type)
            try container.encode(token, forKey: .token)
        case .projects:
            try container.encode("projects", forKey: .type)
        case .projectDetail(let id):
            try container.encode("projectDetail", forKey: .type)
            try container.encode(id, forKey: .id)
        case .discover:
            try container.encode("discover", forKey: .type)
        case .profile:
            try container.encode("profile", forKey: .type)
        case .settings:
            try container.encode("settings", forKey: .type)
        case .createProject:
            try container.encode("createProject", forKey: .type)
        case .uploadTrack(let projectID):
            try container.encode("uploadTrack", forKey: .type)
            try container.encode(projectID, forKey: .projectID)
        case .sharing(let projectID):
            try container.encode("sharing", forKey: .type)
            try container.encode(projectID, forKey: .projectID)
        }
    }
    
    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let type = try container.decode(String.self, forKey: .type)
        
        switch type {
        case "signIn":
            self = .signIn
        case "signUp":
            self = .signUp
        case "forgotPassword":
            self = .forgotPassword
        case "resetPassword":
            let token = try container.decode(String.self, forKey: .token)
            self = .resetPassword(token: token)
        case "projects":
            self = .projects
        case "projectDetail":
            let id = try container.decode(UUID.self, forKey: .id)
            self = .projectDetail(id: id)
        case "discover":
            self = .discover
        case "profile":
            self = .profile
        case "settings":
            self = .settings
        case "createProject":
            self = .createProject
        case "uploadTrack":
            let projectID = try container.decode(UUID.self, forKey: .projectID)
            self = .uploadTrack(projectID: projectID)
        case "sharing":
            let projectID = try container.decode(UUID.self, forKey: .projectID)
            self = .sharing(projectID: projectID)
        default:
            self = .projects
        }
    }
    
    enum CodingKeys: String, CodingKey {
        case type, token, id, projectID
    }
}

// MARK: - Tab Selection

public enum TabSelection: String, Hashable {
    case projects
    case discover
    case stats
    case profile
}

// MARK: - Navigation Manager

@MainActor
public final class NavigationManager: NSObject, ObservableObject {
    @Published public var navigationPath = NavigationPath()
    @Published public var selectedTab: TabSelection = .projects
    @Published public var presentedSheet: NavigationRoute?
    @Published public var presentedAlert: AlertConfig?
    
    public override init() {
        super.init()
    }
    
    // MARK: - Navigation Actions
    
    public func navigate(to route: NavigationRoute) {
        navigationPath.append(route)
    }
    
    public func navigateToRoot() {
        navigationPath = NavigationPath()
    }
    
    public func popToRoot() {
        navigationPath = NavigationPath()
    }
    
    public func pop() {
        navigationPath.removeLast()
    }
    
    public func selectTab(_ tab: TabSelection) {
        selectedTab = tab
        navigationPath = NavigationPath()
    }
    
    public func presentSheet(_ route: NavigationRoute) {
        presentedSheet = route
    }
    
    public func dismissSheet() {
        presentedSheet = nil
    }
    
    public func presentAlert(_ config: AlertConfig) {
        presentedAlert = config
    }
    
    public func dismissAlert() {
        presentedAlert = nil
    }
    
    // MARK: - Deep Linking
    
    public func handleDeepLink(_ url: URL) {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: true) else {
            return
        }
        
        let path = components.path
        let queryItems = components.queryItems ?? []
        let pathComponents = path.split(separator: "/").map(String.init)
        
        switch pathComponents.first {
        case "projects":
            if pathComponents.count > 1, let projectID = UUID(uuidString: pathComponents[1]) {
                selectTab(.projects)
                navigate(to: .projectDetail(id: projectID))
            } else {
                selectTab(.projects)
            }
        case "discover":
            selectTab(.discover)
        case "profile":
            selectTab(.profile)
        case "reset-password":
            if let token = queryItems.first(where: { $0.name == "token" })?.value {
                navigate(to: .resetPassword(token: token))
            }
        default:
            break
        }
    }
    
    // MARK: - Quick Navigation Helpers
    
    public func goToProjects() {
        selectTab(.projects)
    }
    
    public func goToDiscover() {
        selectTab(.discover)
    }
    
    public func goToProfile() {
        selectTab(.profile)
    }

    public func goToStats() {
        selectTab(.stats)
    }
    
    public func showProjectDetail(_ projectID: UUID) {
        selectTab(.projects)
        navigate(to: .projectDetail(id: projectID))
    }
    
    public func showCreateProject() {
        presentSheet(.createProject)
    }
    
    public func showUploadTrack(for projectID: UUID) {
        presentSheet(.uploadTrack(projectID: projectID))
    }
    
    public func showSharing(for projectID: UUID) {
        presentSheet(.sharing(projectID: projectID))
    }
    
    public func goToForgotPassword() {
        navigate(to: .forgotPassword)
    }
    
    public func goToSignUp() {
        navigate(to: .signUp)
    }
}

// MARK: - Alert Configuration

public struct AlertConfig: Identifiable {
    public let id = UUID()
    public let title: String
    public let message: String?
    public let primaryButton: AlertButton
    public let secondaryButton: AlertButton?
    
    public init(
        title: String,
        message: String? = nil,
        primaryButton: AlertButton,
        secondaryButton: AlertButton? = nil
    ) {
        self.title = title
        self.message = message
        self.primaryButton = primaryButton
        self.secondaryButton = secondaryButton
    }
}

public struct AlertButton {
    public enum Role {
        case `default`
        case destructive
        case cancel
    }
    
    public let label: String
    public let role: Role
    public let action: (() -> Void)?
    
    public init(label: String, role: Role = .default, action: (() -> Void)? = nil) {
        self.label = label
        self.role = role
        self.action = action
    }
}

// MARK: - Environment Key

struct NavigationManagerKey: EnvironmentKey {
    static let defaultValue: NavigationManager = {
        MainActor.assumeIsolated {
            NavigationManager()
        }
    }()
}

public extension EnvironmentValues {
    var navigationManager: NavigationManager {
        get { self[NavigationManagerKey.self] }
        set { self[NavigationManagerKey.self] = newValue }
    }
}

public extension View {
    func withNavigationManager(_ manager: NavigationManager) -> some View {
        environment(\.navigationManager, manager)
    }
}
