import SwiftUI

// MARK: - Color Palette

public struct PurpTapeColors {
    // MARK: - Primary Colors
    
    public static let primary = Color(red: 0.49, green: 0.27, blue: 0.91) // Purple
    public static let primaryLight = Color(red: 0.69, green: 0.54, blue: 0.96)
    public static let primaryDark = Color(red: 0.38, green: 0.17, blue: 0.78)
    
    // MARK: - Secondary Colors
    
    public static let secondary = Color(red: 0.94, green: 0.92, blue: 0.99)
    public static let secondaryLight = Color(red: 0.98, green: 0.97, blue: 1.0)
    public static let secondaryDark = Color(red: 0.87, green: 0.83, blue: 0.97)
    
    // MARK: - Accent Colors
    
    public static let accent = Color(red: 0.73, green: 0.51, blue: 0.97)
    public static let accentLight = Color(red: 0.84, green: 0.7, blue: 0.98)
    public static let accentDark = Color(red: 0.57, green: 0.35, blue: 0.86)
    
    // MARK: - Semantic Colors
    
    public static let success = Color(red: 0.2, green: 0.8, blue: 0.4)
    public static let warning = Color(red: 1.0, green: 0.8, blue: 0.2)
    public static let error = Color(red: 1.0, green: 0.2, blue: 0.2)
    public static let info = Color(red: 0.2, green: 0.6, blue: 1.0)
    
    // MARK: - Neutral Colors
    
    public static let background = Color.white
    public static let surface = Color.white
    public static let text = Color(red: 0.12, green: 0.12, blue: 0.16)
    public static let textSecondary = Color(red: 0.46, green: 0.46, blue: 0.52)
    public static let border = Color(red: 0.9, green: 0.9, blue: 0.93)
    public static let divider = Color(red: 0.92, green: 0.92, blue: 0.95)
    
    // MARK: - Shadow Colors
    
    public static let shadow = Color.black.opacity(0.08)
    public static let shadowLight = Color.black.opacity(0.04)
    public static let shadowDark = Color.black.opacity(0.14)
}

// MARK: - Theme

public enum Theme {
    case light
    case dark
    case auto
}

// MARK: - Color Extensions

public extension Color {
    static let purpTapePrimary = PurpTapeColors.primary
    static let purpTapeSecondary = PurpTapeColors.secondary
    static let purpTapeAccent = PurpTapeColors.accent
}

// MARK: - Usage Example

extension View {
    /// Apply primary color to view
    public func primaryColored() -> some View {
        foregroundColor(PurpTapeColors.primary)
    }
    
    /// Apply secondary color to view
    public func secondaryColored() -> some View {
        foregroundColor(PurpTapeColors.textSecondary)
    }
    
    /// Apply background color
    public func purpTapeBackground() -> some View {
        background(PurpTapeColors.background)
    }
    
    /// Apply surface color
    public func purpTapeSurface() -> some View {
        background(PurpTapeColors.surface)
    }
}

// MARK: - Gradient

public extension LinearGradient {
    static let purpTapePrimary = LinearGradient(
        gradient: Gradient(colors: [
            PurpTapeColors.primaryDark,
            PurpTapeColors.primary,
            PurpTapeColors.primaryLight
        ]),
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
    
    static let purpTapeSecondary = LinearGradient(
        gradient: Gradient(colors: [
            PurpTapeColors.secondaryDark,
            PurpTapeColors.secondary,
            PurpTapeColors.secondaryLight
        ]),
        startPoint: .leading,
        endPoint: .trailing
    )
}
