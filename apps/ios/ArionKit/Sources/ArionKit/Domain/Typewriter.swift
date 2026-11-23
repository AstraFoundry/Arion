import Foundation

/// Paces streaming recognition text so it flows out at a steady, readable
/// speed instead of jumping by whole partials: the desktop pill's typewriter
/// (`apps/desktop/ui/entities/dictation/typewriter.go`, after Doubao's
/// ASRTypeWriter; docs/research/voice-typewriter.md). A spring keeps a small
/// backlog of unshown characters: the further the display falls behind the
/// recognizer, the faster it types; caught up, it slows to a calm pace.
/// Revised characters already shown change in place, so the line never
/// retreats and only new characters are typed; a revision shorter than what
/// is shown cuts the display to it. Characters are grapheme clusters, so an
/// emoji or a combining mark never splits.
public struct Typewriter: Sendable {
    /// What the draft shows now.
    public struct Frame: Equatable, Sendable {
        public var text: String
        /// Opacities of the characters at the end of `text` still fading in,
        /// oldest first; the characters before them are opaque.
        public var opacity: [Double]
        /// Another frame is needed.
        public var animating: Bool
    }

    /// The backlog the spring settles around, in characters.
    static let balance = 6.0
    /// The pace at balance; `elastic` shortens or lengthens it by this
    /// fraction per character of backlog above or below balance.
    static let interval = 0.045
    static let elastic = 0.25
    /// The pace stays between these. A finished recognition drains at the
    /// fastest pace, so the final text never waits on the animation long.
    static let minInterval = 0.012
    static let maxInterval = 0.090
    /// How long a new character takes to reach full opacity, from `fadeFrom`.
    public static let fadeIn = 0.150
    static let fadeFrom = 0.35

    private var session: UUID?
    private var shown = 0.0
    private var appeared: [TimeInterval] = []
    private var last: TimeInterval = 0

    public init() {}

    /// Advances to `now` toward `text` of `session`. `final` means recognition
    /// has finished, so the rest drains at full speed.
    public mutating func update(session: UUID, text: String, final: Bool, now: TimeInterval) -> Frame {
        if session != self.session {
            self = Typewriter()
            self.session = session
            last = now
        }
        let target = Array(text)
        if Int(shown) > target.count {
            shown = Double(target.count)
            appeared.removeLast(appeared.count - target.count)
        }
        var elapsed = now - last
        if elapsed < 0 || elapsed > 0.25 {
            // A stalled or first frame advances one frame's worth, not a jump.
            elapsed = 0.016
        }
        last = now
        let backlog = Double(target.count) - shown
        var pace = Self.minInterval
        if !final {
            let scale = max(0.1, 1 + Self.elastic * (backlog - Self.balance))
            pace = min(Self.maxInterval, max(Self.minInterval, Self.interval / scale))
        }
        shown = min(Double(target.count), shown + elapsed / pace)
        while appeared.count < Int(shown) { appeared.append(now) }
        var frame = Frame(text: String(target[..<Int(shown)]), opacity: [], animating: Int(shown) < target.count)
        for time in appeared.reversed() {
            let age = now - time
            if age >= Self.fadeIn { break }
            frame.opacity.insert(Self.fadeFrom + (1 - Self.fadeFrom) * age / Self.fadeIn, at: 0)
            frame.animating = true
        }
        return frame
    }
}
