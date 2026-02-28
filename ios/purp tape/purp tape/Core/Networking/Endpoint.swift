import Foundation

public struct Endpoint: Sendable {
    public let path: String
    public let method: String
    public let headers: [String: String]
    public let body: Data?

    public init(path: String, method: String, headers: [String: String] = [:], body: Data? = nil) {
        self.path = path
        self.method = method
        self.headers = headers
        self.body = body
    }
}
