import ArionRime
import CryptoKit
import Foundation
import Security

/// The typing layouts the keyboard offers besides the microphone (ADR 053).
public enum TypingLayout: String, CaseIterable, Sendable {
    /// Full Pinyin on a QWERTY grid.
    case pinyin
    /// Pinyin typed on the phone keypad's digit groups.
    case pinyin9
    /// English letters, inserted as typed.
    case english

    /// The Rime schema behind a Chinese layout; English needs none.
    var schema: String? {
        switch self {
        case .pinyin: "arion_pinyin"
        case .pinyin9: "arion_pinyin_9key"
        case .english: nil
        }
    }
}

/// Why typing could not start. The keyboard says so; it never types without
/// the engine or learns in plaintext.
public enum TypingFailure: Error, Equatable, Sendable {
    /// The shipped Rime data is missing from the keyboard bundle.
    case dataMissing
    /// The learning key is unavailable and the vault already holds learning.
    case learningKey
    case engine(String)
}

/// A composition's state as the keyboard draws it.
public struct TypingState: Equatable, Sendable {
    public var input = ""
    public var preedit = ""
    /// 9-key: the Pinyin spellings of the pending digits.
    public var syllables: [String] = []
    public var candidates: [String] = []
    public var isComposing: Bool { !input.isEmpty }
    public init() {}
}

/// One Rime session per layout, on the engine prebuilt into the keyboard.
/// Learning is kept encrypted with a device-only key; see `LearningKey`.
@MainActor
public final class TypingEngine {
    public private(set) var state = TypingState()
    private var session: ArionRimeSession = 0
    public private(set) var layout: TypingLayout
    /// How many candidates the bar shows before the expanded list.
    public static let barCandidates = 30

    nonisolated(unsafe) private static var started = false

    /// Starts the process-wide engine once; later calls return at once.
    public static func start(data: URL, support: URL) throws(TypingFailure) {
        if started { return }
        let build = data.appendingPathComponent("build/arion_pinyin.schema.yaml")
        guard FileManager.default.fileExists(atPath: build.path) else { throw .dataMissing }
        let user = support.appendingPathComponent("Rime", isDirectory: true)
        let vault = user.appendingPathComponent("learning", isDirectory: true)
        do {
            try FileManager.default.createDirectory(at: vault, withIntermediateDirectories: true,
                                                    attributes: [.posixPermissions: 0o700])
        } catch {
            throw .engine("user directory")
        }
        guard LearningKey.prepare(vaultHasData: LearningKey.vaultHasData(vault)) else { throw .learningKey }
        var cipher = ArionRimeCipher(context: nil, seal: LearningKey.seal, open: LearningKey.open)
        var reason = [CChar](repeating: 0, count: 256)
        let ok = arion_rime_start(data.path, user.path, vault.path, &cipher, &reason, reason.count)
        guard ok == 1 else { throw .engine(String(cString: reason)) }
        started = true
    }

    public init(layout: TypingLayout) {
        self.layout = layout
        open(layout)
    }

    isolated deinit {
        arion_rime_close(session)
    }

    /// Switches layout; an unfinished composition is dropped, not committed.
    public func select(_ layout: TypingLayout) {
        guard layout != self.layout else { return }
        self.layout = layout
        open(layout)
    }

    private func open(_ layout: TypingLayout) {
        arion_rime_close(session)
        session = 0
        state = TypingState()
        if let schema = layout.schema { session = arion_rime_open(schema) }
    }

    /// Feeds a key; returns text to insert now. English never composes.
    @discardableResult
    public func type(_ character: Character) -> String? {
        guard session != 0, let ascii = character.asciiValue else { return String(character) }
        let handled = arion_rime_type(session, CChar(bitPattern: ascii))
        let commit = refresh()
        if handled == 0, !state.isComposing { return (commit ?? "") + String(character) }
        return commit
    }

    /// Deletes the last typed key; false when there was nothing composing.
    public func backspace() -> Bool {
        guard session != 0, state.isComposing else { return false }
        arion_rime_backspace(session)
        refresh()
        return true
    }

    /// Chooses a candidate by its index in the whole list.
    public func choose(_ index: Int) -> String? {
        guard session != 0 else { return nil }
        arion_rime_select(session, index)
        return refresh()
    }

    /// 9-key: replaces the pending digits a syllable spells with its letters.
    public func chooseSyllable(_ syllable: String) {
        guard session != 0, let start = state.input.firstIndex(where: { ("2"..."9").contains($0) }) else { return }
        let digits = state.input[start...].prefix { ("2"..."9").contains($0) }
        guard digits.count >= syllable.count else { return }
        let replaced = String(state.input[..<start]) + syllable + String(state.input[start...].dropFirst(syllable.count))
        arion_rime_set_input(session, replaced)
        refresh()
    }

