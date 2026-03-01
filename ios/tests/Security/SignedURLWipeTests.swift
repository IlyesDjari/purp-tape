import XCTest
@testable import purp_tape

final class SignedURLWipeTests: XCTestCase {
    func testWipeAllRemovesEphemeralURLs() async {
        let cache = InMemorySignedURLCache()
        let url = URL(string: "https://example.r2.dev/file.wav?sig=123")!

        await cache.set(url, for: "track-v1", expiresAt: Date().addingTimeInterval(300))
        let beforeWipe = await cache.get(for: "track-v1")
        XCTAssertEqual(beforeWipe, url)

        await cache.wipeAll()
        let afterWipe = await cache.get(for: "track-v1")
        XCTAssertNil(afterWipe)
    }
}
