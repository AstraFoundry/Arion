import CoreGraphics

/// Which key a touch presses. Keys own their half of each gap and edge keys
/// own the border, as the Rime frontends Hamster, Trime and fcitx5 lay out
/// their touch areas, so a tap between keys never goes nowhere. Near a
/// boundary the next letter's feasibility can move the touch to the
/// neighbor, the invisible target resizing of Apple's US 8,232,973: geometry
/// wins everywhere else (docs/research/keyboard-touch-targets.md).
public enum KeyTargets {
    public struct Key: Equatable, Sendable {
        public var frame: CGRect
        /// The letter it types, when the input engine can judge it.
        public var letter: Character?
        public init(frame: CGRect, letter: Character? = nil) {
            self.frame = frame
            self.letter = letter
        }
    }

    /// The part of a key's width, at its edge toward a neighbor, where the
    /// next letter can decide the touch (LatinIME and FlorisBoard tune bands
    /// of this size; iOS publishes none).
    public static let band: CGFloat = 0.1

    /// The index of the key a touch at `point` presses. `continues` says
    /// whether a letter continues the input, or nil when it cannot tell.
    public static func resolve(_ point: CGPoint, in keys: [Key], continues: (Character) -> Bool?) -> Int? {
        guard let hit = keys.indices.min(by: { distance(point, keys[$0].frame) < distance(point, keys[$1].frame) })
        else { return nil }
        let frame = keys[hit].frame
        guard let letter = keys[hit].letter else { return hit }
        let right = point.x > frame.midX
        let edge = right ? frame.maxX - point.x : point.x - frame.minX
        guard edge <= band * frame.width else { return hit }
        // The letter key beside it in the same row, on the touch's side.
        let beside = keys.indices.filter { index in
            index != hit && keys[index].letter != nil && abs(keys[index].frame.midY - frame.midY) < frame.height / 2
                && (right ? keys[index].frame.minX >= frame.maxX : keys[index].frame.maxX <= frame.minX)
        }
        guard let next = beside.min(by: { abs(keys[$0].frame.midX - frame.midX) < abs(keys[$1].frame.midX - frame.midX) }),
              let other = keys[next].letter
        else { return hit }
        return continues(letter) == false && continues(other) == true ? next : hit
    }

    static func distance(_ point: CGPoint, _ rect: CGRect) -> CGFloat {
        let dx = max(rect.minX - point.x, 0, point.x - rect.maxX)
        let dy = max(rect.minY - point.y, 0, point.y - rect.maxY)
        return (dx * dx + dy * dy).squareRoot()
    }
}

/// Whether typed letters can still become Pinyin: complete syllables of the
/// shipped dictionary (`arion_syllables.txt`), then the start of one. It says
/// what is feasible, not what is likely, so it only breaks near-ties.
public struct PinyinSpelling: Sendable {
    private let syllables: Set<String>
    private let prefixes: Set<String>
    private let longest: Int

    public init(syllables: [String]) {
        self.syllables = Set(syllables)
        var prefixes = Set<String>()
        for syllable in syllables {
            for end in syllable.indices.dropFirst() { prefixes.insert(String(syllable[..<end])) }
            prefixes.insert(syllable)
        }
        self.prefixes = prefixes
        longest = syllables.map(\.count).max() ?? 0
    }

    /// Apostrophes separate syllables, as Rime reads them.
    public func continues(_ input: String) -> Bool {
        let parts = input.lowercased().split(separator: "'", omittingEmptySubsequences: true).map { Array($0) }
        guard let last = parts.last else { return true }
        var memo: [String: Bool] = [:]
        return parts.dropLast().allSatisfy { open(ArraySlice($0), complete: true, &memo) } && open(ArraySlice(last), complete: false, &memo)
    }

    /// Whether letters split into syllables, the last allowed to be the start
    /// of one unless `complete`.
    private func open(_ letters: ArraySlice<Character>, complete: Bool, _ memo: inout [String: Bool]) -> Bool {
        if letters.isEmpty { return true }
        let key = "\(complete)" + String(letters)
        if let known = memo[key] { return known }
        var result = !complete && prefixes.contains(String(letters))
        var length = 1
        while !result, length <= min(longest, letters.count) {
            let head = String(letters.prefix(length))
            if syllables.contains(head), open(letters.dropFirst(length), complete: complete, &memo) { result = true }
            length += 1
        }
        memo[key] = result
        return result
    }
}
