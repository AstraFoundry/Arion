// swift-tools-version: 6.2
import PackageDescription

// Shared by the Arion app and its keyboard extension: Sunbeam design, the
// localized catalog, dictation domain rules and the app–keyboard bridge.
let package = Package(
    name: "ArionKit",
    defaultLocalization: "en",
    platforms: [.iOS(.v26)],
    products: [
        .library(name: "ArionKit", targets: ["ArionKit"]),
    ],
    targets: [
        .target(name: "ArionKit", resources: [.process("Resources")]),
    ]
)
