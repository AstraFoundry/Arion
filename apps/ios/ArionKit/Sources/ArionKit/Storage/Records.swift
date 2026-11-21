import Foundation

/// A JSON file in the App Group container, protected by iOS data protection
/// until the first unlock so the keyboard can read it while the phone is in use.
struct SharedFile<Value: Codable & Sendable>: Sendable {
    let name: String

    var url: URL? { AppGroup.container?.appendingPathComponent(name) }

    func load() throws -> Value? {
        guard let url, FileManager.default.fileExists(atPath: url.path) else { return nil }
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(Value.self, from: Data(contentsOf: url))
    }

    func save(_ value: Value) throws {
        guard let url else { throw CocoaError(.fileNoSuchFile) }
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        try encoder.encode(value).write(to: url, options: [.atomic, .completeFileProtectionUntilFirstUserAuthentication])
    }
}

/// Words the user wants recognized and kept as written: names, products and
/// terms. They bias recognition and are preserved by polishing.
public final class Vocabulary: @unchecked Sendable {
    private let file = SharedFile<[String]>(name: "vocabulary.json")
    public static let limit = 200

    public init() {}

    public func terms() throws -> [String] { try file.load() ?? [] }

    /// Adds a term once, ignoring case and surrounding space. Returns false for
    /// an empty or duplicate term.
    @discardableResult
    public func add(_ term: String) throws -> Bool {
        let trimmed = term.trimmingCharacters(in: .whitespacesAndNewlines)
        var all = try terms()
        guard !trimmed.isEmpty, all.count < Self.limit,
              !all.contains(where: { $0.caseInsensitiveCompare(trimmed) == .orderedSame }) else { return false }
        all.insert(trimmed, at: 0)
        try file.save(all)
        return true
    }

    public func remove(_ term: String) throws {
        try file.save(try terms().filter { $0 != term })
    }
}

/// One committed dictation, kept only when history is on.
public struct HistoryEntry: Codable, Equatable, Sendable, Identifiable {
    public let id: UUID
    public let date: Date
    public let text: String
    /// The transcript before polishing, when polishing changed it.
    public let original: String?

    public init(id: UUID = UUID(), date: Date = Date(), text: String, original: String?) {
        self.id = id
        self.date = date
        self.text = text
        self.original = original == text ? nil : original
    }
}

/// Recent dictations on this device. Nothing is recorded unless the user turns
/// history on; turning it off erases what was kept.
public final class History: @unchecked Sendable {
    private let file = SharedFile<[HistoryEntry]>(name: "history.json")
    public static let limit = 100

    public init() {}

    public func entries() throws -> [HistoryEntry] { try file.load() ?? [] }

    public func record(_ entry: HistoryEntry) throws {
        try file.save(Array(([entry] + entries()).prefix(Self.limit)))
    }

    public func remove(_ id: UUID) throws {
        try file.save(try entries().filter { $0.id != id })
    }

    public func erase() throws {
        guard let url = file.url, FileManager.default.fileExists(atPath: url.path) else { return }
        try FileManager.default.removeItem(at: url)
    }
}
