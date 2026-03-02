import Foundation

@MainActor
public final class AppContainer: ObservableObject {
    public let authContainer: AuthContainer
    public var authService: AuthService { authContainer.authService }
    public var authViewModel: AuthViewModel { authContainer.authViewModel }
    public let apiClient: APIClient
    public let signedURLCache: SignedURLCache

    public init() {
        let vault = SecureEnclaveKeychainVault()

        guard
            let supabaseURLRaw = Bundle.main.object(forInfoDictionaryKey: "SUPABASE_URL") as? String,
            let supabaseURL = URL(string: supabaseURLRaw),
            let supabaseAnonKey = Bundle.main.object(forInfoDictionaryKey: "SUPABASE_ANON_KEY") as? String,
            !supabaseAnonKey.isEmpty,
            let apiBaseURLRaw = Bundle.main.object(forInfoDictionaryKey: "PURPTAPE_API_BASE_URL") as? String,
            let apiBaseURL = URL(string: apiBaseURLRaw)
        else {
            fatalError("Missing required runtime configuration for Supabase/API.")
        }

        let authService = SupabaseAuthService(
            supabaseURL: supabaseURL,
            supabaseAnonKey: supabaseAnonKey,
            vault: vault
        )
        self.authContainer = AuthContainer(authService: authService, vault: vault)
        self.apiClient = URLSessionAPIClient(baseURL: apiBaseURL, authService: authService)
        self.signedURLCache = InMemorySignedURLCache()
    }
}
