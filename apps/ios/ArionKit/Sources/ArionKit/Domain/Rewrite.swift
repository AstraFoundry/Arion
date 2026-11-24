import Foundation

/// The conservative rules applied to every transcript before delivery. They
/// mirror the spacing rules of the desktop's `domain.Correct`
/// (`arion-rules-v4`): collapse whitespace, then drop recognizer spaces
/// between CJK runes and before closing punctuation. They never add words.
public enum CorrectionRules {
    public static func apply(_ raw: String) -> String {
        joinCJKSpacing(collapseWhitespace(raw))
    }

    static func collapseWhitespace(_ s: String) -> String {
        s.split(whereSeparator: \.isWhitespace).joined(separator: " ")
    }

    static func isCJKPunctuation(_ c: Unicode.Scalar) -> Bool {
        (0x3000...0x303F).contains(c.value) || (0xFF00...0xFFEF).contains(c.value)
    }

    static func isCJK(_ c: Unicode.Scalar) -> Bool {
        let v = c.value
        let han = (0x3400...0x9FFF).contains(v) || (0xF900...0xFAFF).contains(v) || (0x20000...0x3134F).contains(v) || (0x2E80...0x2FDF).contains(v)
        let kana = (0x3040...0x30FF).contains(v) || (0x31F0...0x31FF).contains(v)
        let hangul = (0xAC00...0xD7AF).contains(v) || (0x1100...0x11FF).contains(v) || (0x3130...0x318F).contains(v)
        return han || kana || hangul || isCJKPunctuation(c)
    }

    static func isClosingPunctuation(_ c: Unicode.Scalar) -> Bool {
        ",.!?;:%)]}".unicodeScalars.contains(c) || (isCJKPunctuation(c) && !"「『（【《〈".unicodeScalars.contains(c))
    }

    static func joinCJKSpacing(_ s: String) -> String {
        let scalars = Array(s.unicodeScalars)
        var out = String.UnicodeScalarView()
        for (i, c) in scalars.enumerated() {
            if c == " ", i > 0, i + 1 < scalars.count {
                let prev = scalars[i - 1], next = scalars[i + 1]
                if (isCJK(prev) && isCJK(next)) || isClosingPunctuation(next) || isCJKPunctuation(prev) { continue }
            }
            out.append(c)
        }
        return String(out)
    }
}

/// Where the text goes, which guides layout but never content. The keyboard
/// derives it from the field's traits; the desktop's scene names apply.
public enum Scene: String, Sendable, CaseIterable {
    case chat, work, email, prompt, list, search
}

/// A translation target the shared recipe supports.
public enum TranslationTarget: String, Sendable, CaseIterable {
    case en, zhCN = "zh-CN", zhTW = "zh-TW", ja, ko, es, fr, de, pt, it, ru, ar

    public var name: String {
        switch self {
        case .en: L10n.targetEn
        case .zhCN: L10n.targetZhCN
        case .zhTW: L10n.targetZhTW
        case .ja: L10n.targetJa
        case .ko: L10n.targetKo
        case .es: L10n.targetEs
        case .fr: L10n.targetFr
        case .de: L10n.targetDe
        case .pt: L10n.targetPt
        case .it: L10n.targetIt
        case .ru: L10n.targetRu
        case .ar: L10n.targetAr
        }
    }
}

/// One on-device rewrite: trusted instructions and the untrusted dictation.
public struct RewriteRequest: Equatable, Sendable {
    public let instructions: String
    public let prompt: String
    public let maximumTokens: Int
}

/// Builds polishing and translation requests from the desktop's formatting
/// recipe (`apps/desktop/infrastructure/llm/prompt/format.json`, bundled
/// unchanged), following `SelectedTemplates.Format` and `FormatTranslation`.
public struct RewriteRecipe: Sendable {
    private struct Example: Decodable, Sendable { let source: String; let output: String }
    private struct SceneRecipe: Decodable, Sendable { let instruction: String; let instructionHan: String }
    private struct Translation: Decodable, Sendable {
        let system: String
        let rules: String
        let route: String
        let editingExamples: [Example]
        let scenes: [String: [Example]]
    }
    private struct Format: Decodable, Sendable {
        let version: String
        let editingExamples: [Example]
        let formatSystem, formatSystemHan, formatRules, formatRulesHan: String
        let prosePolicy, prosePolicyHan: String
        let routeInstructions: [String: String]
        let contextRules, contextRulesHan, outputRule, outputRuleHan: String
        let format: [String: SceneRecipe]
        let translate: [String: Translation]
    }

    private let recipe: Format
    public var version: String { recipe.version }

    public init(data: Data) throws {
        recipe = try JSONDecoder().decode(Format.self, from: data)
    }

    /// The recipe bundled with ArionKit.
    public static func bundled() throws -> RewriteRecipe {
        guard let url = Bundle.module.url(forResource: "format", withExtension: "json") else {
            throw CocoaError(.fileNoSuchFile)
        }
        return try RewriteRecipe(data: Data(contentsOf: url))
    }

    enum Route: String { case han, latin, mixed }

    static func isHan(_ c: Unicode.Scalar) -> Bool {
        let v = c.value
        return (0x3400...0x4DBF).contains(v) || (0x4E00...0x9FFF).contains(v) || (0xF900...0xFAFF).contains(v) || (0x20000...0x3134F).contains(v)
    }

