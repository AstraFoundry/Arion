import Foundation

/// The App Group both the app and the keyboard read. The keyboard can write
/// to it only with Full Access, so its writes also prove Full Access.
public enum AppGroup {
    public static let identifier = "group.si.astrafoundry.arion"

    public static var defaults: UserDefaults {
        guard let defaults = UserDefaults(suiteName: identifier) else {
            preconditionFailure("App Group \(identifier) is missing from the entitlements")
        }
        return defaults
    }

    public static var container: URL? {
        FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: identifier)
    }
}

/// The user's dictation preferences, shared by the app and the keyboard.
public final class Preferences: @unchecked Sendable {
    private let defaults: UserDefaults

    public init(defaults: UserDefaults = AppGroup.defaults) {
        self.defaults = defaults
    }

    private enum Key {
        static let polishing = "polishing.enabled"
        static let languages = "dictation.languages"
        static let language = "dictation.language"
        static let history = "history.enabled"
        static let target = "translation.target"
        static let keyboardSeen = "keyboard.lastSeen"
        static let onboarded = "onboarding.completed"
    }

    /// Polishing with the on-device model is on by default; it runs only when
    /// Apple Intelligence is available and never leaves the device.
    public var polishingEnabled: Bool {
        get { defaults.object(forKey: Key.polishing) as? Bool ?? true }
        set { defaults.set(newValue, forKey: Key.polishing) }
    }

    /// Recognition languages the keyboard cycles through, as BCP 47 identifiers.
    public var languages: [String] {
        get {
            let stored = defaults.stringArray(forKey: Key.languages) ?? []
            return stored.isEmpty ? DictationLanguage.defaults() : stored
        }
        set { defaults.set(newValue, forKey: Key.languages) }
    }

    /// The language the next dictation uses.
    public var language: String {
        get {
            let current = defaults.string(forKey: Key.language)
            let all = languages
            return current.flatMap { all.contains($0) ? $0 : nil } ?? all[0]
        }
        set { defaults.set(newValue, forKey: Key.language) }
    }

    /// The language the translate key produces. By default the other of the
    /// first two dictation languages: Chinese speakers translate into English,
    /// English speakers into Simplified Chinese.
    public var translationTarget: TranslationTarget {
        get {
            if let raw = defaults.string(forKey: Key.target), let target = TranslationTarget(rawValue: raw) { return target }
            return languages.first == DictationLanguage.english ? .zhCN : .en
        }
        set { defaults.set(newValue.rawValue, forKey: Key.target) }
    }

    /// Keeping recent dictations is opt-in and stays on this device.
    public var historyEnabled: Bool {
        get { defaults.bool(forKey: Key.history) }
        set { defaults.set(newValue, forKey: Key.history) }
    }

    public var onboardingCompleted: Bool {
        get { defaults.bool(forKey: Key.onboarded) }
        set { defaults.set(newValue, forKey: Key.onboarded) }
    }

    /// When the keyboard last ran with Full Access.
    public var keyboardLastSeen: Date? {
        get { defaults.object(forKey: Key.keyboardSeen) as? Date }
        set { defaults.set(newValue, forKey: Key.keyboardSeen) }
    }
}

/// The recognition languages Arion offers and how they are named.
public enum DictationLanguage {
    public static let english = "en-US"
    public static let chinese = "zh-CN"
    public static let all = [english, chinese]

    /// The device's preferred languages among those Arion offers, English last
    /// when it is not preferred.
    public static func defaults(preferred: [String] = Locale.preferredLanguages) -> [String] {
        var result: [String] = []
        for identifier in preferred {
            let language = Locale(identifier: identifier).language.languageCode?.identifier
            let match = all.first { Locale(identifier: $0).language.languageCode?.identifier == language }
            if let match, !result.contains(match) { result.append(match) }
        }
        return result.isEmpty ? [english] : result
    }

    /// The short mark the keyboard shows for a language.
    public static func mark(_ identifier: String) -> String {
        identifier == chinese ? L10n.languageMarkChinese : L10n.languageMarkEnglish
    }

    public static func name(_ identifier: String) -> String {
        identifier == chinese ? L10n.languageNameChinese : L10n.languageNameEnglish
    }
}
