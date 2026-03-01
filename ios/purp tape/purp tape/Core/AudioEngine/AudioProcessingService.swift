import AVFoundation
import Foundation

public struct AudioCompressionRequest: Sendable {
    public let sourceURL: URL
    public let outputURL: URL

    public init(sourceURL: URL, outputURL: URL) {
        self.sourceURL = sourceURL
        self.outputURL = outputURL
    }
}

public protocol AudioProcessingService: Sendable {
    func compress(_ request: AudioCompressionRequest) async throws -> URL
    func waveformSamples(for fileURL: URL) async throws -> [Float]
}
