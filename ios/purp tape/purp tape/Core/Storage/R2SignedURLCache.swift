import Foundation

public protocol SignedURLCache: Sendable {
    func set(_ url: URL, for key: String, expiresAt: Date) async
    func get(for key: String) async -> URL?
    func wipeAll() async
}

public actor InMemorySignedURLCache: SignedURLCache {
    private struct Entry: Sendable {
        let url: URL
        let expiresAt: Date
    }

    private var entries: [String: Entry] = [:]

    public init() {}

    public func set(_ url: URL, for key: String, expiresAt: Date) async {
        entries[key] = Entry(url: url, expiresAt: expiresAt)
    }

    public func get(for key: String) async -> URL? {
        guard let entry = entries[key] else { return nil }
        if entry.expiresAt <= Date() {
            entries[key] = nil
            return nil
        }
        return entry.url
    }

    public func wipeAll() async {
        entries.removeAll(keepingCapacity: false)
    }
}
