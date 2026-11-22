import Foundation
import Security

/// Generic-password items, service "Arion". Items carry no explicit access
/// group, so they land in the first `keychain-access-groups` entry, which the
/// app and the keyboard share.
enum Keychain {
    static let service = "Arion"

    static func read(_ account: String) throws(CloudFailure) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
        ]
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess else { throw .store }
        return result as? Data
    }

    static func write(_ data: Data, _ account: String) throws(CloudFailure) {
        let match: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        let update: [String: Any] = [kSecValueData as String: data]
        var status = SecItemUpdate(match as CFDictionary, update as CFDictionary)
        if status == errSecItemNotFound {
            var add = match
            add[kSecValueData as String] = data
            // The keyboard reads keys while the phone is unlocked after first unlock.
            add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
            status = SecItemAdd(add as CFDictionary, nil)
        }
        guard status == errSecSuccess else { throw .store }
    }

    static func delete(_ account: String) throws(CloudFailure) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else { throw .store }
    }
}

/// One provider's stored credential, as the desktop's `cloud-polish/<provider>` item.
struct CloudCredential: Codable, Equatable {
    var provider: String
    var key: String
    var model: String?
    var baseUrl: String?
}

/// Endpoints and request shapes per provider, as the desktop's Republic adapters.
enum CloudAPI {
    static func base(_ provider: CloudProvider, _ custom: URL?) -> URL? {
        switch provider {
        case .openai: URL(string: "https://api.openai.com/v1")
        case .anthropic: URL(string: "https://api.anthropic.com/v1")
        case .gemini: URL(string: "https://generativelanguage.googleapis.com/v1beta")
        case .deepseek: URL(string: "https://api.deepseek.com/v1")
        case .openrouter: URL(string: "https://openrouter.ai/api/v1")
        case .openaiCompatible, .anthropicCompatible: custom
        }
    }

    static func isAnthropic(_ provider: CloudProvider) -> Bool { provider == .anthropic || provider == .anthropicCompatible }

    static func authorize(_ request: inout URLRequest, _ provider: CloudProvider, key: String) {
        switch provider {
        case .anthropic, .anthropicCompatible:
            request.setValue(key, forHTTPHeaderField: "x-api-key")
            request.setValue("2023-06-01", forHTTPHeaderField: "anthropic-version")
        case .gemini:
            request.setValue(key, forHTTPHeaderField: "x-goog-api-key")
        default:
            request.setValue("Bearer \(key)", forHTTPHeaderField: "Authorization")
        }
    }

