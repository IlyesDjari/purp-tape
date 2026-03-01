import Foundation
import Testing
import purp_tape

struct SecuritySwiftTesting {
    @Test("Secure vault round-trip and clear")
    func keychainRoundTripAndClear() async throws {
        let vault = SecureEnclaveKeychainVault(service: "ilyes.purp-tape.swift-testing")
        let session = AuthSession(
            accessToken: "access-token",
            refreshToken: "refresh-token",
            userID: UUID(),
            expiresAt: Date().addingTimeInterval(300)
        )

        try await vault.saveSession(session)
        let loaded = try await vault.loadSession()
        #expect(loaded == session)

        try await vault.clearSession()
        let cleared = try await vault.loadSession()
        #expect(cleared == nil)
    }

    @Test("Signed URL cache wipe semantics")
    func signedURLWipe() async {
        let cache = InMemorySignedURLCache()
        let url = URL(string: "https://example.r2.dev/file.wav?sig=abc")!

        await cache.set(url, for: "track-v3", expiresAt: Date().addingTimeInterval(120))
        let beforeWipe = await cache.get(for: "track-v3")
        #expect(beforeWipe == url)

        await cache.wipeAll()
        let afterWipe = await cache.get(for: "track-v3")
        #expect(afterWipe == nil)
    }
}
