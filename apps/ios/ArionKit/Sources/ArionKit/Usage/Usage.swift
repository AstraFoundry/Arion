import Foundation

/// The dictations committed on one local calendar day: numbers and scene
/// names only, never text (005). Mirrors `domain.UsageDay`.
public struct UsageDay: Sendable, Equatable, Codable {
    /// The local day as yyyy-MM-dd.
    public var date: String
    public var sessions = 0
    public var speechMs: Int64 = 0
    public var words = 0
    /// Words of polished dictations by scene raw value.
    public var scenes: [String: Int] = [:]

    public init(date: String) { self.date = date }

    public var polishedWords: Int { scenes.values.reduce(0, +) }

    /// Average speed, 0 before any speech.
    public var wordsPerMinute: Int {
        speechMs > 0 ? Int((Double(words) * 60000 / Double(speechMs)).rounded()) : 0
    }
}

/// Reads the per-day counts the keyboard records. Implemented by
/// `UsageLedger`, stored in the App Group.
public protocol UsageReading: Sendable {
    /// Every day with a dictation, oldest first.
    func days() -> [UsageDay]
    /// The sum of the days from the day of `from` on; nil sums every day.
    func total(since from: Date?) -> UsageDay
    /// The `count` days ending on the day of `end`, oldest first, with zero
    /// counts for days without dictation.
    func days(ending end: Date, count: Int) -> [UsageDay]
}
