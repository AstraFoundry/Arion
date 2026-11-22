import AVFAudio
import Speech

/// The microphone and speech permissions only the app can request; the
/// keyboard relies on them.
enum Permissions {
    static var microphoneGranted: Bool { AVAudioApplication.shared.recordPermission == .granted }
    static var microphoneDenied: Bool { AVAudioApplication.shared.recordPermission == .denied }
    static var microphoneUndetermined: Bool { AVAudioApplication.shared.recordPermission == .undetermined }

    /// Asks for the microphone, then speech recognition.
    static func request() async -> Bool {
        let microphone = await AVAudioApplication.requestRecordPermission()
        guard microphone else { return false }
        let speech = await withCheckedContinuation { continuation in
            SFSpeechRecognizer.requestAuthorization { continuation.resume(returning: $0) }
        }
        return speech == .authorized
    }
}
