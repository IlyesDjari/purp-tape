import SwiftUI
import PhotosUI
#if canImport(UIKit)
import UIKit
#endif

@MainActor
struct CreateProjectSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var description = ""
    @State private var isPublic = false
    @State private var selectedArtworkItem: PhotosPickerItem?
    @State private var selectedArtworkData: Data?

    let isLoading: Bool
    let onCreate: (_ name: String, _ description: String, _ isPublic: Bool, _ artworkData: Data?) async -> Void
    let onHeightChange: @Sendable (CGFloat) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.lg) {
            HStack {
                VStack(alignment: .leading, spacing: Spacing.xs) {
                    Text("Create Project")
                        .font(PurpTapeTypography.headlineMedium)
                        .foregroundColor(PurpTapeColors.text)

                    Text("Start something new")
                        .font(PurpTapeTypography.bodySmall)
                        .foregroundColor(PurpTapeColors.textSecondary)
                }

                Spacer()

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundColor(PurpTapeColors.textSecondary)
                        .frame(width: 28, height: 28)
                        .background(PurpTapeColors.background)
                        .clipShape(Circle())
                }
                .buttonStyle(PurpTapeInteractiveButtonStyle())
            }

            VStack(alignment: .leading, spacing: Spacing.sm) {
                Text("Name")
                    .font(PurpTapeTypography.labelLarge)
                    .foregroundColor(PurpTapeColors.text)

                TextField("Project name", text: $name)
                    .purpTapeInputContainer()
            }

            VStack(alignment: .leading, spacing: Spacing.sm) {
                Text("Description")
                    .font(PurpTapeTypography.labelLarge)
                    .foregroundColor(PurpTapeColors.text)

                TextField("Optional description", text: $description, axis: .vertical)
                    .lineLimit(3, reservesSpace: true)
                    .purpTapeInputContainer()
            }

            VStack(alignment: .leading, spacing: Spacing.sm) {
                Text("Artwork")
                    .font(PurpTapeTypography.labelLarge)
                    .foregroundColor(PurpTapeColors.text)

                let artworkData = selectedArtworkData

                PhotosPicker(selection: $selectedArtworkItem, matching: .images) {
                    ZStack {
                        RoundedRectangle(cornerRadius: 16)
                            .fill(PurpTapeColors.background)
                            .frame(height: 132)

                        if let artworkData,
                           let image = UIImage(data: artworkData) {
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFill()
                                .frame(height: 132)
                                .clipShape(RoundedRectangle(cornerRadius: 16))
                        } else {
                            VStack(spacing: Spacing.xs) {
                                Image(systemName: "photo")
                                    .font(.system(size: 22, weight: .semibold))
                                    .foregroundColor(PurpTapeColors.primary)
                                Text("Pick artwork")
                                    .font(PurpTapeTypography.labelMedium)
                                    .foregroundColor(PurpTapeColors.textSecondary)
                            }
                        }
                    }
                }
                .buttonStyle(PurpTapeInteractiveButtonStyle())
            }

            HStack {
                Text("Visibility")
                    .font(PurpTapeTypography.labelLarge)
                    .foregroundColor(PurpTapeColors.text)

                Spacer()

                Picker("Visibility", selection: $isPublic) {
                    Text("Private").tag(false)
                    Text("Public").tag(true)
                }
                .pickerStyle(.segmented)
                .frame(width: 180)
            }

            HStack(spacing: Spacing.md) {
                PurpTapeSecondaryButton("Cancel") {
                    dismiss()
                }

                PurpTapePrimaryButton(
                    "Create",
                    isLoading: isLoading,
                    isDisabled: name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                ) {
                    let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
                    let trimmedDescription = description.trimmingCharacters(in: .whitespacesAndNewlines)
                    let selectedVisibility = isPublic
                    let artworkData = selectedArtworkData

                    Task {
                        await onCreate(
                            trimmedName,
                            trimmedDescription,
                            selectedVisibility,
                            artworkData
                        )
                    }
                }
            }
        }
        .padding(Spacing.lg)
        .background(
            LinearGradient(
                gradient: Gradient(colors: [Color.white, PurpTapeColors.primary.opacity(0.05)]),
                startPoint: .top,
                endPoint: .bottom
            )
        )
        .clipShape(RoundedRectangle(cornerRadius: 24))
        .background(
            GeometryReader { geometry in
                Color.clear
                    .preference(key: SheetHeightPreferenceKey.self, value: geometry.size.height)
            }
        )
        .onPreferenceChange(SheetHeightPreferenceKey.self) { newHeight in
            Task { @MainActor in
                onHeightChange(newHeight)
            }
        }
        .onChange(of: selectedArtworkItem) { _, newItem in
            guard let newItem else { return }
            Task {
                guard let data = try? await newItem.loadTransferable(type: Data.self) else { return }
                await MainActor.run { selectedArtworkData = data }
            }
        }
    }
}

private struct SheetHeightPreferenceKey: PreferenceKey {
    static let defaultValue: CGFloat = 380

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}
