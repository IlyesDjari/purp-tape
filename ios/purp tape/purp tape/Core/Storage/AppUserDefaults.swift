import Foundation

enum AppDefaultsKey: String {
    case cachedProjectsCount = "cached_projects_count"
}

final class AppUserDefaults: @unchecked Sendable {
    static let shared = AppUserDefaults()

    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    func setInt(_ value: Int, for key: AppDefaultsKey) {
        defaults.set(value, forKey: key.rawValue)
    }

    func int(for key: AppDefaultsKey, default defaultValue: Int = 0) -> Int {
        if defaults.object(forKey: key.rawValue) == nil {
            return defaultValue
        }
        return defaults.integer(forKey: key.rawValue)
    }
}