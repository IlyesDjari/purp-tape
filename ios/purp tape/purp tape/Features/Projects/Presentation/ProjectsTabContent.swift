import SwiftUI

struct ProjectsTabContent: View {
    @Environment(\.navigationManager) var navigationManager
    @EnvironmentObject private var appContainer: AppContainer
    
    @ObservedObject var authViewModel: AuthViewModel
    let playbackController: TrackPlaybackController
    let onActiveTrackChange: (Track?) -> Void
    @StateObject private var viewModel: ProjectsTabViewModel
    @State private var showCreateProjectSheet = false
    @State private var createSheetHeight: CGFloat = 360
    @State private var topNotification: InAppTopNotification?
    @State private var hasLoadedInitially = false
    @State private var activeCarouselID: String?
    @State private var deleteArmedProjectID: UUID?
    @State private var isTrackListOverflowing = false
    @State private var hasScrolledDownForCompactHeader = false
    @Namespace private var projectHeaderNamespace
    
    private let carouselCardWidth: CGFloat = 230
    private let carouselTransitionDistance: CGFloat = 300
    
    init(
        authViewModel: AuthViewModel,
        playbackController: TrackPlaybackController,
        onActiveTrackChange: @escaping (Track?) -> Void
    ) {
        self.authViewModel = authViewModel
        self.playbackController = playbackController
        self.onActiveTrackChange = onActiveTrackChange
        _viewModel = StateObject(wrappedValue: ProjectsTabViewModel(authService: authViewModel.service))
    }
    
    var body: some View {
        mainContent()
    }
    
    @ViewBuilder
    private func mainContent() -> some View {
        zstackCore()
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .background(PurpTapeColors.background)
            .task {
                guard !hasLoadedInitially else { return }
                hasLoadedInitially = true
                let accessToken = await resolveAccessToken()
                await viewModel.loadProjects(accessToken: accessToken)
            }
            .sheet(isPresented: $showCreateProjectSheet) {
                createProjectSheet
            }
            .onChange(of: viewModel.errorMessage) { _, message in
                guard let message, !message.isEmpty else { return }
                withAnimation(.spring(response: 0.32, dampingFraction: 0.82)) {
                    topNotification = .error(message)
                }
            }
            .inAppTopNotification(notification: $topNotification, duration: 3)
    }
    
