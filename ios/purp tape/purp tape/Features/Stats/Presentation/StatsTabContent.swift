import SwiftUI

struct StatsTabContent: View {
    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.lg) {
            Text("Stats")
                .titleLarge()
                .foregroundColor(PurpTapeColors.text)

            Spacer()
        }
        .paddingHorizontalLG()
        .purpTapeBackground()
    }
}
