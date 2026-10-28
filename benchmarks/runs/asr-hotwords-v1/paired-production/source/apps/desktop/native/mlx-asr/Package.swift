// swift-tools-version:6.2
import PackageDescription

let package = Package(
    name: "ArionASR",
    platforms: [.macOS(.v14)],
    products: [.executable(name: "arion-asr", targets: ["ArionASR"])],
    dependencies: [
        .package(url: "https://github.com/Blaizzy/mlx-audio-swift.git", revision: "a5506c1c7978aeea9a3295f83d43beff3f06bcdf")
    ],
    targets: [.executableTarget(name: "ArionASR", dependencies: [
        .product(name: "MLXAudioSTT", package: "mlx-audio-swift"),
        .product(name: "MLXAudioCore", package: "mlx-audio-swift")
    ])]
)
