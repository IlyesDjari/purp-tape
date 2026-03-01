import SwiftUI
import Supabase

/// Example Integration in PurpTapeApp
/// This shows how to set up auth in your app entry point

struct AuthIntegrationExample {
    // MARK: - Setup Steps
    
    /// Step 1: Configure in App Entry Point
    /// Add this to your PurpTapeApp.swift or main app delegate
    static let appSetupExample = """
    import SwiftUI
    import Supabase

    @main
    struct PurpTapeApp: App {
        @State private var authContainer: AuthContainer?
        
        init() {
            // Get configuration from xcconfig files
            guard let supabaseURL = URL(string: Config.supabaseURL) else {
                fatalError("Invalid Supabase URL")
            }
            
            // Initialize Keychain vault for secure token storage
            let vault = SecureEnclaveKeychainVault()
            
            // Setup Supabase auth service
            let authService = SupabaseAuthService(
                supabaseURL: supabaseURL,
                supabaseAnonKey: Config.supabaseAnonKey,
                vault: vault
            )
            
            // Initialize auth container (dependency injection)
            self.authContainer = AuthContainer(
                authService: authService,
                vault: vault
            )
        }
        
        var body: some Scene {
            WindowGroup {
                if let container = authContainer {
                    RootView(authViewModel: container.authViewModel)
                } else {
                    ProgressView("Initializing...")
                }
            }
        }
    }
    """
    
    /// Step 2: Root Navigation View
    static let rootViewExample = """
    import SwiftUI

    struct RootView: View {
        @ObservedObject var authViewModel: AuthViewModel
        
        var body: some View {
            ZStack {
                if authViewModel.isAuthenticated {
                    TabView {
                        ProjectsView()
                            .tabItem {
                                Label("Projects", systemImage: "music.note.list")
                            }
                        
                        DiscoverView()
                            .tabItem {
                                Label("Discover", systemImage: "sparkles")
                            }
                        
                        ProfileView()
                            .tabItem {
                                Label("Profile", systemImage: "person")
                            }
                    }
                } else {
                    AuthNavigationStack(viewModel: authViewModel)
                }
            }
            .onAppear {
                // Start monitoring session changes
                authViewModel.startSessionMonitoring()
            }
            .onDisappear {
                // Clean up when app goes to background
                authViewModel.stopSessionMonitoring()
            }
        }
    }
    """
    
    /// Step 3: Auth Navigation Stack
    static let authNavigationExample = """
    import SwiftUI

    struct AuthNavigationStack: View {
        @ObservedObject var viewModel: AuthViewModel
        @State private var navigationPath = NavigationPath()
        
        var body: some View {
            NavigationStack(path: $navigationPath) {
                SignInView(viewModel: viewModel)
                    .navigationDestination(for: AuthRoute.self) { route in
                        switch route {
                        case .signup:
                            SignUpView(viewModel: viewModel)
                        case .forgotPassword:
                            ForgotPasswordView(viewModel: viewModel)
                        case let .resetPassword(token):
                            PasswordResetView(viewModel: viewModel, resetToken: token)
                        }
                    }
            }
        }
    }
    
    enum AuthRoute: Hashable {
        case signup
        case forgotPassword
        case resetPassword(String)
    }
    """
    
    /// Step 4: Using Auth in Feature Views
    static let featureViewExample = """
    import SwiftUI

    struct ProjectsView: View {
        @ObservedObject var authViewModel: AuthViewModel
        @State private var projects: [Project] = []
        @State private var isLoading = false
        @State private var error: Error?
        
        var body: some View {
            NavigationStack {
                List {
                    ForEach(projects) { project in
                        NavigationLink(destination: ProjectDetailView(project: project)) {
                            ProjectListItem(project: project)
                        }
                    }
                }
                .navigationTitle("My Projects")
                .toolbar {
                    ToolbarItem(placement: .topBarTrailing) {
                        Menu {
                            Button("Sign Out") {
                                Task {
                                    await authViewModel.signOut()
                                }
                            }
                        } label: {
                            Image(systemName: "ellipsis.circle")
                        }
                    }
                }
                .onAppear {
                    loadProjects()
                }
            }
        }
        
        private func loadProjects() {
            isLoading = true
            
            // Use auth token from viewModel
            guard let token = authViewModel.currentSession?.accessToken else {
                error = AuthError.sessionExpired
                return
            }
            
            // Make API request with Bearer token
            Task {
                do {
                    let apiClient = URLSessionAPIClient(token: token)
                    self.projects = try await apiClient.send(
                        .getProjects,
                        decode: [Project].self
                    )
                } catch {
                    self.error = error
                }
                isLoading = false
            }
        }
    }
    """
    
