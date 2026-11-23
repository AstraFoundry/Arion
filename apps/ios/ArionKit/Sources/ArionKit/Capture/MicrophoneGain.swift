@preconcurrency import AVFAudio

/// Gain applied to the captured voice before recognition, for quiet
/// microphones or soft speakers. Samples clip at full scale.
public enum MicrophoneGain {
    /// One setting and the speaking distance it suits.
    public struct Stop: Equatable, Sendable {
        public let decibels: Int
        public let centimeters: Int
    }

    /// The desktop's stops (`packages/core/application/gain.go`), quietest
    /// first: speech falls 6 dB per doubling of distance, so each 3 dB step
    /// suits a distance √2 farther; 0 dB is the device's own level.
    public static let stops: [Stop] = [(-6, 15), (-3, 21), (0, 30), (3, 42), (6, 60), (9, 85), (12, 120)]
        .map { Stop(decibels: $0.0, centimeters: $0.1) }
    public static let step = 3
    public static var choices: [Int] { stops.map(\.decibels) }

    /// The stop nearest a slider position.
    public static func nearest(_ value: Float) -> Stop {
        stops.min { abs(Float($0.decibels) - value) < abs(Float($1.decibels) - value) }!
    }

    public static func factor(decibels: Int) -> Float { Float(pow(10, Double(decibels) / 20)) }

    /// The meter level of an RMS amplitude after gain, as the desktop
    /// computes it (`Level` in `packages/core/application/dictation.go`):
    /// -60 dBFS to -10 dBFS fill 0...1, so quiet speech still moves the bars.
    public static func level(rms: Float, decibels: Int) -> Float {
        let db = 20 * log10(Double(rms * factor(decibels: decibels)) + 1e-9)
        return Float(max(0, min(1, (db + 60) / 50)))
    }

    /// Scales 16-bit PCM in place.
    public static func apply(_ buffer: AVAudioPCMBuffer, decibels: Int) {
        guard decibels != 0, let samples = buffer.int16ChannelData?[0] else { return }
        let gain = factor(decibels: decibels)
        for i in 0..<Int(buffer.frameLength) {
            samples[i] = Int16(max(-32768, min(32767, (Float(samples[i]) * gain).rounded())))
        }
    }
}
