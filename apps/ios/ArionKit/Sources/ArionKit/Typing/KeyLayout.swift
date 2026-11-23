import Foundation

/// What a typing key does. Letter keys carry the key Rime receives.
public enum TypingKey: Equatable, Sendable {
    /// A letter or digit for the engine (`a`…`z`, the 9-key digits `2`…`9`).
    case letter(Character)
    /// Text inserted as is: punctuation and symbols.
    case text(String)
    /// The 9-key syllable separator, sent to the engine as an apostrophe.
    case separator
    case shift
    case delete
    case space
    case `return`
    /// Shows the numbers and symbols page.
    case symbols
    /// Back from the symbols page to the letters.
    case letters
    /// Toggles Chinese (the Pinyin layout chosen in settings) and English.
    case layout
    /// Back to the microphone; always the bottom-left key, as on the voice panel.
    case voice
    /// The system's next-keyboard key, when the device needs one.
    case globe
}

/// Sizes both panels share, so switching between voice and typing keeps the
/// switch key and the bottom row in place.
public enum KeyboardMetrics {
    public static let height: CGFloat = 266
    public static let switchKeyWidth: CGFloat = 48
    public static let rowHeight: CGFloat = 44
    public static let sideInset: CGFloat = 6
    public static let bottomInset: CGFloat = 6
}

/// The rows of each page. Widths are relative; a row fills the keyboard.
public enum KeyLayout {
    public struct Key: Equatable, Sendable {
        public var key: TypingKey
        public var label: String?
        /// Relative to the other flexible keys of the row.
        public var width: CGFloat
        /// A width in points that does not stretch, such as the switch key.
        public var fixed: CGFloat?
        public init(_ key: TypingKey, label: String? = nil, width: CGFloat = 1, fixed: CGFloat? = nil) {
            self.key = key
            self.label = label
            self.width = width
            self.fixed = fixed
        }
    }

    private static func letters(_ row: String) -> [Key] {
        row.map { Key(.letter($0), label: String($0)) }
    }

    /// QWERTY for Pinyin and English, with Shift on the left as on the
    /// system keyboards. In Pinyin a shifted letter is inserted as is.
    public static func qwerty(_ layout: TypingLayout) -> [[Key]] {
        let third = Key(.shift, width: 1.4)
        return [
            letters("qwertyuiop"),
            letters("asdfghjkl"),
            [third] + letters("zxcvbnm") + [Key(.delete, width: 1.4)],
        ]
    }

    /// The bottom row of every page: the switch key first, at the same place
    /// as on the voice panel, then the page key, 中/EN, space and return.
    public static func bottom(symbols: Bool, globe: Bool, keypad: Bool) -> [Key] {
        var row = [Key(.voice, fixed: KeyboardMetrics.switchKeyWidth)]
        if globe { row.append(Key(.globe, fixed: KeyboardMetrics.switchKeyWidth)) }
        row += [Key(symbols ? .letters : .symbols, width: 1.3), Key(.layout, width: 1.3)]
        // 9-key's return is the tall key of its right column.
        row += keypad ? [Key(.space, width: 6)] : [Key(.space, width: 4), Key(.return, width: 2)]
        return row
    }

    /// The phone keypad: digit groups in a 3×3 grid around a side column.
    public static let keypad: [[Key]] = [
        [Key(.separator), Key(.letter("2"), label: "ABC"), Key(.letter("3"), label: "DEF")],
        [Key(.letter("4"), label: "GHI"), Key(.letter("5"), label: "JKL"), Key(.letter("6"), label: "MNO")],
        [Key(.letter("7"), label: "PQRS"), Key(.letter("8"), label: "TUV"), Key(.letter("9"), label: "WXYZ")],
    ]

    /// 9-key's left column while nothing is composing.
    public static let keypadPunctuation = ["，", "。", "？", "！", "、", "…", "：", "～"]

    /// Digits and the punctuation of the active script.
    public static func symbols(chinese: Bool) -> [[Key]] {
        let digits = "1234567890".map { Key(.text(String($0)), label: String($0)) }
        let middle = (chinese ? ["-", "/", "：", "；", "（", "）", "¥", "@", "“", "”"] : ["-", "/", ":", ";", "(", ")", "$", "@", "\"", "#"])
            .map { Key(.text($0), label: $0) }
        let third = (chinese ? ["。", "，", "、", "？", "！", "…"] : [".", ",", "?", "!", "'", "&"])
            .map { Key(.text($0), label: $0) }
        return [digits, middle, third + [Key(.delete, width: 1.4)]]
    }
}
