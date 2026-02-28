//
//  purp_tapeApp.swift
//  purp tape
//
//  Created by Ilyes Djari on 28/02/2026.
//

import SwiftUI

@main
struct purp_tapeApp: App {
    init() {
        AppTelemetry.shared.start()
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
