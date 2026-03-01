import SwiftUI

// MARK: - Navigation Examples & Usage Patterns

struct NavigationExamples {
    static let basicNavigationExample = """
    struct ProjectsView: View {
        @Environment(\\.navigationManager) var navigationManager
        
        var body: some View {
            Button("View Project Details") {
                navigationManager.showProjectDetail(projectID)
            }
            
            Button("Create New Project") {
                navigationManager.showCreateProject()
            }
        }
    }
    """
    
    static let tabNavigationExample = """
    struct MainTabView: View {
        @Environment(\\.navigationManager) var navigationManager
        
        var body: some View {
            TabView(selection: $navigationManager.selectedTab) {
                ProjectsTabContent()
                    .tabItem { Label("Projects", systemImage: "music.note.list") }
                    .tag(TabSelection.projects)
                
                DiscoverTabContent()
                    .tabItem { Label("Discover", systemImage: "sparkles") }
                    .tag(TabSelection.discover)
                
                ProfileTabContent()
                    .tabItem { Label("Profile", systemImage: "person.circle") }
                    .tag(TabSelection.profile)
            }
        }
    }
    """
    
    static let deepLinkingExample = """
    // purptape://projects/123abc-def
    // purptape://discover
    // purptape://reset-password?token=xyz123
    """
}

extension NavigationManager {
    public func transitionFromAuthToApp() {
        navigationPath = NavigationPath()
        selectedTab = .projects
    }
    
    public func handleSearchSelection(projectID: UUID) {
        selectTab(.projects)
        navigate(to: .projectDetail(id: projectID))
    }
    
    public func handleNotificationTap(deeplinkURL: URL) {
        handleDeepLink(deeplinkURL)
    }
}

public extension View {
    func showAlert(title: String, message: String?) {
        let manager = EnvironmentValues().navigationManager
        Task { @MainActor in
            let alert = AlertConfig(
                title: title,
                message: message,
                primaryButton: AlertButton(label: "OK")
            )
            manager.presentAlert(alert)
        }
    }
}

public extension NavigationManager {
    func resetForTesting() {
        navigationPath = NavigationPath()
        selectedTab = .projects
        presentedSheet = nil
        presentedAlert = nil
    }
    
    var isAtRoot: Bool {
        navigationPath.isEmpty && selectedTab == .projects
    }
}
