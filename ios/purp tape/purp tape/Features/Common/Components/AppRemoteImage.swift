import SwiftUI
import ImageIO

public struct AppRemoteImage: View {
    public let data: Data?
    public let url: URL?
    public var fallbackSystemName: String = "photo"
    public var fallbackColor: Color = PurpTapeColors.surface
    public var debugLabel: String? = nil

    @State private var remoteData: Data?
    @State private var isLoading: Bool = false

    public init(data: Data? = nil, url: URL? = nil, fallbackSystemName: String = "photo", fallbackColor: Color = PurpTapeColors.surface, debugLabel: String? = nil) {
        self.data = data
        self.url = url
        self.fallbackSystemName = fallbackSystemName
        self.fallbackColor = fallbackColor
        self.debugLabel = debugLabel
    }

    public var body: some View {
        Group {
            if let imageData = data ?? remoteData {
                let byteCount = imageData.count
                let prefix = imageData.prefix(12).map { String(format: "%02X", $0) }.joined(separator: " ")
                if let label = debugLabel {
                    let _ = print("[AppRemoteImage] \(label) size: \(byteCount) bytes, prefix: \(prefix)")
                } else {
                    let _ = print("[AppRemoteImage] size: \(byteCount) bytes, prefix: \(prefix)")
                }

                if let source = CGImageSourceCreateWithData(imageData as CFData, nil),
                   let cgImage = CGImageSourceCreateImageAtIndex(source, 0, nil) {
                    Image(decorative: cgImage, scale: 1)
                        .resizable()
                        .scaledToFill()
                        .clipped()
                } else {
                    fallbackView
                }
            } else if isLoading {
                ProgressView()
            } else {
                fallbackView
            }
        }
        .task {
            await fetchRemoteImageIfNeeded()
        }
    }

    @ViewBuilder
    private var fallbackView: some View {
        ZStack {
            fallbackColor
            Image(systemName: fallbackSystemName)
                .resizable()
                .scaledToFit()
                .frame(width: 64, height: 64)
                .foregroundColor(.gray.opacity(0.3))
        }
    }

    private func fetchRemoteImageIfNeeded() async {
        guard remoteData == nil, data == nil, let url else { return }
        isLoading = true
        do {
            let (fetchedData, _) = try await URLSession.shared.data(from: url)
            remoteData = fetchedData
        } catch {
            print("[AppRemoteImage] Failed to fetch image from \(url): \(error.localizedDescription)")
        }
        isLoading = false
    }
}