    static func isLatin(_ c: Unicode.Scalar) -> Bool {
        let v = c.value
        if (0x41...0x5A).contains(v) || (0x61...0x7A).contains(v) { return true }
        if (0xC0...0x24F).contains(v) { return v != 0xD7 && v != 0xF7 }
        return (0x1E00...0x1EFF).contains(v) || (0xFF21...0xFF3A).contains(v) || (0xFF41...0xFF5A).contains(v)
    }

    /// Which script the source uses, as the desktop's `route`.
    static func route(_ source: String) -> Route {
        let han = source.unicodeScalars.contains(where: isHan)
        let latin = source.unicodeScalars.contains(where: isLatin)
        if han && !latin { return .han }
        if latin && !han { return .latin }
        return .mixed
    }

    /// The formatter's scene: search is written as chat.
    private func formatScene(_ scene: Scene) -> String {
        scene == .search ? Scene.chat.rawValue : scene.rawValue
    }

    static func budget(_ source: String, perCharacter: Int) -> Int {
        min(2048, max(256, source.count * perCharacter + 128))
    }

    static func wrap(_ source: String) -> String { "<source>" + source + "</source>" }

    public func polish(_ source: String, scene: Scene) -> RewriteRequest {
        let route = Self.route(source)
        let sceneName = formatScene(scene)
        let sceneRecipe = recipe.format[sceneName] ?? recipe.format["chat"]!
        let han = route == .han
        var sections = [
            han ? recipe.formatSystemHan : recipe.formatSystem,
            han ? recipe.formatRulesHan : recipe.formatRules,
            "# Scene\n\n" + (han ? sceneRecipe.instructionHan : sceneRecipe.instruction),
        ]
        if sceneName != Scene.list.rawValue {
            sections.append(han ? recipe.prosePolicyHan : recipe.prosePolicy)
        }
        sections.append(recipe.routeInstructions[route.rawValue] ?? "")
        sections.append("# Context\n\n" + (han ? recipe.contextRulesHan : recipe.contextRules))
        let examples = recipe.editingExamples
            .filter { route == .mixed || Self.route($0.source) == route }
            .map { han ? "原文：\($0.source)\n输出：\($0.output)" : "Input: \($0.source)\nOutput: \($0.output)" }
        sections.append("# Examples\n\n" + examples.joined(separator: "\n\n"))
        sections.append("# Output\n\n" + (han ? recipe.outputRuleHan : recipe.outputRule))
        return RewriteRequest(instructions: sections.joined(separator: "\n\n"), prompt: Self.wrap(source), maximumTokens: Self.budget(source, perCharacter: 1))
    }

    /// Polishes and translates into the target. The desktop sends the scene's
    /// demonstrations as chat turns; the on-device session takes them inside
    /// its instructions in the same order.
    public func translate(_ source: String, scene: Scene, target: TranslationTarget) -> RewriteRequest? {
        guard let translation = recipe.translate[target.rawValue] else { return nil }
        let sceneName = formatScene(scene)
        guard let sceneRecipe = recipe.format[sceneName], let sceneExamples = translation.scenes[sceneName] else { return nil }
        let chinese = target == .zhCN
        var lines = translation.rules.components(separatedBy: "\n")
        lines.append(chinese ? sceneRecipe.instructionHan : sceneRecipe.instruction)
        if sceneName != Scene.list.rawValue {
            lines.append(chinese ? recipe.prosePolicyHan : recipe.prosePolicy)
        }
        lines.append(translation.route)
        var groups = [sceneExamples, translation.editingExamples]
        if sceneName == Scene.list.rawValue { groups.swapAt(0, 1) }
        var demonstrations: [String] = []
        for (index, group) in groups.enumerated() {
            for example in group {
                let output = sceneName == Scene.list.rawValue && index == 0 ? "1. " + example.output : example.output
                demonstrations.append("Input: \(Self.wrap(example.source))\nOutput: \(output)")
            }
        }
        let instructions = translation.system + "\n\n<rules>\n" + lines.joined(separator: "\n") + "\n</rules>"
            + "\n\n# Context\n\n" + (chinese ? recipe.contextRulesHan : recipe.contextRules)
            + "\n\n# Examples\n\n" + demonstrations.joined(separator: "\n\n")
        return RewriteRequest(instructions: instructions, prompt: Self.wrap(source), maximumTokens: Self.budget(source, perCharacter: 2))
    }
}

/// The delivery check every model proposal passes before insertion, as the
/// desktop's `domain.ValidateRewriteDelivery`: complete, nonempty output
/// without chat or prompt control markup.
public enum RewriteValidation {
    static let markers: [String] = {
        var out = ["<think", "</think", "<source>", "</source>", "<|", "|>", "[terms", "\"transcript\""]
        out += ["<context>", "</context>", "<screen>", "</screen>", "<before_cursor>", "</before_cursor>", "<after_cursor>", "</after_cursor>",
                "<history>", "</history>", "<dictation>", "</dictation>"]
        out += Scene.allCases.map { "[\($0.rawValue)]" }
        return out
    }()

    public static func accepts(_ candidate: String, truncated: Bool = false) -> Bool {
        guard !truncated else { return false }
        let trimmed = candidate.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        let lower = trimmed.lowercased()
        return !markers.contains { lower.contains($0) }
    }
}
