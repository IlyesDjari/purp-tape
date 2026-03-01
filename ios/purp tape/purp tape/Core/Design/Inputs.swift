import SwiftUI

private struct PurpTapeInputContainerModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .paddingMD()
            .background(PurpTapeColors.surface)
            .overlay(
                RoundedRectangle(cornerRadius: CornerRadius.md)
                    .stroke(PurpTapeColors.border, lineWidth: 1)
            )
            .cornerRadiusMD()
            .shadow(color: PurpTapeColors.shadowLight, radius: 6, x: 0, y: 2)
    }
}

public extension View {
    func purpTapeInputContainer() -> some View {
        modifier(PurpTapeInputContainerModifier())
    }
}
