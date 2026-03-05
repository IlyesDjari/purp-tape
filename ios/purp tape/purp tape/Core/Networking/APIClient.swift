import Foundation

public enum APIClientError: Error, Sendable {
    case invalidResponse
    case unauthorized
    case rateLimited
    case serverError(statusCode: Int)
    case transportError(String)
}

extension APIClientError: LocalizedError {
    public var errorDescription: String? {
        switch self {
        case .invalidResponse:
            return "Invalid response from server."
        case .unauthorized:
            return "You are not authorized. Please sign in again."
        case .rateLimited:
            return "Too many requests. Please try again in a moment."
        case .serverError(let statusCode):
            return "Server error (HTTP \(statusCode))."
        case .transportError(let message):
            return message
        }
    }
}

public protocol APIClient: Sendable {
    func send<T: Decodable & Sendable>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T
    func upload<T: Decodable & Sendable>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T
}
