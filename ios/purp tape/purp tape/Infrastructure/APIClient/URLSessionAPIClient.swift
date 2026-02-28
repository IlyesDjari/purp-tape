import Foundation

public actor URLSessionAPIClient: APIClient {
    private let baseURL: URL
    private let session: URLSession
    private let authService: AuthService

    public init(baseURL: URL, session: URLSession = .shared, authService: AuthService) {
        self.baseURL = baseURL
        self.session = session
        self.authService = authService
    }

    public func send<T: Decodable & Sendable>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T {
        let request = try await buildRequest(endpoint: endpoint)
        let (data, response) = try await session.data(for: request)
        return try decodeResponse(data: data, response: response, as: type)
    }

    public func upload<T: Decodable & Sendable>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T {
        let request = try await buildRequest(endpoint: endpoint)
        let (data, response) = try await session.upload(for: request, fromFile: fileURL)
        return try decodeResponse(data: data, response: response, as: type)
    }

    private func buildRequest(endpoint: Endpoint) async throws -> URLRequest {
        guard let session = try await authService.currentSession() else {
            throw APIClientError.unauthorized
        }
        let fullURL = baseURL.appending(path: endpoint.path)
        var request = URLRequest(url: fullURL)
        request.httpMethod = endpoint.method
        request.httpBody = endpoint.body
        request.setValue("Bearer \(session.accessToken)", forHTTPHeaderField: "Authorization")
        endpoint.headers.forEach { key, value in
            request.setValue(value, forHTTPHeaderField: key)
        }
        return request
    }

    private func decodeResponse<T: Decodable & Sendable>(data: Data, response: URLResponse, as type: T.Type) throws -> T {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIClientError.invalidResponse
        }

        switch httpResponse.statusCode {
        case 200 ..< 300:
            do {
                return try JSONDecoder().decode(T.self, from: data)
            } catch {
                throw APIClientError.transportError("Decoding failure: \(error.localizedDescription)")
            }
        case 401:
            throw APIClientError.unauthorized
        case 429:
            throw APIClientError.rateLimited
        default:
            throw APIClientError.serverError(statusCode: httpResponse.statusCode)
        }
    }
}
