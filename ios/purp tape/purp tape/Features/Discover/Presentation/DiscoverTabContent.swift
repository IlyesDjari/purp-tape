import SwiftUI

struct DiscoverTabContent: View {
    @Environment(\.navigationManager) var navigationManager

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.lg) {
            Text("Discover")
                .titleLarge()
                .foregroundColor(PurpTapeColors.text)

            Spacer()
        }
        .paddingHorizontalLG()
        .purpTapeBackground()
    }
}
