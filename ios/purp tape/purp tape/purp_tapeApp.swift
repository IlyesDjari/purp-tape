//
//  purp_tapeApp.swift
//  purp tape
//
//  Created by Ilyes Djari on 28/02/2026.
//

import SwiftUI
import Supabase

private let logger = DebugLogger(category: "app.init")

@main
struct purp_tapeApp: App {
    @State private var authContainer: AuthContainer? = purp_tapeApp.makeAuthContainer()
    @State private var navigationManager = NavigationManager()
    
    init() {
        AppTelemetry.shared.start()
    }
    
    private static func makeAuthContainer() -> AuthContainer? {
        // Get configuration from app bundle Info.plist
        guard let supabaseURLString = Bundle.main.infoDictionary?["SUPABASE_URL"] as? String,
              !supabaseURLString.trimmingCharacters(in: .whitespaces).isEmpty else {
            logger.error("Supabase URL not configured in Info.plist")
            return nil
        }
        
        guard let supabaseURL = URL(string: supabaseURLString),
              supabaseURL.scheme != nil,
              supabaseURL.host != nil else {
            logger.error("Invalid Supabase URL format: \(supabaseURLString)")
            logger.warning("Expected format: https://xxxxx.supabase.co")
            return nil
        }
        
        guard let supabaseAnonKey = Bundle.main.infoDictionary?["SUPABASE_ANON_KEY"] as? String,
              !supabaseAnonKey.isEmpty else {
            logger.error("Supabase Anon Key not configured in Info.plist")
            return nil
        }
        
        logger.success("Supabase configuration loaded successfully")
        
        // Initialize Keychain vault for secure token storage
        let vault = SecureEnclaveKeychainVault()
        
        // Setup Supabase auth service
        let authService = SupabaseAuthService(
            supabaseURL: supabaseURL,
            supabaseAnonKey: supabaseAnonKey,
            vault: vault
        )
        
        logger.success("Auth service initialized")
        
        // Initialize auth container (dependency injection)
        return AuthContainer(authService: authService, vault: vault)
    }

    private func setupAuth() {
        authContainer = Self.makeAuthContainer()
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if let container = authContainer {
                    RootView(authViewModel: container.authViewModel)
                        .environment(\.navigationManager, navigationManager)
                } else {
                    ErrorView(message: "Failed to initialize app", action: setupAuth)
                }
            }
            .preferredColorScheme(.light)
        }
    }
}

// MARK: - Error View

private struct ErrorView: View {
    let message: String
    let action: () -> Void
    
    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 40))
                .foregroundColor(.red)
            
            Text("Initialization Error")
                .font(.headline)
            
            Text(message)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
            
            Button("Retry", action: action)
                .buttonStyle(.borderedProminent)
        }
        .padding()
    }
}
