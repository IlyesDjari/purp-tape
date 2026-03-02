import SwiftUI
import ImageIO

private enum ProjectCardStyle {
    static let width: CGFloat = 230
    static let height: CGFloat = 320
    static let shadowRadius: CGFloat = 18
    static let shadowYOffset: CGFloat = 10
}

struct ProjectCardView: View {
    let project: Project

    var body: some View {
        AppRemoteImage(url: URL(string: project.coverImageURL ?? ""), debugLabel: "Project \(project.id)")
            .frame(width: ProjectCardStyle.width, height: ProjectCardStyle.height)
            .clipShape(RoundedRectangle(cornerRadius: 24))
            .shadow(color: PurpTapeColors.shadowLight, radius: ProjectCardStyle.shadowRadius, x: 0, y: ProjectCardStyle.shadowYOffset)
            .overlay(RoundedRectangle(cornerRadius: 24).stroke(Color.pink.opacity(0.12), lineWidth: 1))
    }
}

struct AddProjectCardView: View {
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: Spacing.md) {
                Image(systemName: "plus")
                    .font(.system(size: 28, weight: .bold))
                    .foregroundColor(.white)
                    .frame(width: 56, height: 56)
                    .background(LinearGradient.purpTapePrimary)
                    .clipShape(Circle())
                
                Text("Add Project")
                    .font(PurpTapeTypography.headlineSmall)
                    .foregroundColor(PurpTapeColors.text)
                
                Text("Create a new project")
                    .font(PurpTapeTypography.bodySmall)
                    .foregroundColor(PurpTapeColors.textSecondary)
            }
            .frame(width: ProjectCardStyle.width, height: ProjectCardStyle.height)
            .background(PurpTapeColors.surface)
            .overlay(
                RoundedRectangle(cornerRadius: 24)
                    .stroke(PurpTapeColors.primary.opacity(0.25), lineWidth: 1.5)
            )
            .clipShape(RoundedRectangle(cornerRadius: 24))
            .shadow(color: PurpTapeColors.shadowLight, radius: ProjectCardStyle.shadowRadius, x: 0, y: ProjectCardStyle.shadowYOffset)
        }
        .buttonStyle(PurpTapeInteractiveButtonStyle())
    }
}

struct ProjectsLoadingCarouselView: View {
    var body: some View {
        TabView {
            ForEach(0..<3, id: \.self) { _ in
                ShimmerProjectCardView()
                    .padding(.horizontal, 2)
            }
        }
        .frame(height: 360)
    .tabViewStyle(.page(indexDisplayMode: .never))
    }

}

private struct ShimmerProjectCardView: View {
    @State private var shimmerOffset: CGFloat = -220
    
    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.md) {
            RoundedRectangle(cornerRadius: 16)
                .fill(PurpTapeColors.background)
                .frame(height: 120)
            
            RoundedRectangle(cornerRadius: 8)
                .fill(PurpTapeColors.background)
                .frame(width: 170, height: 18)
            
            RoundedRectangle(cornerRadius: 8)
                    .clipShape(RoundedRectangle(cornerRadius: PurpTapeShapes.cardCornerRadius))
                    .shadow(color: PurpTapeColors.shadowLight, radius: ProjectCardStyle.shadowRadius, x: 0, y: ProjectCardStyle.shadowYOffset)
                    .overlay(RoundedRectangle(cornerRadius: PurpTapeShapes.cardCornerRadius).stroke(PurpTapeColors.primary.opacity(0.12), lineWidth: 1))
            RoundedRectangle(cornerRadius: 8)
                .fill(PurpTapeColors.background)
                .frame(width: 220, height: 14)
            
            Spacer()
        }
        .padding(Spacing.lg)
        .frame(width: ProjectCardStyle.width, height: ProjectCardStyle.height, alignment: .topLeading)
        .background(PurpTapeColors.surface)
        .clipShape(RoundedRectangle(cornerRadius: 24))
        .overlay {
            GeometryReader { proxy in
                LinearGradient(
                    gradient: Gradient(colors: [
                        Color.white.opacity(0.0),
                        Color.white.opacity(0.45),
                        Color.white.opacity(0.0)
                    ]),
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .frame(width: 120)
                .offset(x: shimmerOffset)
                .onAppear {
                    withAnimation(.linear(duration: 1.25).repeatForever(autoreverses: false)) {
                        shimmerOffset = proxy.size.width + 140
                    }
                }
            }
            .clipShape(RoundedRectangle(cornerRadius: 24))
        }
        .shadow(color: PurpTapeColors.shadowLight, radius: ProjectCardStyle.shadowRadius, x: 0, y: ProjectCardStyle.shadowYOffset)
    }
}
