import Foundation

public struct AuthSession: Sendable, Equatable, Codable {
    public let accessToken: String
    public let refreshToken: String
    public let userID: UUID
    public let expiresAt: Date

    public init(accessToken: String, refreshToken: String, userID: UUID, expiresAt: Date) {
        self.accessToken = accessToken
        self.refreshToken = refreshToken
        self.userID = userID
        self.expiresAt = expiresAt
    }

    public var isExpired: Bool {
        expiresAt <= Date()
    }
}
