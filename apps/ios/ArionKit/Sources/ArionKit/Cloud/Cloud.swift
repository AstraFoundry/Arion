import Foundation

/// Where polishing and translation run.
public enum PolishEngine: String, Sendable, CaseIterable {
    case device
    case cloud
}

/// Cloud providers the user connects with their own API key, as on the
/// desktop (docs/project/cloud-polishing.md). The ChatGPT subscription
/// sign-in is desktop-only: its callback is a fixed loopback port.
public enum CloudProvider: String, Sendable, CaseIterable, Identifiable {
    case openai
    case anthropic
    case gemini
    case deepseek
    case openrouter
    case openaiCompatible = "openai-compatible"
    case anthropicCompatible = "anthropic-compatible"

    public var id: String { rawValue }

    /// Compatible endpoints take a base URL with the key.
    public var needsBaseURL: Bool { self == .openaiCompatible || self == .anthropicCompatible }

    /// The provider's name; brand names are not translated.
    public var name: String {
        switch self {
        case .openai: "OpenAI"
        case .anthropic: "Anthropic"
        case .gemini: "Google Gemini"
        case .deepseek: "DeepSeek"
        case .openrouter: "OpenRouter"
        case .openaiCompatible: L10n.cloudOpenaiCompatible
        case .anthropicCompatible: L10n.cloudAnthropicCompatible
        }
    }
}

/// Content-free connection failures, as `cloudpolish.Code` on the desktop.
public enum CloudFailure: String, Error, Sendable, Equatable {
    /// The provider refused the key (401 or an authentication error).
    case rejected
    case network
    /// The model list could not be read.
    case models
    /// The keychain could not read or save the credential.
    case store
    /// A compatible base URL is not an http or https address with a host.
    case address
}

public struct CloudConnection: Sendable, Equatable {
    public var provider: CloudProvider
    public var model: String?
    public var baseURL: URL?

    public init(provider: CloudProvider, model: String?, baseURL: URL?) {
        self.provider = provider
        self.model = model
        self.baseURL = baseURL
    }
}

/// The connected providers. Credentials live in the keychain group the app
/// and the keyboard share; the keyboard only reads them.
@MainActor
public protocol CloudServing: AnyObject {
    /// The provider in use, if any.
    var active: CloudConnection? { get }
    /// Providers with a stored key, the active one included.
    var saved: Set<CloudProvider> { get }
    /// Lists the provider's models with the key; stores it and makes the
    /// provider active only when that works. Returns the models.
    func connect(_ provider: CloudProvider, key: String, baseURL: String?) async throws(CloudFailure) -> [String]
    /// Makes a provider with a stored key active.
    func activate(_ provider: CloudProvider) throws(CloudFailure)
    /// Deletes the active provider's key.
    func disconnect() throws(CloudFailure)
    /// The active provider's models.
    func models() async throws(CloudFailure) -> [String]
    /// Chooses the active provider's model.
    func choose(model: String)
}
