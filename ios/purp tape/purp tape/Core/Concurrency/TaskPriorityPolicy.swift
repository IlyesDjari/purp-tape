import Foundation

public enum TaskPriorityPolicy: Sendable {
    public static let streaming: TaskPriority = .userInitiated
    public static let upload: TaskPriority = .utility
    public static let heavyAudioProcessing: TaskPriority = .background
    public static let stemSeparation: TaskPriority = .background
}
