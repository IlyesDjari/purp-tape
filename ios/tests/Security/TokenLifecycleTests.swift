import XCTest
@testable import purp_tape

final class TokenLifecycleTests: XCTestCase {
    func testSessionRoundTripAndClear() async throws {
        let vault = SecureEnclaveKeychainVault(service: "ilyes.purp-tape.tests")
        let session = AuthSession(
            accessToken: "access-token",
            refreshToken: "refresh-token",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(300)
        )

        try await vault.saveSession(session)
        let loaded = try await vault.loadSession()

        XCTAssertEqual(loaded, session)

        try await vault.clearSession()
        let afterClear = try await vault.loadSession()
        XCTAssertNil(afterClear)
    }
}