    /// The return key while composing: the raw letters, as typed.
    public func commitRaw() -> String? {
        guard session != 0, state.isComposing else { return nil }
        arion_rime_commit_raw(session)
        return refresh()
    }

    public func clear() {
        guard session != 0 else { return }
        arion_rime_clear(session)
        refresh()
    }

    /// Candidates beyond the bar, for the expanded list.
    public func candidates(from offset: Int, limit: Int) -> [String] {
        guard session != 0, limit > 0 else { return [] }
        var texts = [UnsafePointer<CChar>?](repeating: nil, count: limit)
        let count = arion_rime_candidates(session, offset, limit, &texts, nil)
        return texts.prefix(count).compactMap { $0.map { String(cString: $0) } }
    }

    @discardableResult
    private func refresh() -> String? {
        var raw = ArionRimeState()
        guard arion_rime_state(session, &raw) == 1 else { return nil }
        let commit = raw.commit.map { String(cString: $0) }
        var next = TypingState()
        next.input = raw.input.map { String(cString: $0) } ?? ""
        next.preedit = raw.preedit.map { String(cString: $0) } ?? ""
        if let syllables = raw.syllables {
            next.syllables = (0..<raw.syllableCount).compactMap { syllables[$0].map { String(cString: $0) } }
        }
        if next.isComposing { next.candidates = candidates(from: 0, limit: Self.barCandidates) }
        state = next
        return commit
    }
}

/// The 256-bit AES-GCM key for typing learning: device-only, never synced,
/// created only while the vault is empty so a lost key never silently
/// replaces learning it cannot read.
enum LearningKey {
    private static let service = "Arion typing learning"
    private static let account = "aes256-gcm-v1"
    nonisolated(unsafe) private static var key: SymmetricKey?

    static func vaultHasData(_ vault: URL) -> Bool {
        let names = (try? FileManager.default.contentsOfDirectory(atPath: vault.path)) ?? []
        return names.contains { $0 != ".learning.lock" && !$0.hasSuffix(".arion-userdb.lock") }
    }

    static func prepare(vaultHasData: Bool) -> Bool {
        if key != nil { return true }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecAttrSynchronizable as String: false,
            kSecReturnData as String: true,
        ]
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecSuccess, let data = result as? Data, data.count == 32 {
            key = SymmetricKey(data: data)
            return true
        }
        guard status == errSecItemNotFound, !vaultHasData else { return false }
        let fresh = SymmetricKey(size: .bits256)
        let add: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecAttrSynchronizable as String: false,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            kSecValueData as String: fresh.withUnsafeBytes { Data($0) },
        ]
        guard SecItemAdd(add as CFDictionary, nil) == errSecSuccess else { return false }
        key = fresh
        return true
    }

    static let seal: @convention(c) (UnsafeMutableRawPointer?, UnsafePointer<CChar>?, UnsafePointer<UInt8>?, Int,
                                     UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>?, UnsafeMutablePointer<Int>?) -> Int32 = {
        _, label, bytes, count, output, outputCount in
        guard let key, let label, let output, let outputCount else { return 0 }
        let plain = bytes.map { Data(bytes: $0, count: count) } ?? Data()
        guard let box = try? AES.GCM.seal(plain, using: key, authenticating: Data(String(cString: label).utf8)),
              let combined = box.combined else { return 0 }
        return copy(combined, output, outputCount)
    }

    static let open: @convention(c) (UnsafeMutableRawPointer?, UnsafePointer<CChar>?, UnsafePointer<UInt8>?, Int,
                                     UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>?, UnsafeMutablePointer<Int>?) -> Int32 = {
        _, label, bytes, count, output, outputCount in
        guard let key, let label, let bytes, let output, let outputCount,
              let box = try? AES.GCM.SealedBox(combined: Data(bytes: bytes, count: count)),
              let plain = try? AES.GCM.open(box, using: key, authenticating: Data(String(cString: label).utf8))
        else { return 0 }
        return copy(plain, output, outputCount)
    }

    private static func copy(_ data: Data, _ output: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>,
                             _ outputCount: UnsafeMutablePointer<Int>) -> Int32 {
        guard let buffer = malloc(max(data.count, 1))?.assumingMemoryBound(to: UInt8.self) else { return 0 }
        data.copyBytes(to: buffer, count: data.count)
        output.pointee = buffer
        outputCount.pointee = data.count
        return 1
    }
}
