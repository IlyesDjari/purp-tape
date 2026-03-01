import SwiftUI

// MARK: - Typography Scales

public struct PurpTapeTypography {
    // MARK: - Display Sizes (Hero/Very Large)
    
    public static let displayLarge = Font.system(size: 32, weight: .bold, design: .default)
    public static let displayMedium = Font.system(size: 28, weight: .bold, design: .default)
    public static let displaySmall = Font.system(size: 24, weight: .bold, design: .default)
    
    // MARK: - Headline Sizes (Large Titles)
    
    public static let headlineLarge = Font.system(size: 22, weight: .bold, design: .default)
    public static let headlineMedium = Font.system(size: 20, weight: .semibold, design: .default)
    public static let headlineSmall = Font.system(size: 18, weight: .semibold, design: .default)
    
    // MARK: - Title Sizes
    
    public static let titleLarge = Font.system(size: 18, weight: .bold, design: .default)
    public static let titleMedium = Font.system(size: 16, weight: .semibold, design: .default)
    public static let titleSmall = Font.system(size: 14, weight: .semibold, design: .default)
    
    // MARK: - Body Sizes
    
    public static let bodyLarge = Font.system(size: 16, weight: .regular, design: .default)
    public static let bodyMedium = Font.system(size: 14, weight: .regular, design: .default)
    public static let bodySmall = Font.system(size: 12, weight: .regular, design: .default)
    
    // MARK: - Label Sizes
    
    public static let labelLarge = Font.system(size: 14, weight: .medium, design: .default)
    public static let labelMedium = Font.system(size: 12, weight: .medium, design: .default)
    public static let labelSmall = Font.system(size: 11, weight: .medium, design: .default)
    
    // MARK: - Caption Sizes
    
    public static let captionLarge = Font.system(size: 12, weight: .regular, design: .default)
    public static let captionSmall = Font.system(size: 10, weight: .regular, design: .default)
    
    // MARK: - Monospace Sizes (for code/data)
    
    public static let monospaceLarge = Font.system(size: 14, weight: .regular, design: .monospaced)
    public static let monospaceMedium = Font.system(size: 12, weight: .regular, design: .monospaced)
    public static let monospaceSmall = Font.system(size: 10, weight: .regular, design: .monospaced)
}

// MARK: - Font Weights

public extension Font.Weight {
    static let thin = Font.Weight.thin
    static let light = Font.Weight.light
    static let regular = Font.Weight.regular
    static let medium = Font.Weight.medium
    static let semibold = Font.Weight.semibold
    static let bold = Font.Weight.bold
    static let heavy = Font.Weight.heavy
    static let black = Font.Weight.black
}

// MARK: - Line Heights

public struct LineHeights {
    public static let tight: CGFloat = 1.2
    public static let normal: CGFloat = 1.5
    public static let relaxed: CGFloat = 1.625
    public static let loose: CGFloat = 2.0
}

// MARK: - Letter Spacing

public struct LetterSpacing {
    public static let tight: CGFloat = -0.5
    public static let normal: CGFloat = 0
    public static let wide: CGFloat = 0.5
    public static let wider: CGFloat = 1.0
}

// MARK: - View Modifiers

public extension Text {
    func displayLarge() -> some View {
        self.font(PurpTapeTypography.displayLarge)
    }
    
    func displayMedium() -> some View {
        self.font(PurpTapeTypography.displayMedium)
    }
    
    func displaySmall() -> some View {
        self.font(PurpTapeTypography.displaySmall)
    }
    
    func headlineLarge() -> some View {
        self.font(PurpTapeTypography.headlineLarge)
    }
    
    func headlineMedium() -> some View {
        self.font(PurpTapeTypography.headlineMedium)
    }
    
    func headlineSmall() -> some View {
        self.font(PurpTapeTypography.headlineSmall)
    }
    
    func titleLarge() -> some View {
        self.font(PurpTapeTypography.titleLarge)
    }
    
    func titleMedium() -> some View {
        self.font(PurpTapeTypography.titleMedium)
    }
    
    func titleSmall() -> some View {
        self.font(PurpTapeTypography.titleSmall)
    }
    
    func bodyLarge() -> some View {
        self.font(PurpTapeTypography.bodyLarge)
    }
    
    func bodyMedium() -> some View {
        self.font(PurpTapeTypography.bodyMedium)
    }
    
    func bodySmall() -> some View {
        self.font(PurpTapeTypography.bodySmall)
    }
    
    func labelLarge() -> some View {
        self.font(PurpTapeTypography.labelLarge)
    }
    
    func labelMedium() -> some View {
        self.font(PurpTapeTypography.labelMedium)
    }
    
    func labelSmall() -> some View {
        self.font(PurpTapeTypography.labelSmall)
    }
    
    func captionLarge() -> some View {
        self.font(PurpTapeTypography.captionLarge)
    }
    
    func captionSmall() -> some View {
        self.font(PurpTapeTypography.captionSmall)
    }
    
    func monospaceLarge() -> some View {
        self.font(PurpTapeTypography.monospaceLarge)
    }
    
    func monospaceMedium() -> some View {
        self.font(PurpTapeTypography.monospaceMedium)
    }
    
    func monospaceSmall() -> some View {
        self.font(PurpTapeTypography.monospaceSmall)
    }
}
