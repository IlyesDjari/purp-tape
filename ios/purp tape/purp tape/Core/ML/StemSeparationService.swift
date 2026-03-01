import Foundation

public struct StemOutput: Sendable, Equatable {
    public let vocals: URL
    public let drums: URL
    public let bass: URL
    public let other: URL

    public init(vocals: URL, drums: URL, bass: URL, other: URL) {
        self.vocals = vocals
        self.drums = drums
        self.bass = bass
        self.other = other
    }
}

public protocol StemSeparationService: Sendable {
    func separateStems(inputFileURL: URL) async throws -> StemOutput
}
