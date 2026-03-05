import Foundation
import os

/// A custom logger that formats all debug messages with [DEBUG] emoji prefix
public struct DebugLogger: Sendable {
    private let subsystem: String
    private let category: String
    private let osLogger: os.Logger
    
    public init(subsystem: String = "ilyes.purp-tape", category: String) {
        self.subsystem = subsystem
        self.category = category
        self.osLogger = os.Logger(subsystem: subsystem, category: category)
    }
    
    /// Log a debug message with emoji
    /// Format: [DEBUG] 🐛 message
    public func debug(_ message: String, emoji: String = "🐛") {
        let formattedMessage = "[\(emoji)][DEBUG] \(message)"
        osLogger.debug("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log an info message with emoji
    /// Format: [DEBUG] ℹ️ message
    public func info(_ message: String, emoji: String = "ℹ️") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.info("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log an error message with emoji
    /// Format: [DEBUG] ❌ message
    public func error(_ message: String, emoji: String = "❌") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.error("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log a warning message with emoji
    /// Format: [DEBUG] ⚠️ message
    public func warning(_ message: String, emoji: String = "⚠️") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.warning("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log a success message with emoji
    /// Format: [DEBUG] ✅ message
    public func success(_ message: String, emoji: String = "✅") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.info("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log a network message with emoji
    /// Format: [DEBUG] 🌐 message
    public func network(_ message: String, emoji: String = "🌐") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.debug("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log an auth message with emoji
    /// Format: [DEBUG] 🔐 message
    public func auth(_ message: String, emoji: String = "🔐") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.debug("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
    
    /// Log a database message with emoji
    /// Format: [DEBUG] 💾 message
    public func database(_ message: String, emoji: String = "💾") {
        let formattedMessage = "[DEBUG] \(emoji) \(message)"
        osLogger.debug("\(formattedMessage, privacy: .public)")
        #if DEBUG
        print(formattedMessage)
        #endif
    }
}


