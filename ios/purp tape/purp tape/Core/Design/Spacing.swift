import SwiftUI

// MARK: - Spacing Scale

public struct Spacing {
    // MARK: - Extra Small
    
    public static let xs: CGFloat = 4
    
    // MARK: - Small
    
    public static let sm: CGFloat = 8
    
    // MARK: - Medium
    
    public static let md: CGFloat = 12
    public static let medium = md
    
    // MARK: - Large
    
    public static let lg: CGFloat = 16
    public static let large = lg
    
    // MARK: - Extra Large
    
    public static let xl: CGFloat = 24
    
    // MARK: - 2X Large
    
    public static let xxl: CGFloat = 32
    
    // MARK: - 3X Large
    
    public static let xxxl: CGFloat = 40
    
    // MARK: - Common Combinations
    
    public static let standard = lg // Default padding
    public static let compact = md
    public static let comfortable = xl
    public static let spacious = xxl
}

// MARK: - Corner Radius

public struct CornerRadius {
    public static let xs: CGFloat = 2
    public static let sm: CGFloat = 4
    public static let md: CGFloat = 8
    public static let lg: CGFloat = 12
    public static let xl: CGFloat = 16
    public static let xxl: CGFloat = 24
    public static let full: CGFloat = .infinity
    
    public static let small = sm
    public static let medium = md
    public static let large = lg
}

// MARK: - View Modifiers for Spacing

public extension View {
    // MARK: - Padding
    
    func paddingXS() -> some View {
        padding(Spacing.xs)
    }
    
    func paddingSM() -> some View {
        padding(Spacing.sm)
    }
    
    func paddingMD() -> some View {
        padding(Spacing.md)
    }
    
    func paddingLG() -> some View {
        padding(Spacing.lg)
    }
    
    func paddingXL() -> some View {
        padding(Spacing.xl)
    }
    
    func paddingXXL() -> some View {
        padding(Spacing.xxl)
    }
    
    // MARK: - Horizontal Padding
    
    func paddingHorizontalXS() -> some View {
        padding(.horizontal, Spacing.xs)
    }
    
    func paddingHorizontalSM() -> some View {
        padding(.horizontal, Spacing.sm)
    }
    
    func paddingHorizontalMD() -> some View {
        padding(.horizontal, Spacing.md)
    }
    
    func paddingHorizontalLG() -> some View {
        padding(.horizontal, Spacing.lg)
    }
    
    func paddingHorizontalXL() -> some View {
        padding(.horizontal, Spacing.xl)
    }
    
    // MARK: - Vertical Padding
    
    func paddingVerticalXS() -> some View {
        padding(.vertical, Spacing.xs)
    }
    
    func paddingVerticalSM() -> some View {
        padding(.vertical, Spacing.sm)
    }
    
    func paddingVerticalMD() -> some View {
        padding(.vertical, Spacing.md)
    }
    
    func paddingVerticalLG() -> some View {
        padding(.vertical, Spacing.lg)
    }
    
    func paddingVerticalXL() -> some View {
        padding(.vertical, Spacing.xl)
    }
    
    // MARK: - Individual Edge Padding
    
    func paddingTopLG() -> some View {
        padding(.top, Spacing.lg)
    }
    
    func paddingBottomLG() -> some View {
        padding(.bottom, Spacing.lg)
    }
    
    func paddingLeadingLG() -> some View {
        padding(.leading, Spacing.lg)
    }
    
    func paddingTrailingLG() -> some View {
        padding(.trailing, Spacing.lg)
    }
    
    // MARK: - Corner Radius
    
    func cornerRadiusSM() -> some View {
        clipShape(RoundedRectangle(cornerRadius: CornerRadius.sm))
    }
    
    func cornerRadiusMD() -> some View {
        clipShape(RoundedRectangle(cornerRadius: CornerRadius.md))
    }
    
    func cornerRadiusLG() -> some View {
        clipShape(RoundedRectangle(cornerRadius: CornerRadius.lg))
    }
    
    func cornerRadiusXL() -> some View {
        clipShape(RoundedRectangle(cornerRadius: CornerRadius.xl))
    }
}

// MARK: - Shadow Styles

public struct ShadowStyle {
    public static let light = Shadow(
        color: Color.black.opacity(0.05),
        radius: 2,
        horizontal: 0,
        vertical: 1
    )
    
    public static let medium = Shadow(
        color: Color.black.opacity(0.1),
        radius: 4,
        horizontal: 0,
        vertical: 2
    )
    
    public static let large = Shadow(
        color: Color.black.opacity(0.15),
        radius: 8,
        horizontal: 0,
        vertical: 4
    )
}

public struct Shadow: ViewModifier {
    let color: Color
    let radius: CGFloat
    let horizontalOffset: CGFloat
    let verticalOffset: CGFloat

    public init(color: Color, radius: CGFloat, horizontal: CGFloat, vertical: CGFloat) {
        self.color = color
        self.radius = radius
        self.horizontalOffset = horizontal
        self.verticalOffset = vertical
    }
    
    public func body(content: Content) -> some View {
        content
            .shadow(color: color, radius: radius, x: horizontalOffset, y: verticalOffset)
    }
}

public extension View {
    func shadowLight() -> some View {
        self.modifier(ShadowStyle.light)
    }
    
    func shadowMedium() -> some View {
        self.modifier(ShadowStyle.medium)
    }
    
    func shadowLarge() -> some View {
        self.modifier(ShadowStyle.large)
    }
}

// MARK: - Dividers & Separators

public extension Divider {
    static func purpTapeDivider() -> some View {
        Divider()
            .background(PurpTapeColors.divider)
    }
}

public struct VerticalDivider: View {
    let color: Color
    
    public init(color: Color = PurpTapeColors.divider) {
        self.color = color
    }
    
    public var body: some View {
        Rectangle()
            .fill(color)
            .frame(width: 1)
    }
}

// MARK: - Common Layouts

public struct CardLayout<Content: View>: View {
    let content: Content
    
    public init(@ViewBuilder content: () -> Content) {
        self.content = content()
    }
    
    public var body: some View {
        content
            .paddingLG()
            .background(PurpTapeColors.surface)
            .cornerRadiusLG()
            .shadowMedium()
    }
}

public extension View {
    func cardStyle() -> some View {
        self
            .paddingLG()
            .background(PurpTapeColors.surface)
            .cornerRadiusLG()
            .shadowMedium()
    }
}
