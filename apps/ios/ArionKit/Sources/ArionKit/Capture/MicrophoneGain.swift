@preconcurrency import AVFAudio

/// Gain applied to the captured voice before recognition, for quiet
/// microphones or soft speakers. Samples clip at full scale.
public enum MicrophoneGain {
    /// The choices the app offers, in decibels; 0 is off.
    public static let choices: [Int] = [0, 3, 6, 9, 12]

    public static func factor(decibels: Int) -> Float { Float(pow(10, Double(decibels) / 20)) }

    /// Scales 16-bit PCM in place.
    public static func apply(_ buffer: AVAudioPCMBuffer, decibels: Int) {
        guard decibels != 0, let samples = buffer.int16ChannelData?[0] else { return }
        let gain = factor(decibels: decibels)
        for i in 0..<Int(buffer.frameLength) {
            samples[i] = Int16(max(-32768, min(32767, (Float(samples[i]) * gain).rounded())))
        }
    }
}
