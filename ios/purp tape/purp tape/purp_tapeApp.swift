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
    @StateObject private var appContainer = AppContainer()
    @State private var navigationManager = NavigationManager()
    
    init() {
        AppTelemetry.shared.start()
    }

    var body: some Scene {
        WindowGroup {
            RootView(authViewModel: appContainer.authViewModel)
                .environment(\.navigationManager, navigationManager)
                .environmentObject(appContainer)
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
