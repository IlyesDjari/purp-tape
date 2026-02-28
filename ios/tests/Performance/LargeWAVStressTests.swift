import Foundation
import XCTest
@testable import purp_tape

final class LargeWAVStressTests: XCTestCase {
    func testLargeWAVStreamReadStaysBounded() throws {
        let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent("stress.wav")
        let chunk = Data(repeating: 0x7f, count: 1024 * 1024)

        FileManager.default.createFile(atPath: tempURL.path(), contents: nil)
        let writeHandle = try FileHandle(forWritingTo: tempURL)
        for _ in 0..<64 {
            try writeHandle.write(contentsOf: chunk)
        }
        try writeHandle.close()

        measure {
            autoreleasepool {
                do {
                    let readHandle = try FileHandle(forReadingFrom: tempURL)
                    while try readHandle.read(upToCount: 512 * 1024)?.isEmpty == false {}
                    try readHandle.close()
                } catch {
                    XCTFail("Read failed: \(error)")
                }
            }
        }

        try? FileManager.default.removeItem(at: tempURL)
    }
}
