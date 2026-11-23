import UIKit

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
    /// Shows the numbers page (123), or returns to it from #+=.
    case symbols
    /// Shows the second symbols page (#+=).
    case moreSymbols
    /// Back from the symbols pages to the letters.
    case letters
    /// Toggles Chinese (the Pinyin layout chosen in settings) and English.
    case layout
    /// The system's next-keyboard key, when the device needs one.
    case globe
}

/// Sizes both panels share, taken from the iOS 26 system keyboard on a
/// 402-point iPhone so keys sit where users expect them: 6-point gaps and
/// margins, letter keys a tenth of the row less the gaps, 42-point rows 12
/// points apart, Shift and Delete 1.35 letters, the bottom row's small keys
/// 1.3 letters and Return 2.73. Switching panels keeps every shared key put.
public enum KeyboardMetrics {
    public static let height: CGFloat = 266
    public static let rowHeight: CGFloat = 42
    public static let rowGap: CGFloat = 12
    public static let gap: CGFloat = 6
    public static let sideInset: CGFloat = 6
    public static let bottomInset: CGFloat = 8

    /// A key `letters` letter-keys wide in a row of the given width.
    @MainActor
    public static func width(_ view: UIView, letters: CGFloat, of row: UIView) -> NSLayoutConstraint {
        view.widthAnchor.constraint(equalTo: row.widthAnchor, multiplier: 0.1 * letters, constant: -0.9 * gap * letters)
    }
    public static let functionKey: CGFloat = 1.3
    public static let returnKey: CGFloat = 2.73
    public static let shiftKey: CGFloat = 1.35
}

/// The rows of each page. Widths are relative; a row fills the keyboard.
public enum KeyLayout {
    public struct Key: Equatable, Sendable {
        public var key: TypingKey
        public var label: String?
        /// In letter keys (see `KeyboardMetrics`); 0 fills what the row leaves.
        public var width: CGFloat
        public init(_ key: TypingKey, label: String? = nil, width: CGFloat = 1) {
            self.key = key
            self.label = label
            self.width = width
        }
    }

    private static func letters(_ row: String) -> [Key] {
        row.map { Key(.letter($0), label: String($0)) }
    }

    /// QWERTY for Pinyin and English, with Shift on the left as on the
    /// system keyboards. In Pinyin a shifted letter is inserted as is.
    public static func qwerty(_ layout: TypingLayout) -> [[Key]] {
        [
            letters("qwertyuiop"),
            letters("asdfghjkl"),
            [Key(.shift, width: KeyboardMetrics.shiftKey)] + letters("zxcvbnm") + [Key(.delete, width: KeyboardMetrics.shiftKey)],
        ]
    }

    /// The bottom row of every page: the page key, 中/EN, space and return.
    public static func bottom(symbols: Bool, globe: Bool, keypad: Bool) -> [Key] {
        let small = KeyboardMetrics.functionKey
        // As on the system keyboard: 123 first; the voice switch is top right.
        var row: [Key] = globe ? [Key(.globe, width: small)] : []
        row += [Key(symbols ? .letters : .symbols, width: small), Key(.layout, width: small), Key(.space, width: 0)]
        // 9-key's return is the tall key of its right column.
        if !keypad { row.append(Key(.return, width: KeyboardMetrics.returnKey)) }
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

    /// The symbols pages, key for key as the iOS 26 system keyboard draws
    /// them for English and for Simplified Chinese. The third row keeps the
    /// page key and Delete at the edges; its punctuation shares the rest.
    public static func symbols(chinese: Bool, more: Bool) -> [[Key]] {
        let rows: [[String]]
        switch (chinese, more) {
        case (false, false): rows = [Array("1234567890").map(String.init), ["-", "/", ":", ";", "(", ")", "$", "&", "@", "\""], [".", ",", "?", "!", "'"]]
        case (false, true): rows = [["[", "]", "{", "}", "#", "%", "^", "*", "+", "="], ["_", "\\", "|", "~", "<", ">", "€", "£", "¥", "•"], [".", ",", "?", "!", "'"]]
        case (true, false): rows = [Array("1234567890").map(String.init), ["-", "/", "：", "；", "（", "）", "¥", "@", "“", "”"], ["。", "，", "、", "？", "！", "."]]
        case (true, true): rows = [["【", "】", "｛", "｝", "#", "%", "^", "*", "+", "="], ["_", "—", "\\", "|", "～", "《", "》", "$", "&", "·"], ["…", "，", "^_^", "？", "！", "'"]]
        }
        let text = { (row: [String]) in row.map { Key(.text($0), label: $0) } }
        let page = Key(more ? .symbols : .moreSymbols, width: KeyboardMetrics.shiftKey)
        return [text(rows[0]), text(rows[1]),
                [page] + rows[2].map { Key(.text($0), label: $0, width: 0) } + [Key(.delete, width: KeyboardMetrics.shiftKey)]]
    }

    /// Long-press alternates, as the system keyboard offers them.
    public static let alternates: [String: [String]] = [
        "a": ["à", "á", "â", "ä", "æ", "ã", "å", "ā"], "c": ["ç", "ć", "č"], "e": ["è", "é", "ê", "ë", "ē", "ė", "ę"],
        "i": ["î", "ï", "í", "ī", "į", "ì"], "l": ["ł"], "n": ["ñ", "ń"], "o": ["ô", "ö", "ò", "ó", "œ", "ø", "ō", "õ"],
        "s": ["ß", "ś", "š"], "u": ["û", "ü", "ù", "ú", "ū"], "y": ["ÿ"], "z": ["ž", "ź", "ż"],
        "0": ["°"], "-": ["–", "—", "•"], "/": ["\\"], "$": ["₽", "¥", "€", "¢", "£", "₩"], "&": ["§"],
        "\"": ["„", "“", "”", "«", "»"], ".": ["…"], "?": ["¿"], "!": ["¡"], "'": ["`", "‘", "’"], "%": ["‰"],
        "。": ["…"], "，": ["、"], "？": ["¿"], "！": ["¡"], "“": ["「", "『"], "”": ["」", "』"], "¥": ["$", "€", "£"],
    ]
}