    /// Step 5: Refresh Token on App Resume
    static let backgroundHandlingExample = """
    import SwiftUI

    extension View {
        func withAuthRefresh(_ viewModel: AuthViewModel) -> some View {
            self.onReceive(
                NotificationCenter.default.publisher(
                    for: UIApplication.willEnterForegroundNotification
                )
            ) { _ in
                Task {
                    await viewModel.refreshSessionIfNeeded()
                }
            }
        }
    }
    """
    
    /// Step 6: Configuration File (Secrets.xcconfig)
    static let configExample = """
    // ios/purp tape/Secrets.xcconfig
    SUPABASE_URL = https://your-project.supabase.co
    SUPABASE_ANON_KEY = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
    SUPABASE_BUCKET_ID = audio-files
    API_BASE_URL = https://purptape-api.fly.dev
    """
}

// MARK: - Configuration Helper

struct Config {
    static var supabaseURL: String {
        guard let url = Bundle.main.infoDictionary?["SUPABASE_URL"] as? String else {
            fatalError("SUPABASE_URL not configured in xcconfig")
        }
        return url
    }
    
    static var supabaseAnonKey: String {
        guard let key = Bundle.main.infoDictionary?["SUPABASE_ANON_KEY"] as? String else {
            fatalError("SUPABASE_ANON_KEY not configured in xcconfig")
        }
        return key
    }
    
    static var apiBaseURL: String {
        guard let url = Bundle.main.infoDictionary?["API_BASE_URL"] as? String else {
            return "https://purptape-api.fly.dev"
        }
        return url
    }
}

// MARK: - Dependency Injection Extension

extension AuthContainer {
    /// Create a production container with all services
    static func production(vault: KeychainVault = SecureEnclaveKeychainVault()) -> AuthContainer {
        guard let supabaseURL = URL(string: Config.supabaseURL) else {
            fatalError("Invalid Supabase URL")
        }
        
        let authService = SupabaseAuthService(
            supabaseURL: supabaseURL,
            supabaseAnonKey: Config.supabaseAnonKey,
            vault: vault
        )
        
        return AuthContainer(authService: authService, vault: vault)
    }
    
    /// Create a test container with mock services
    static func test() -> AuthContainer {
        let mockService = MockAuthService()
        _ = MockAuthRepository()
        
        return AuthContainer(authService: mockService)
    }
}

// MARK: - Mock for Integration Tests

private actor MockAuthRepository: AuthRepository {
    func signUp(request: SignUpRequest) async throws -> AuthSession {
        throw AuthError.unknownError("Mock")
    }
    
    func signIn(request: SignInRequest) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    
    func signOut() async throws {}
    func currentSession() async throws -> AuthSession? { nil }
    func refreshSessionIfNeeded() async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func requestPasswordReset(email: String) async throws {}
    func resetPassword(token: String, newPassword: String) async throws {}
    func signInWithApple(token: String, nonce: String) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    nonisolated func validateEmail(_ email: String) throws {}
    nonisolated func validatePassword(_ password: String) throws {}
}

private actor MockAuthService: AuthService {
    func currentSession() async throws -> AuthSession? { nil }
    func signIn(email: String, password: String) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func signInWithApple(idToken: String, nonce: String?) async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
    func signOut() async throws {}
    func refreshIfNeeded() async throws -> AuthSession {
        AuthSession(
            accessToken: "mock_token",
            refreshToken: "mock_refresh",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(3600)
        )
    }
}
