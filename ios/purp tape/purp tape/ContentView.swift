//
//  ContentView.swift
//  purp tape
//
//  Created by Ilyes Djari on 28/02/2026.
//

import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack {
            Image(systemName: "globe")
                .imageScale(.large)
                .foregroundStyle(.tint)
            Text(String(localized: "home.welcome", defaultValue: "Hello, world!"))
        }
        .padding()
    }
}

#Preview {
    ContentView()
}
