import Foundation
import Testing
import purp_tape

struct AuthSessionSwiftTesting {
    @Test("Session expiration reflects current time")
    func sessionExpiration() {
        let active = AuthSession(
            accessToken: "a",
            refreshToken: "r",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(60)
        )
        let expired = AuthSession(
            accessToken: "a",
            refreshToken: "r",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(-60)
        )

        #expect(active.isExpired == false)
        #expect(expired.isExpired)
    }
}
