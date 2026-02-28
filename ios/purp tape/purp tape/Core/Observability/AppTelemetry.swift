import Foundation
import MetricKit
import os

@MainActor
public final class AppTelemetry: NSObject {
    public static let shared = AppTelemetry()

    private let logger = Logger(subsystem: "ilyes.purp-tape", category: "telemetry")
    private var hasStarted = false

    private override init() {
        super.init()
    }

    public func start() {
        guard !hasStarted else { return }
        hasStarted = true
        MXMetricManager.shared.add(self)
    }

    deinit {
        MXMetricManager.shared.remove(self)
    }

    public func record(error: Error, context: String) {
        logger.error("Non-fatal error in \(context, privacy: .public): \(error.localizedDescription, privacy: .public)")
    }
}

extension AppTelemetry: MXMetricManagerSubscriber {
    public nonisolated func didReceive(_ payloads: [MXDiagnosticPayload]) {
        let crashCount = payloads.reduce(0) { partialResult, payload in
            partialResult + (payload.crashDiagnostics?.count ?? 0)
        }

        guard crashCount > 0 else { return }

        Task { @MainActor in
            self.logger.error("Received \(crashCount, privacy: .public) crash diagnostics payload(s)")
        }
    }
}