    /// Validates a compatible endpoint: http or https with a host, no query,
    /// fragment or user info, stored without a trailing slash.
    static func address(_ text: String?) throws(CloudFailure) -> URL {
        let trimmed = (text ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard let parts = URLComponents(string: trimmed), let scheme = parts.scheme?.lowercased(), scheme == "http" || scheme == "https",
              let host = parts.host, !host.isEmpty, parts.query == nil, parts.fragment == nil, parts.user == nil, parts.password == nil else {
            throw .address
        }
        var clean = trimmed
        while clean.hasSuffix("/") { clean.removeLast() }
        guard let url = URL(string: clean) else { throw .address }
        return url
    }

    static func modelsRequest(_ provider: CloudProvider, base: URL, key: String) -> URLRequest {
        var request = URLRequest(url: base.appending(path: "models"), timeoutInterval: 20)
        authorize(&request, provider, key: key)
        return request
    }

    /// Model IDs from a list response: `data[].id`, or Gemini's `models[].name`.
    static func parseModels(_ provider: CloudProvider, _ data: Data) throws(CloudFailure) -> [String] {
        guard let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { throw .models }
        if provider == .gemini {
            guard let models = object["models"] as? [[String: Any]] else { throw .models }
            return models.compactMap { model -> String? in
                let methods = model["supportedGenerationMethods"] as? [String] ?? ["generateContent"]
                guard methods.contains("generateContent"), let name = model["name"] as? String else { return nil }
                return name.hasPrefix("models/") ? String(name.dropFirst(7)) : name
            }
        }
        guard let list = object["data"] as? [[String: Any]] else { throw .models }
        return list.compactMap { $0["id"] as? String }
    }

    /// The completion request: instructions as the system message, the prompt
    /// as the user message, greedy, thinking off.
    static func completionRequest(_ provider: CloudProvider, base: URL, key: String, model: String, request: RewriteRequest) throws -> URLRequest {
        var body: [String: Any]
        var url: URL
        switch provider {
        case .anthropic, .anthropicCompatible:
            url = base.appending(path: "messages")
            body = [
                "model": model, "max_tokens": request.maximumTokens, "temperature": 0,
                "system": request.instructions,
                "messages": [["role": "user", "content": request.prompt]],
                "thinking": ["type": "disabled"],
            ]
        case .gemini:
            url = base.appending(path: "models/\(model):generateContent")
            body = [
                "systemInstruction": ["parts": [["text": request.instructions]]],
                "contents": [["role": "user", "parts": [["text": request.prompt]]]],
                "generationConfig": ["temperature": 0, "maxOutputTokens": request.maximumTokens, "thinkingConfig": ["thinkingBudget": 0]],
            ]
        default:
            url = base.appending(path: "chat/completions")
            body = [
                "model": model, "temperature": 0,
                "messages": [["role": "system", "content": request.instructions], ["role": "user", "content": request.prompt]],
            ]
            switch provider {
            case .openai:
                body["max_completion_tokens"] = request.maximumTokens
                body["reasoning_effort"] = "none"
            case .deepseek:
                body["max_tokens"] = request.maximumTokens
                body["thinking"] = ["type": "disabled"]
            case .openrouter:
                body["max_tokens"] = request.maximumTokens
                body["reasoning"] = ["effort": "none"]
            default:
                body["max_tokens"] = request.maximumTokens
                body["reasoning_effort"] = "none"
            }
        }
        var urlRequest = URLRequest(url: url, timeoutInterval: 15)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        authorize(&urlRequest, provider, key: key)
        urlRequest.httpBody = try JSONSerialization.data(withJSONObject: body)
        return urlRequest
    }

    /// The completion text; nil when the response is not a complete answer.
    static func parseCompletion(_ provider: CloudProvider, _ data: Data) -> String? {
        guard let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        switch provider {
        case .anthropic, .anthropicCompatible:
            guard (object["stop_reason"] as? String) != "max_tokens", let content = object["content"] as? [[String: Any]] else { return nil }
            return content.filter { ($0["type"] as? String) == "text" }.compactMap { $0["text"] as? String }.joined()
        case .gemini:
            guard let candidate = (object["candidates"] as? [[String: Any]])?.first,
                  (candidate["finishReason"] as? String).map({ $0 == "STOP" }) ?? true,
                  let parts = (candidate["content"] as? [String: Any])?["parts"] as? [[String: Any]] else { return nil }
            return parts.filter { ($0["thought"] as? Bool) != true }.compactMap { $0["text"] as? String }.joined()
        default:
            guard let choice = (object["choices"] as? [[String: Any]])?.first,
                  (choice["finish_reason"] as? String) != "length",
                  let message = choice["message"] as? [String: Any] else { return nil }
            return message["content"] as? String
        }
    }

    // MARK: Default model, as `cloudpolish.DefaultModel`

    private static let tiers: [CloudProvider: [[String]]] = [
        .openai: [["luna"], ["mini"]],
        .openaiCompatible: [["luna"], ["mini"]],
        .anthropic: [["haiku"]],
        .anthropicCompatible: [["haiku"]],
        .gemini: [["flash-lite"], ["flash"]],
        .deepseek: [["flash"], ["deepseek-chat"]],
        .openrouter: [["openai/", "luna"], ["anthropic/", "haiku"], ["deepseek/", "flash"], ["google/", "flash-lite"]],
    ]
    private static let unsuited = ["preview", "exp", "realtime", "audio", "tts", "image", "transcribe", "embed", "search", "vision"]

    /// The newest model of the provider's first tier that has one; nil leaves the choice to the user.
    public static func defaultModel(_ provider: CloudProvider, _ models: [String]) -> String? {
        for tier in tiers[provider] ?? [] {
            var best: String?
            for id in models where matches(id.lowercased(), tier) {
                if best == nil || newer(id, best!) { best = id }
            }
            if let best { return best }
        }
        return nil
    }

    private static func matches(_ id: String, _ tier: [String]) -> Bool {
        !unsuited.contains { id.contains($0) } && tier.allSatisfy { id.contains($0) }
    }

    private static func newer(_ a: String, _ b: String) -> Bool {
        let va = version(a), vb = version(b)
        for (x, y) in zip(va, vb) where x != y { return x > y }
        if va.count != vb.count { return va.count > vb.count }
        return a.count < b.count
    }

    private static func version(_ id: String) -> [Int] {
        let undated = id.replacingOccurrences(of: #"\d{4}-\d{2}-\d{2}|\d{4,}"#, with: "", options: .regularExpression)
        return undated.split(whereSeparator: { !$0.isNumber }).compactMap { Int($0) }
    }
}

/// The connected providers, with one keychain item per provider and an
/// active-provider pointer, as the desktop's cloud polishing service.
@MainActor
public final class CloudService: CloudServing {
    private static let pointer = "cloud-polish"
    private static func account(_ provider: CloudProvider) -> String { "cloud-polish/\(provider.rawValue)" }

    private let session: URLSession
    public private(set) var active: CloudConnection?
    public private(set) var saved: Set<CloudProvider> = []

    public init(session: URLSession = .shared) {
        self.session = session
        reload()
    }

    /// Reads the items again; the keyboard calls this before each dictation.
    public func reload() {
        saved = Set(CloudProvider.allCases.filter { (try? Keychain.read(Self.account($0))) != nil })
        active = nil
        if let data = try? Keychain.read(Self.pointer),
           let pointer = try? JSONDecoder().decode([String: String].self, from: data),
           let provider = pointer["provider"].flatMap(CloudProvider.init(rawValue:)),
           let credential = try? credential(provider) {
            active = CloudConnection(provider: provider, model: credential.model, baseURL: credential.baseUrl.flatMap(URL.init(string:)))
        }
    }

    func credential(_ provider: CloudProvider) throws(CloudFailure) -> CloudCredential? {
        guard let data = try Keychain.read(Self.account(provider)) else { return nil }
        guard let credential = try? JSONDecoder().decode(CloudCredential.self, from: data) else { throw .store }
        return credential
    }

    public func connect(_ provider: CloudProvider, key: String, baseURL: String?) async throws(CloudFailure) -> [String] {
        let custom: URL? = provider.needsBaseURL ? try CloudAPI.address(baseURL) : nil
        guard let base = CloudAPI.base(provider, custom) else { throw .address }
        let models = try await list(provider, base: base, key: key)
        var credential = CloudCredential(provider: provider.rawValue, key: key, model: CloudAPI.defaultModel(provider, models), baseUrl: custom?.absoluteString)
        if let previous = try? self.credential(provider), let model = previous.model, models.contains(model) { credential.model = model }
        try store(credential, provider)
        try setPointer(provider)
        reload()
        return models
    }

    public func activate(_ provider: CloudProvider) throws(CloudFailure) {
        guard try credential(provider) != nil else { throw .store }
        try setPointer(provider)
        reload()
    }

    public func disconnect() throws(CloudFailure) {
        guard let provider = active?.provider else { return }
        try Keychain.delete(Self.account(provider))
        try Keychain.delete(Self.pointer)
        reload()
    }

    public func models() async throws(CloudFailure) -> [String] {
        guard let provider = active?.provider, let credential = try credential(provider),
              let base = CloudAPI.base(provider, credential.baseUrl.flatMap(URL.init(string:))) else { throw .store }
        return try await list(provider, base: base, key: credential.key)
    }

    public func choose(model: String) {
        guard let provider = active?.provider, var credential = try? credential(provider) else { return }
        credential.model = model
        try? store(credential, provider)
        reload()
    }

    private func list(_ provider: CloudProvider, base: URL, key: String) async throws(CloudFailure) -> [String] {
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: CloudAPI.modelsRequest(provider, base: base, key: key))
        } catch {
            throw .network
        }
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0
        if status == 401 { throw .rejected }
        guard status == 200 else { throw .models }
        return try CloudAPI.parseModels(provider, data)
    }

