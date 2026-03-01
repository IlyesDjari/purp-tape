import Foundation
import os

public protocol PerformanceLogging: Sendable {
    func begin(_ name: StaticString) -> OSSignpostID
    func end(_ name: StaticString, id: OSSignpostID)
}

public struct PerformanceLogger: PerformanceLogging {
    private let log = OSLog(subsystem: "ilyes.purp-tape", category: "performance")

    public init() {}

    public func begin(_ name: StaticString) -> OSSignpostID {
        let id = OSSignpostID(log: log)
        os_signpost(.begin, log: log, name: name, signpostID: id)
        return id
    }

    public func end(_ name: StaticString, id: OSSignpostID) {
        os_signpost(.end, log: log, name: name, signpostID: id)
    }
}
