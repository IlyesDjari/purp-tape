import SwiftUI
#if canImport(UIKit)
import UIKit
#endif

struct ProjectCardView: View {
    let project: Project
    let artworkData: Data?

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.md) {
#if canImport(UIKit)
            if let artworkData, let image = UIImage(data: artworkData) {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
                    .frame(height: 120)
                    .frame(maxWidth: .infinity)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
            }
#endif

            HStack {
                Text(project.name)
                    .font(PurpTapeTypography.headlineSmall)
                    .foregroundColor(PurpTapeColors.text)
                    .lineLimit(2)

                Spacer()

                Text(project.isPublic ? "Public" : "Private")
                    .font(PurpTapeTypography.labelSmall)
                    .foregroundColor(project.isPublic ? PurpTapeColors.success : PurpTapeColors.textSecondary)
                    .padding(.horizontal, Spacing.sm)
                    .padding(.vertical, Spacing.xs)
                    .background(PurpTapeColors.background)
                    .clipShape(Capsule())
            }

            Text(project.description ?? "No description")
                .font(PurpTapeTypography.bodyMedium)
                .foregroundColor(PurpTapeColors.textSecondary)
                .lineLimit(4)

            Spacer()
        }
        .padding(Spacing.lg)
        .frame(maxWidth: .infinity, minHeight: 320, maxHeight: 320, alignment: .topLeading)
        .background(PurpTapeColors.surface)
        .clipShape(RoundedRectangle(cornerRadius: 24))
        .shadow(color: PurpTapeColors.shadowLight, radius: 12, x: 0, y: 6)
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
            .frame(maxWidth: .infinity, minHeight: 320, maxHeight: 320)
            .background(PurpTapeColors.surface)
            .overlay(
                RoundedRectangle(cornerRadius: 24)
                    .stroke(PurpTapeColors.primary.opacity(0.25), lineWidth: 1.5)
            )
            .clipShape(RoundedRectangle(cornerRadius: 24))
            .shadow(color: PurpTapeColors.shadowLight, radius: 12, x: 0, y: 6)
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
#if os(iOS)
        .tabViewStyle(.page(indexDisplayMode: .automatic))
        .indexViewStyle(.page(backgroundDisplayMode: .always))
#endif
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
                .fill(PurpTapeColors.background)
                .frame(height: 14)

            RoundedRectangle(cornerRadius: 8)
                .fill(PurpTapeColors.background)
                .frame(width: 220, height: 14)

            Spacer()
        }
        .padding(Spacing.lg)
        .frame(maxWidth: .infinity, minHeight: 320, maxHeight: 320, alignment: .topLeading)
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
        .shadow(color: PurpTapeColors.shadowLight, radius: 12, x: 0, y: 6)
    }
}