    private func store(_ credential: CloudCredential, _ provider: CloudProvider) throws(CloudFailure) {
        guard let data = try? JSONEncoder().encode(credential) else { throw .store }
        try Keychain.write(data, Self.account(provider))
    }

    private func setPointer(_ provider: CloudProvider) throws(CloudFailure) {
        guard let data = try? JSONEncoder().encode(["provider": provider.rawValue]) else { throw .store }
        try Keychain.write(data, Self.pointer)
    }
}

/// Runs the shared recipe on the provider and model in use.
public struct CloudModel: Sendable {
    public enum Failure: Error, Equatable { case unavailable, timedOut, rejected, failed }

    private let session: URLSession

    public init(session: URLSession = .shared) { self.session = session }

    /// Whether a provider with a chosen model is connected.
    @MainActor public static var ready: Bool {
        let cloud = CloudService()
        return cloud.active?.model != nil
    }

    public func run(_ request: RewriteRequest) async throws -> String {
        let (provider, credential) = try await MainActor.run { () throws -> (CloudProvider, CloudCredential) in
            let cloud = CloudService()
            guard let provider = cloud.active?.provider, let credential = try cloud.credential(provider), credential.model != nil else { throw Failure.unavailable }
            return (provider, credential)
        }
        guard let model = credential.model, let base = CloudAPI.base(provider, credential.baseUrl.flatMap(URL.init(string:))) else { throw Failure.unavailable }
        let urlRequest = try CloudAPI.completionRequest(provider, base: base, key: credential.key, model: model, request: request)
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: urlRequest)
        } catch let error as URLError where error.code == .timedOut {
            throw Failure.timedOut
        } catch {
            throw Failure.failed
        }
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0
        guard status == 200 else { throw status == 401 || status == 403 ? Failure.rejected : Failure.failed }
        guard let text = CloudAPI.parseCompletion(provider, data), RewriteValidation.accepts(text) else { throw Failure.rejected }
        return text.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
