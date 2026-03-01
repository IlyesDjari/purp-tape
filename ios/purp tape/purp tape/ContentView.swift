//
//  ContentView.swift
//  purp tape
//
//  Created by Ilyes Djari on 28/02/2026.
//

import SwiftUI

/// Root navigation view that shows either auth flow or main app based on authentication state
struct RootView: View {
    @ObservedObject var authViewModel: AuthViewModel
    @StateObject private var authNavigationManager = NavigationManager()
    
    var body: some View {
        ZStack {
            if !authViewModel.didResolveInitialSession {
                VStack(spacing: Spacing.md) {
                    ProgressView()
                    Text("Loading...")
                        .bodyMedium()
                        .foregroundColor(PurpTapeColors.textSecondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(PurpTapeColors.background)
            } else if authViewModel.isAuthenticated {
                MainAppView(authViewModel: authViewModel)
            } else {
                NavigationStack(path: $authNavigationManager.navigationPath) {
                    SignInView(viewModel: authViewModel)
                        .navigationDestination(for: NavigationRoute.self) { route in
                            switch route {
                            case .signUp:
                                SignUpView(viewModel: authViewModel)
                            case .forgotPassword:
                                ForgotPasswordView(viewModel: authViewModel)
                            case .resetPassword(let token):
                                PasswordResetView(viewModel: authViewModel, resetToken: token)
                            default:
                                EmptyView()
                            }
                        }
                }
                .environment(\.navigationManager, authNavigationManager)
            }
        }
        .onChange(of: authViewModel.isAuthenticated) { _, isAuth in
            if isAuth {
                authNavigationManager.popToRoot()
            } else {
                authNavigationManager.popToRoot()
            }
        }
        .task {
            await MainActor.run {
                authViewModel.startSessionMonitoring()
            }
        }
    }
}

/// Main app content after authentication
struct MainAppView: View {
    @ObservedObject var authViewModel: AuthViewModel
    @StateObject private var navigationManager = NavigationManager()
    @Namespace private var tabAnimation
    
    var body: some View {
        NavigationStack(path: $navigationManager.navigationPath) {
            VStack(spacing: 0) {
                ZStack {
                    ProjectsTabContent(authViewModel: authViewModel)
                        .opacity(navigationManager.selectedTab == .projects ? 1 : 0)
                        .zIndex(navigationManager.selectedTab == .projects ? 1 : 0)
                        .allowsHitTesting(navigationManager.selectedTab == .projects)

                    DiscoverTabContent()
                        .opacity(navigationManager.selectedTab == .discover ? 1 : 0)
                        .zIndex(navigationManager.selectedTab == .discover ? 1 : 0)
                        .allowsHitTesting(navigationManager.selectedTab == .discover)

                    StatsTabContent()
                        .opacity(navigationManager.selectedTab == .stats ? 1 : 0)
                        .zIndex(navigationManager.selectedTab == .stats ? 1 : 0)
                        .allowsHitTesting(navigationManager.selectedTab == .stats)

                    ProfileTabContent(authViewModel: authViewModel)
                        .opacity(navigationManager.selectedTab == .profile ? 1 : 0)
                        .zIndex(navigationManager.selectedTab == .profile ? 1 : 0)
                        .allowsHitTesting(navigationManager.selectedTab == .profile)
                }
                .animation(.easeInOut(duration: 0.22), value: navigationManager.selectedTab)
                .frame(maxWidth: .infinity, maxHeight: .infinity)

                PurpTapeAnimatedTabBar(
                    selectedTab: $navigationManager.selectedTab,
                    namespace: tabAnimation
                )
            }
            .navigationDestination(for: NavigationRoute.self) { route in
                NavigationDestinationView(route: route)
            }
        }
        .environment(\.navigationManager, navigationManager)
        .sheet(item: $navigationManager.presentedSheet) { route in
            SheetDestinationView(route: route)
        }
        .alert("Alert", isPresented: .constant(navigationManager.presentedAlert != nil)) {
            if let alert = navigationManager.presentedAlert {
                buildAlert(alert)
            }
        } message: {
            if let alert = navigationManager.presentedAlert, let message = alert.message {
                Text(message)
            }
        }
    }
    
    @ViewBuilder
    private func buildAlert(_ config: AlertConfig) -> some View {
        Button(config.primaryButton.label) {
            config.primaryButton.action?()
            navigationManager.dismissAlert()
        }
        
        if let secondary = config.secondaryButton {
            Button(secondary.label) {
                secondary.action?()
                navigationManager.dismissAlert()
            }
        }
    }
}

// MARK: - Navigation Destinations

struct NavigationDestinationView: View {
    let route: NavigationRoute
    
    @ViewBuilder
    var body: some View {
        switch route {
        case .projectDetail(let id):
            Text("Project Details: \(id)")
        case .settings:
            Text("Settings")
        default:
            EmptyView()
        }
    }
}

struct SheetDestinationView: View {
    let route: NavigationRoute
    @Environment(\.dismiss) var dismiss
    
    @ViewBuilder
    var body: some View {
        switch route {
        case .createProject:
            Text("Create Project")
        case .uploadTrack(let projectID):
            Text("Upload Track to \(projectID)")
        case .sharing(let projectID):
            Text("Share Project \(projectID)")
        default:
            EmptyView()
        }
    }
}

private struct TabBarItemModel: Identifiable {
    let tab: TabSelection
    let title: String
    let icon: String

    var id: String { tab.rawValue }
}

private struct PurpTapeAnimatedTabBar: View {
    @Binding var selectedTab: TabSelection
    let namespace: Namespace.ID

    private let items: [TabBarItemModel] = [
        .init(tab: .projects, title: "Projects", icon: "music.note.list"),
        .init(tab: .discover, title: "Discover", icon: "sparkles"),
        .init(tab: .stats, title: "Stats", icon: "chart.bar.fill"),
        .init(tab: .profile, title: "Profile", icon: "person.circle")
    ]

    var body: some View {
        HStack(spacing: Spacing.sm) {
            ForEach(items) { item in
                sideItem(for: item)
            }
        }
        .padding(.horizontal, Spacing.md)
        .padding(.top, Spacing.sm)
        .padding(.bottom, Spacing.md)
        .background(PurpTapeColors.surface)
        .sensoryFeedback(.selection, trigger: selectedTab)
        .shadow(color: PurpTapeColors.shadowLight, radius: 10, x: 0, y: -1)
    }

    @ViewBuilder
    private func sideItem(for item: TabBarItemModel) -> some View {
        Button {
            guard selectedTab != item.tab else { return }
            withAnimation(.spring(response: 0.3, dampingFraction: 0.82)) {
                selectedTab = item.tab
            }
        } label: {
            VStack(spacing: 6) {
                Image(systemName: item.icon)
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundColor(selectedTab == item.tab ? PurpTapeColors.primary : PurpTapeColors.textSecondary)

                Text(item.title)
                    .font(PurpTapeTypography.labelSmall)
                    .foregroundColor(selectedTab == item.tab ? PurpTapeColors.primary : PurpTapeColors.textSecondary)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 10)
            .background(
                Group {
                    if selectedTab == item.tab {
                        RoundedRectangle(cornerRadius: CornerRadius.md)
                            .fill(PurpTapeColors.primary.opacity(0.12))
                            .matchedGeometryEffect(id: "tabSelection", in: namespace)
                    }
                }
            )
        }
        .buttonStyle(PurpTapeInteractiveButtonStyle())
    }
}

struct ContentView: View {
    var body: some View {
        VStack {
            Image(systemName: "globe")
                .imageScale(.large)
                .foregroundStyle(.tint)
            Text(String(localized: "home.welcome", defaultValue: "Hello, world!"))
        }
        .padding()
    }
}

#Preview {
    ContentView()
}