    @ViewBuilder
    private func zstackCore() -> some View {
        let headerHeight: CGFloat = 400
        let topBleed: CGFloat = 140
        let topContentPadding: CGFloat = 64
        let compactHeaderScrollThreshold: CGFloat = 64

        ScrollView(.vertical, showsIndicators: false) {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Color.clear
                    .frame(height: 0)
                    .background(
                        GeometryReader { proxy in
                            Color.clear.preference(
                                key: ProjectsHeaderScrollOffsetKey.self,
                                value: proxy.frame(in: .named("projectsScroll")).minY
                            )
                        }
                    )
                headerSection
                projectsSection()
                errorSection
                Spacer(minLength: 0)
            }
            .paddingHorizontalLG()
            .padding(.top, topContentPadding)
            .padding(.bottom, Spacing.xl)
            .frame(maxWidth: .infinity, alignment: .top)
            .background(alignment: .top) {
                PurpTapeColors.primary
                    .frame(height: headerHeight + topBleed)
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
                    .shadow(color: PurpTapeColors.primary.opacity(0.18), radius: 32, x: 0, y: 18)
                    .offset(y: -(topContentPadding + topBleed))
            }
        }
        .ignoresSafeArea(edges: .top)
        .coordinateSpace(name: "projectsScroll")
        .onPreferenceChange(ProjectsHeaderScrollOffsetKey.self) { minY in
            let shouldCollapse = minY < -compactHeaderScrollThreshold
            Task { @MainActor in
                if shouldCollapse != hasScrolledDownForCompactHeader {
                    withAnimation(.spring(response: 0.42, dampingFraction: 0.84)) {
                        hasScrolledDownForCompactHeader = shouldCollapse
                    }
                }
            }
        }
        .refreshable {
            let accessToken = await resolveAccessToken()
            await viewModel.loadProjects(accessToken: accessToken)
        }
    }
    
    private var createProjectSheet: some View {
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
    
    @ViewBuilder
    private func projectsSection() -> some View {
        if viewModel.isLoading {
            ProjectsLoadingCarouselView()
                .padding(.top, Spacing.md)
        } else {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                adaptiveProjectHeader
                activeProjectTrackListSection
            }
        }
    }

    @ViewBuilder
    private var adaptiveProjectHeader: some View {
        Group {
            if shouldShowCompactHeader {
                compactProjectHeader
                    .transition(
                        .asymmetric(
                            insertion: .move(edge: .top).combined(with: .opacity).combined(with: .scale(scale: 0.86, anchor: .top)),
                            removal: .opacity.combined(with: .scale(scale: 0.92, anchor: .top))
                        )
                    )
            } else {
                projectCarousel(containerWidth: UIScreen.main.bounds.width)
                    .padding(.top, Spacing.md)
                    .transition(
                        .asymmetric(
                            insertion: .opacity.combined(with: .scale(scale: 1.04, anchor: .top)),
                            removal: .move(edge: .top).combined(with: .opacity)
                        )
                    )
            }
        }
        .animation(.spring(response: 0.62, dampingFraction: 0.74, blendDuration: 0.2), value: isTrackListOverflowing)
    }

    private var shouldShowCompactHeader: Bool {
        isTrackListOverflowing && hasScrolledDownForCompactHeader
    }
    
    @ViewBuilder
    private var errorSection: some View {
        if let errorMessage = viewModel.errorMessage {
            Text(errorMessage)
                .font(PurpTapeTypography.bodySmall)
                .foregroundColor(.white.opacity(0.9))
        }
    }
    
    private var headerSection: some View {
        HStack {
            Text("Projects")
                .font(PurpTapeTypography.displayLarge)
                .foregroundStyle(.white)
            Spacer()
        }
    }
    
    private func projectCarousel(containerWidth: CGFloat) -> some View {
        let centeredInset = max(0, (containerWidth - carouselCardWidth) / 2)
        let leftEdgeNudge = Spacing.lg + 10
        let cardSpacing = Spacing.lg + 8
        
        return carouselScrollView(cardSpacing: cardSpacing)
            .scrollTargetBehavior(.viewAligned(limitBehavior: .alwaysByOne))
            .scrollPosition(id: $activeCarouselID)
            .safeAreaPadding(.horizontal, centeredInset)
            .padding(.leading, -leftEdgeNudge)
            .padding(.trailing, -Spacing.lg)
            .frame(height: 360)
            .onAppear {
                normalizeActiveCarouselID()
            }
            .onChange(of: carouselItems.count) { _, _ in
                normalizeActiveCarouselID()
            }
            .simultaneousGesture(
                DragGesture(minimumDistance: 8)
                    .onEnded { value in
                        handleCarouselDragEnd(value)
                    }
            )
            .animation(.spring(response: 0.46, dampingFraction: 0.84, blendDuration: 0.18), value: activeCarouselID)
            .sensoryFeedback(.impact(weight: .medium), trigger: activeCarouselID)
    }
    
    @ViewBuilder
    private func carouselScrollView(cardSpacing: CGFloat) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            LazyHStack(alignment: .top, spacing: cardSpacing) {
                ForEach(carouselItems, id: \.id) { item in
                    carouselItemView(item: item)
                }
            }
            .scrollTargetLayout()
        }
    }
    
    @ViewBuilder
    private func carouselItemView(item: CarouselItem) -> some View {
        VStack(spacing: 0) {
            carouselCard(for: item)
        }
        .id(item.id)
        .visualEffect { content, proxy in
            let minX = proxy.frame(in: .scrollView(axis: .horizontal)).minX
            let rightProgress = max(0, minX) / carouselTransitionDistance
            let leftProgress = max(0, -minX) / carouselTransitionDistance
            
            let easedRight = premiumEase(min(1, rightProgress))
            let easedLeft = premiumEase(min(1, leftProgress))
            
            let rightScale = 1 - (easedRight * 0.10)
            
            // Hide previous card on the left while keeping right preview visible.
            let leftHide = min(1, easedLeft * 1.05)
            let leftShift = leftHide * 60
            let rightParallax = easedRight * 8
            let rightTilt = easedRight * 2.5
            
            return content
                .scaleEffect(rightScale, anchor: .top)
                .rotation3DEffect(.degrees(rightTilt), axis: (x: 0, y: 1, z: 0), anchor: .top)
                .offset(x: -leftShift - rightParallax)
                .blur(radius: leftHide * 0.6)
        }
    }
    
    nonisolated private func premiumEase(_ value: CGFloat) -> CGFloat {
        // Smooth ease-out cubic for premium-feeling interpolation.
        let t = min(max(value, 0), 1)
        return 1 - pow(1 - t, 3)
    }
    
    private var carouselItems: [CarouselItem] {
        let projects = viewModel.projects.map { CarouselItem(id: $0.id.uuidString, project: $0) }
        return projects + [CarouselItem(id: "add-project-card", project: nil)]
    }
    
    private func normalizeActiveCarouselID() {
        let allIDs = Set(carouselItems.map(\.id))
        let firstProjectID = viewModel.projects.first.map { $0.id.uuidString }
        
        if let firstProjectID {
            // If projects exist, always prefer starting/landing on the first real project.
            if activeCarouselID == nil || activeCarouselID == "add-project-card" || !allIDs.contains(activeCarouselID ?? "") {
                activeCarouselID = firstProjectID
            }
            return
        }
        
        // No projects: default to add card.
        if activeCarouselID == nil || !allIDs.contains(activeCarouselID ?? "") {
            activeCarouselID = carouselItems.first?.id
        }
    }
    
    private func handleCarouselDragEnd(_ value: DragGesture.Value) {
        guard !carouselItems.isEmpty else { return }
        guard let currentID = activeCarouselID,
              let currentIndex = carouselItems.firstIndex(where: { $0.id == currentID }) else {
            normalizeActiveCarouselID()
            return
        }
        
        let translation = value.translation.width
        let threshold: CGFloat = 12
        
        if translation < -threshold {
            let nextIndex = min(currentIndex + 1, carouselItems.count - 1)
            if nextIndex != currentIndex {
                withAnimation(.interactiveSpring(response: 0.48, dampingFraction: 0.86, blendDuration: 0.16)) {
                    activeCarouselID = carouselItems[nextIndex].id
                }
            }
        } else if translation > threshold {
            let previousIndex = max(currentIndex - 1, 0)
            if previousIndex != currentIndex {
                withAnimation(.interactiveSpring(response: 0.48, dampingFraction: 0.86, blendDuration: 0.16)) {
                    activeCarouselID = carouselItems[previousIndex].id
                }
            }
        }
    }
    
    @ViewBuilder
    private func carouselCard(for item: CarouselItem) -> some View {
        if let project = item.project {
            ZStack(alignment: .topTrailing) {
                ProjectCardView(
                    project: project
                )
                .matchedGeometryEffect(id: "project-hero-\(item.id)", in: projectHeaderNamespace)
                .contentShape(Rectangle())
                .onTapGesture {
                    if deleteArmedProjectID == project.id {
                        withAnimation(.spring(response: 0.28, dampingFraction: 0.88)) {
                            deleteArmedProjectID = nil
                        }
                    } else {
                        navigationManager.showProjectDetail(project.id)
                    }
                }
                .onLongPressGesture(minimumDuration: 0.35) {
                    withAnimation(.spring(response: 0.32, dampingFraction: 0.8)) {
                        deleteArmedProjectID = project.id
                    }
                }
                
                if deleteArmedProjectID == project.id {
                    Button {
                        Task {
                            let accessToken = await resolveAccessToken()
                            let deleted = await viewModel.deleteProject(projectID: project.id, accessToken: accessToken)
                            if deleted {
                                withAnimation(.spring(response: 0.28, dampingFraction: 0.85)) {
                                    deleteArmedProjectID = nil
                                }
                                withAnimation(.spring(response: 0.32, dampingFraction: 0.82)) {
                                    topNotification = .success("Project deleted")
                                }
                            }
                        }
                    } label: {
                        Image(systemName: "trash.fill")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(.white)
                            .frame(width: 34, height: 34)
                            .background(Color.red)
                            .clipShape(Circle())
                    }
                    .buttonStyle(.plain)
                    .padding(Spacing.sm)
                    .transition(.scale.combined(with: .opacity))
                }
            }
        } else {
            AddProjectCardView {
                showCreateProjectSheet = true
            }
            .matchedGeometryEffect(id: "project-hero-\(item.id)", in: projectHeaderNamespace)
        }
    }
    
    @ViewBuilder
    private var activeProjectTrackListSection: some View {
        if let project = activeProjectForTrackList {
            TrackListView(
                projectID: project.id,
                apiClient: appContainer.apiClient,
                maxNonScrollableHeight: 340,
                onOverflowChange: { isOverflowing in
                    withAnimation(.spring(response: 0.62, dampingFraction: 0.74, blendDuration: 0.2)) {
                        isTrackListOverflowing = isOverflowing
                    }
                },
                playbackController: playbackController,
                onActiveTrackChange: { track in
                    onActiveTrackChange(track)
                }
            )
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private var compactProjectHeader: some View {
        if let item = activeCarouselItem {
            HStack(spacing: Spacing.md) {
                compactHeaderLeadingView(for: item)
                Text(compactHeaderTitle(for: item))
                    .font(PurpTapeTypography.headlineSmall)
                    .foregroundStyle(PurpTapeColors.text)
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .center)
                Color.clear
                    .frame(width: 58, height: 58)
            }
            .padding(.horizontal, Spacing.md)
            .padding(.vertical, Spacing.sm)
            .frame(maxWidth: .infinity)
            .background(PurpTapeColors.surface)
            .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
            .shadow(color: PurpTapeColors.shadowLight, radius: 12, x: 0, y: 8)
            .onTapGesture {
                if let project = item.project {
                    navigationManager.showProjectDetail(project.id)
                } else {
                    showCreateProjectSheet = true
                }
            }
        }
    }

    @ViewBuilder
    private func compactHeaderLeadingView(for item: CarouselItem) -> some View {
        if let project = item.project {
            AppRemoteImage(url: URL(string: project.coverImageURL ?? ""), debugLabel: "Project \(project.id)")
                .frame(width: 58, height: 58)
                .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                .matchedGeometryEffect(id: "project-hero-\(item.id)", in: projectHeaderNamespace)
        } else {
            ZStack {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(LinearGradient.purpTapePrimary)
                Image(systemName: "plus")
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(.white)
            }
            .frame(width: 58, height: 58)
            .matchedGeometryEffect(id: "project-hero-\(item.id)", in: projectHeaderNamespace)
        }
    }

    private func compactHeaderTitle(for item: CarouselItem) -> String {
        if let project = item.project {
            let trimmedName = project.name.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmedName.isEmpty ? "Project" : trimmedName
        }
        return "Project"
    }

    private var activeCarouselItem: CarouselItem? {
        guard let activeCarouselID,
              let matchedItem = carouselItems.first(where: { $0.id == activeCarouselID }) else {
            return carouselItems.first
        }
        return matchedItem
    }
    
    private var activeProjectForTrackList: Project? {
        guard let activeCarouselID,
              let activeUUID = UUID(uuidString: activeCarouselID) else {
            return viewModel.projects.first
        }
        return viewModel.projects.first(where: { $0.id == activeUUID })
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

private struct ProjectsHeaderScrollOffsetKey: PreferenceKey {
    static let defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

private struct CarouselItem {
    let id: String
    let project: Project?
}
