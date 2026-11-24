// swift-tools-version: 6.2
import PackageDescription

// Shared by the Arion app and its keyboard extension: Sunbeam design, the
// localized catalog, dictation domain rules, the app–keyboard bridge and the
// Rime typing engine (`pnpm mobile:rime` stages ArionRime.xcframework).
let package = Package(
    name: "ArionKit",
    defaultLocalization: "en",
    platforms: [.iOS(.v26)],
    products: [
        .library(name: "ArionKit", targets: ["ArionKit"]),
    ],
    targets: [
        .target(name: "ArionKit", dependencies: ["ArionRime"], resources: [.process("Resources")]),
        .binaryTarget(name: "ArionRime", path: "Frameworks/ArionRime.xcframework"),
    ]
)
