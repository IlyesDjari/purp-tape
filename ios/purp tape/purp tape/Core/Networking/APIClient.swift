import Foundation

public enum APIClientError: Error, Sendable {
    case invalidResponse
    case unauthorized
    case rateLimited
    case serverError(statusCode: Int)
    case transportError(String)
}

public protocol APIClient: Sendable {
    func send<T: Decodable & Sendable>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T
    func upload<T: Decodable & Sendable>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T
}
