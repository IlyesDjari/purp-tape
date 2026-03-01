import SwiftUI

struct ProjectsTabContent: View {
    @ObservedObject var authViewModel: AuthViewModel
    @Environment(\.navigationManager) var navigationManager
    @StateObject private var viewModel = ProjectsTabViewModel()
    @State private var showCreateProjectSheet = false
    @State private var createSheetHeight: CGFloat = 360
    @State private var topNotification: InAppTopNotification?

    var body: some View {
        GeometryReader { proxy in
            let headerHeight = proxy.size.height * 0.5

            ZStack(alignment: .top) {
                PurpTapeColors.primary
                    .frame(height: headerHeight)
                    .clipShape(
                        UnevenRoundedRectangle(
                            cornerRadii: .init(
                                topLeading: 0,
                                bottomLeading: 50,
                                bottomTrailing: 50,
                                topTrailing: 0
                            )
                        )
                    )
                    .ignoresSafeArea(edges: .top)

                ScrollView(.vertical, showsIndicators: false) {
                    VStack(alignment: .leading, spacing: Spacing.lg) {
                        HStack {
                            Text("Projects")
                                .font(PurpTapeTypography.displayLarge)
                                .foregroundStyle(.white)

                            Spacer()
                        }

                        if viewModel.isLoading {
                            ProjectsLoadingCarouselView()
                                .padding(.top, Spacing.md)
                        } else {
                            projectCarousel
                                .padding(.top, Spacing.md)
                        }

                        if let errorMessage = viewModel.errorMessage {
                            Text(errorMessage)
                                .font(PurpTapeTypography.bodySmall)
                                .foregroundColor(.white.opacity(0.9))
                        }

                        Spacer(minLength: 0)
                    }
                    .paddingHorizontalLG()
                    .padding(.top, Spacing.xl)
                    .padding(.bottom, Spacing.xl)
                    .frame(maxWidth: .infinity, minHeight: proxy.size.height, alignment: .top)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .background(PurpTapeColors.background)
            .task {
                let accessToken = await resolveAccessToken()
                await viewModel.loadProjects(accessToken: accessToken)
            }
            .sheet(isPresented: $showCreateProjectSheet) {
                CreateProjectSheet(
                    isLoading: viewModel.isCreating,
                    onCreate: { name, description, isPublic, artworkData in
                        let accessToken = await resolveAccessToken()
                        let created = await viewModel.createProject(
                            name: name,
                            description: description,
                            isPublic: isPublic,
                            artworkData: artworkData,
                            accessToken: accessToken
                        )
                        if created {
                            showCreateProjectSheet = false
                            withAnimation(.spring(response: 0.32, dampingFraction: 0.82)) {
                                topNotification = .success("Project created")
                            }
                        }
                    },
                    onHeightChange: { height in
                        Task { @MainActor in
                            createSheetHeight = height
                        }
                    }
                )
                .presentationDetents([.height(max(300, createSheetHeight + 30))])
                .presentationDragIndicator(.visible)
            }
            .onChange(of: authViewModel.currentSession?.accessToken) { _, token in
                guard let token else { return }
                Task {
                    await viewModel.loadProjects(accessToken: token)
                }
            }
            .onChange(of: viewModel.errorMessage) { _, message in
                guard let message, !message.isEmpty else { return }
                withAnimation(.spring(response: 0.32, dampingFraction: 0.82)) {
                    topNotification = .error(message)
                }
            }
            .inAppTopNotification(notification: $topNotification, duration: 3)
        }
    }

    private var projectCarousel: some View {
        TabView {
            ForEach(viewModel.projects) { project in
                ProjectCardView(
                    project: project,
                    artworkData: viewModel.projectCoverData[project.id]
                )
                    .contentShape(Rectangle())
                    .onTapGesture {
                        navigationManager.showProjectDetail(project.id)
                    }
                    .padding(.horizontal, 2)
            }

            AddProjectCardView {
                showCreateProjectSheet = true
            }
            .padding(.horizontal, 2)
        }
        .frame(height: 360)
        .tabViewStyle(.page(indexDisplayMode: .automatic))
        .indexViewStyle(.page(backgroundDisplayMode: .always))
    }

    private func resolveAccessToken() async -> String? {
        if authViewModel.isAuthenticated {
            await authViewModel.refreshSessionIfNeeded()
            if let token = authViewModel.currentSession?.accessToken, !token.isEmpty {
                return token
            }
        }

        if let token = authViewModel.currentSession?.accessToken, !token.isEmpty {
            return authViewModel.currentSession?.accessToken
        }

        return nil
    }
}
