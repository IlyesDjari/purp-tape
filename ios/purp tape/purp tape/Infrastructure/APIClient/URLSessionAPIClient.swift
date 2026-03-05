import Foundation

// PERFORMANCE: Shared decoder instance avoids allocations on every response
private let sharedDecoder = JSONDecoder()

// PERFORMANCE: Optimized URLSession configuration for API calls
private func createOptimizedURLSession() -> URLSession {
    let config = URLSessionConfiguration.default
    
    // Aggressive timeouts - fail fast on network issues
    config.timeoutIntervalForRequest = 15      // 15s instead of 60s default
    config.timeoutIntervalForResource = 30     // 30s instead of 300s+ default
    
    // Reliability improvements
    config.waitsForConnectivity = true         // Don't fail on transient network changes
    config.shouldUseExtendedBackgroundIdleMode = false
    
    // Performance tuning
    config.httpMaximumConnectionsPerHost = 4   // Connection pooling
    config.httpShouldUsePipelining = true      // Pipeline HTTP requests
    
    return URLSession(configuration: config)
}

public actor URLSessionAPIClient: APIClient {
    private let baseURL: URL
    private let session: URLSession
    private let authService: AuthService
    private let logger = DebugLogger(category: "api.client")
    
    // SECURITY: Track 401 retries to prevent infinite loops
    private var retryCount = 0
    private let maxRetries = 1

    public init(baseURL: URL, session: URLSession? = nil, authService: AuthService) {
        self.baseURL = baseURL
        self.session = session ?? createOptimizedURLSession()
        self.authService = authService
    }

    public func send<T: Decodable & Sendable>(_ endpoint: Endpoint, decode type: T.Type) async throws -> T {
        return try await sendWithRetry(endpoint: endpoint, decode: type)
    }

    public func upload<T: Decodable & Sendable>(_ endpoint: Endpoint, fileURL: URL, decode type: T.Type) async throws -> T {
        let request = try await buildRequest(endpoint: endpoint)
        let (data, response) = try await session.upload(for: request, fromFile: fileURL)
        return try decodeResponse(data: data, response: response, as: type)
    }
    
    /// SECURITY: Automatic retry on 401 with token refresh
    private func sendWithRetry<T: Decodable & Sendable>(
        endpoint: Endpoint,
        decode type: T.Type,
        attempt: Int = 0
    ) async throws -> T {
        do {
            let request = try await buildRequest(endpoint: endpoint)
            let (data, response) = try await session.data(for: request)
            return try decodeResponse(data: data, response: response, as: type)
        } catch APIClientError.unauthorized where attempt < maxRetries {
            logger.warning("HTTP 401, attempting token refresh (attempt \(attempt + 1))")
            
            do {
                _ = try await authService.refreshIfNeeded()
                try await Task.sleep(nanoseconds: 100_000_000) // 0.1s
                return try await sendWithRetry(endpoint: endpoint, decode: type, attempt: attempt + 1)
            } catch {
                logger.error("Token refresh failed during retry: \(error.localizedDescription)")
                throw APIClientError.unauthorized
            }
        }
    }

    private func buildRequest(endpoint: Endpoint) async throws -> URLRequest {
        guard let session = try await authService.currentSession() else {
            logger.error("No active session - cannot build request")
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
                return try sharedDecoder.decode(T.self, from: data)
            } catch {
                throw APIClientError.transportError("Decoding failure: \(error.localizedDescription)")
            }
        case 401:
            throw APIClientError.unauthorized
        case 429:
            throw APIClientError.rateLimited
        default:
            let bodyText = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
            if let bodyText, !bodyText.isEmpty {
                throw APIClientError.transportError("HTTP \(httpResponse.statusCode): \(bodyText)")
            }
            throw APIClientError.serverError(statusCode: httpResponse.statusCode)
        }
    }
}
