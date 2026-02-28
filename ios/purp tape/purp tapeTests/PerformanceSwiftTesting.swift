import Foundation
import Testing
import purp_tape

struct PerformanceSwiftTesting {
    @Test("Large WAV sequential read stress", .timeLimit(.minutes(1)))
    func largeWavReadStress() throws {
        let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent("swift-testing-stress.wav")
        defer { try? FileManager.default.removeItem(at: tempURL) }

        let oneMB = Data(repeating: 0x7f, count: 1024 * 1024)
        FileManager.default.createFile(atPath: tempURL.path(), contents: nil)
        let writer = try FileHandle(forWritingTo: tempURL)
        for _ in 0..<64 {
            try writer.write(contentsOf: oneMB)
        }
        try writer.close()

        let clock = ContinuousClock()
        let elapsed = clock.measure {
            do {
                let reader = try FileHandle(forReadingFrom: tempURL)
                while let chunk = try reader.read(upToCount: 512 * 1024), !chunk.isEmpty {}
                try reader.close()
            } catch {
                Issue.record("Read stress failed: \(error)")
            }
        }

        #expect(elapsed < .seconds(5))
    }
}
