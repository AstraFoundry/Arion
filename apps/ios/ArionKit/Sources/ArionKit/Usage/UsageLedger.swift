import Foundation

/// Counts text as people count what they wrote, as `domain.CountWords`: each
/// Han, kana or Hangul character is one word, as are runs of other letters
/// and digits joined by apostrophes, hyphens or decimal points.
public enum WordCount {
    public static func count(_ text: String) -> Int {
        let scalars = Array(text.unicodeScalars)
        var words = 0
        var inWord = false
        for (index, scalar) in scalars.enumerated() {
            if isCJK(scalar) {
                words += 1
                inWord = false
            } else if isWordPart(scalar) {
                if !inWord { words += 1 }
                inWord = true
            } else if inWord, "'’-.".unicodeScalars.contains(scalar), index + 1 < scalars.count,
                      isLetterOrDigit(scalars[index + 1]), !isCJK(scalars[index + 1]) {
                // Joined: don't, e-mail, 3.5
            } else {
                inWord = false
            }
        }
        return words
    }

    private static func isCJK(_ s: Unicode.Scalar) -> Bool {
        switch s.properties.generalCategory {
        case .otherLetter:
            break
        default:
            return false
        }
        let v = s.value
        return (0x3040...0x30FF).contains(v) || (0x31F0...0x31FF).contains(v) || (0xFF66...0xFF9D).contains(v)
            || (0x1100...0x11FF).contains(v) || (0x3130...0x318F).contains(v) || (0xAC00...0xD7AF).contains(v) || (0xA960...0xA97F).contains(v) || (0xD7B0...0xD7FF).contains(v)
            || s.properties.isIdeographic
    }

    private static func isLetterOrDigit(_ s: Unicode.Scalar) -> Bool {
        s.properties.isAlphabetic || s.properties.numericType != nil
    }

    private static func isWordPart(_ s: Unicode.Scalar) -> Bool {
        if isLetterOrDigit(s) { return true }
        switch s.properties.generalCategory {
        case .nonspacingMark, .spacingMark, .enclosingMark: return true
        default: return false
        }
    }
}

/// The per-day counts the keyboard records and the app shows, as JSON in the
/// App Group container. Mirrors `domain.UsageLog`: one entry per local day,
/// oldest first, numbers and scene names only.
public final class UsageLedger: UsageReading, @unchecked Sendable {
    private let url: URL?
    private let lock = NSLock()

    /// The ledger in the App Group, or at `url` in tests.
    public init(url: URL? = AppGroup.container?.appending(path: "usage.json")) {
        self.url = url
    }

    static let dayKey: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    public static func key(_ date: Date) -> String { dayKey.string(from: date) }

    /// Adds one committed dictation to its day. The text is counted, not kept.
    public func record(speechMs: Int64, text: String, scene: Scene?, at date: Date = .now) throws {
        lock.lock()
        defer { lock.unlock() }
        var log = read()
        let key = Self.key(date)
        let words = WordCount.count(text)
        var day = log.first { $0.date == key } ?? UsageDay(date: key)
        day.sessions += 1
        day.speechMs += max(speechMs, 0)
        day.words += words
        if let scene { day.scenes[scene.rawValue, default: 0] += words }
        log.removeAll { $0.date == key }
        log.append(day)
        log.sort { $0.date < $1.date }
        try write(log)
    }

    public func days() -> [UsageDay] {
        lock.lock()
        defer { lock.unlock() }
        return read()
    }

    public func total(since from: Date?) -> UsageDay {
        let first = from.map(Self.key) ?? ""
        var total = UsageDay(date: "")
        for day in days() where day.date >= first {
            total.sessions += day.sessions
            total.speechMs += day.speechMs
            total.words += day.words
            for (scene, words) in day.scenes { total.scenes[scene, default: 0] += words }
        }
        return total
    }

    public func days(ending end: Date, count: Int) -> [UsageDay] {
        let byDate = Dictionary(days().map { ($0.date, $0) }, uniquingKeysWith: { a, _ in a })
        return (0..<count).map { index in
            let date = Calendar.current.date(byAdding: .day, value: index - count + 1, to: end) ?? end
            let key = Self.key(date)
            return byDate[key] ?? UsageDay(date: key)
        }
    }

    // The keyboard writes while the app reads, so every call reads the file.
    private func read() -> [UsageDay] {
        guard let url, let data = try? Data(contentsOf: url) else { return [] }
        return (try? JSONDecoder().decode([UsageDay].self, from: data)) ?? []
    }

    private func write(_ log: [UsageDay]) throws {
        guard let url else { throw CocoaError(.fileNoSuchFile) }
        try JSONEncoder().encode(log).write(to: url, options: .atomic)
    }
}
