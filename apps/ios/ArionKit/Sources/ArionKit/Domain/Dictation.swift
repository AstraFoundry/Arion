import Foundation

/// Text recognized so far in one dictation: the finalized part, which no
/// longer changes, and the volatile tail the recognizer may still revise.
/// A draft is never target text; only a commit writes into the editor.
public struct Draft: Equatable, Sendable {
    public var finalized: String
    public var volatile: String

    public init(finalized: String = "", volatile: String = "") {
        self.finalized = finalized
        self.volatile = volatile
    }

    public var text: String { finalized + volatile }
    public var isEmpty: Bool { text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
}

/// Why a dictation could not start or finish. Each case is shown to the user
/// with the step that resolves it; none falls back silently.
public enum DictationFailure: Error, Equatable, Sendable {
    /// The keyboard runs without Full Access, so WebKit cannot open the microphone.
    case fullAccessRequired
    /// Microphone permission is missing; only the app can ask for it.
    case microphoneDenied
    /// The recognizer has no on-device model for the language yet.
    case languageUnavailable
    /// The on-device model for the language is still downloading.
    case modelInstalling
    /// The microphone stopped: another app took it, or the route ended.
    case captureEnded
    /// The recognizer failed.
    case recognitionFailed
    /// Nothing was recognized.
    case nothingHeard
}

/// One dictation's lifecycle as the keyboard shows it.
public enum DictationPhase: Equatable, Sendable {
    case idle
    /// The microphone is opening; speech is not yet captured.
    case starting
    case listening(Draft, level: Float)
    /// Capture stopped; the recognizer is finalizing the tail.
    case finishing(Draft)
    /// The finalized transcript is being polished on device.
    case polishing(Draft)
    case failed(DictationFailure)

    public var draft: Draft? {
        switch self {
        case .listening(let d, _), .finishing(let d), .polishing(let d): d
        default: nil
        }
    }

    public var isActive: Bool {
        switch self {
        case .starting, .listening, .finishing, .polishing: true
        default: false
        }
    }
}

/// Identifies one dictation. A late result carrying an older generation is
/// dropped, so cancellation and restarts cannot resurrect stale text.
public struct DictationGeneration: Hashable, Sendable, Comparable {
    public let value: UInt64
    public init(_ value: UInt64) { self.value = value }
    public func next() -> DictationGeneration { DictationGeneration(value &+ 1) }
    public static func < (a: Self, b: Self) -> Bool { a.value < b.value }
}
