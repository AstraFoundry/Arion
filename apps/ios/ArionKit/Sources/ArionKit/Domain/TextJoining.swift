import Foundation

/// How a dictation joins the text before the caret.
public enum TextJoining {
    /// Adds a space between Latin words, as the system keyboard would.
    public static func joining(_ text: String, after context: String?) -> String {
        guard let last = context?.last, let first = text.first else { return text }
        let wordLike = { (c: Character) in (c.isLetter && c.unicodeScalars.allSatisfy { $0.value < 0x2E80 }) || c.isNumber }
        if (wordLike(last) || ".,!?;:".contains(last)) && wordLike(first) { return " " + text }
        return text
    }
}
