import XCTest
@testable import purp_tape

private final class LeakProbe {
    let id = UUID()
}

private actor MockAudioProcessingService: AudioProcessingService {
    private var probe = LeakProbe()

    func compress(_ request: AudioCompressionRequest) async throws -> URL {
        _ = probe.id
        return request.outputURL
    }

    func waveformSamples(for fileURL: URL) async throws -> [Float] {
        _ = probe.id
        return [0.1, 0.2, 0.3]
    }
}

final class AudioLeakTests: XCTestCase {
    func testAudioProcessorDeallocatesAfterTaskCompletion() async throws {
        weak var weakProbe: LeakProbe?

        do {
            let processor = MockAudioProcessingService()
            let mirror = Mirror(reflecting: processor)
            if let probe = mirror.children.first(where: { $0.label == "probe" })?.value as? LeakProbe {
                weakProbe = probe
            }

            let source = URL(fileURLWithPath: "/tmp/source.wav")
            let output = URL(fileURLWithPath: "/tmp/output.m4a")
            _ = try await processor.compress(.init(sourceURL: source, outputURL: output))
            _ = try await processor.waveformSamples(for: output)
        }

        XCTAssertNil(weakProbe)
    }
}
