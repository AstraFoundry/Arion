import AuthenticationServices
import CryptoKit
import Foundation

/// The parts of OpenID Connect the account needs, as the desktop's
/// `internal/account/oidc.go`.
struct Discovery: Decodable {
    let issuer: String
    let authorization_endpoint: URL
    let token_endpoint: URL
    let revocation_endpoint: URL?
    let userinfo_endpoint: URL?
    let code_challenge_methods_supported: [String]?
}

struct TokenResponse: Decodable {
    let access_token: String
    let refresh_token: String?
    let id_token: String?
    let expires_in: Double?
}

enum OIDC {
    static func verifier() -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        return base64URL(Data(bytes))
    }

    static func challenge(_ verifier: String) -> String {
        base64URL(Data(SHA256.hash(data: Data(verifier.utf8))))
    }

    static func base64URL(_ data: Data) -> String {
        data.base64EncodedString().replacingOccurrences(of: "+", with: "-").replacingOccurrences(of: "/", with: "_").replacingOccurrences(of: "=", with: "")
    }

    static func authorizeURL(_ discovery: Discovery, _ config: AccountConfig, state: String, challenge: String) -> URL? {
        guard var parts = URLComponents(url: discovery.authorization_endpoint, resolvingAgainstBaseURL: false) else { return nil }
        let existing = parts.queryItems ?? []
        parts.queryItems = existing + [
            URLQueryItem(name: "client_id", value: config.clientID),
            URLQueryItem(name: "redirect_uri", value: config.redirect.absoluteString),
            URLQueryItem(name: "response_type", value: "code"),
            URLQueryItem(name: "scope", value: config.scopes.joined(separator: " ")),
            URLQueryItem(name: "state", value: state),
            URLQueryItem(name: "code_challenge", value: challenge),
            URLQueryItem(name: "code_challenge_method", value: "S256"),
            URLQueryItem(name: "resource", value: config.resource),
        ]
        return parts.url
    }

    /// The profile claims of an ID token; its signature is not checked on the device.
    static func claims(_ token: String?) -> [String: Any] {
        guard let token, case let parts = token.split(separator: "."), parts.count >= 2 else { return [:] }
        var payload = String(parts[1]).replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        while payload.count % 4 != 0 { payload += "=" }
        guard let data = Data(base64Encoded: payload), let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return [:] }
        return object
    }
}

private struct AccountError: Error {
    let failure: AccountFailure?
}

/// The AstraFoundry account on the phone: sign-in in the system web sheet,
/// a refresh token in the keychain, the platform's usage consent and the
/// consented daily usage report, as the desktop's account service.
@MainActor
public final class AccountService: NSObject, AccountServing, ASWebAuthenticationPresentationContextProviding {
    public private(set) var state = AccountState() {
        didSet { if state != oldValue { onChange?(state) } }
    }
    public var onChange: ((AccountState) -> Void)?
    /// The window the sign-in sheet presents over.
    public var anchor: () -> ASPresentationAnchor = { ASPresentationAnchor() }

    private let config: AccountConfig
    private let http: URLSession
    private let ledger: UsageLedger
    private let defaults = UserDefaults.standard
    private var accessToken: String?
    private var expiry: Date?
    private var webSession: ASWebAuthenticationSession?
    private var generation = 0
    private var reporting = false

    private static let tokenAccount = "account-refresh-token"
    private enum Key {
        static let profile = "account.profile"
        static let source = "usage.sourceId"
        static let sent = "usage.sent"
    }

    public init(config: AccountConfig = AccountConfig(), session: URLSession = .shared, ledger: UsageLedger = UsageLedger()) {
        self.config = config
        http = session
        self.ledger = ledger
        super.init()
        state.profile = storedProfile()
    }

    public nonisolated func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        MainActor.assumeIsolated { anchor() }
    }

    // MARK: Actions

    public func restore() {
        guard state.phase == .signedOut, (try? Keychain.read(Self.tokenAccount)) != nil else {
            if state.phase == .signedOut { state.profile = nil }
            return
        }
        state.phase = .restoring
        let current = generation
        Task {
            do {
                try await refresh()
                guard current == generation else { return }
                state.phase = .signedIn
                state.failure = nil
                await readConsent()
                reportUsage()
            } catch let error as AccountError {
                guard current == generation else { return }
                // Transient failures keep the account and try again on the next launch.
                if error.failure == .sessionEnded {
                    signedOut(failure: .sessionEnded)
                } else {
                    state.phase = .signedIn
                    state.failure = error.failure
                }
            } catch {
                guard current == generation else { return }
                state.phase = .signedIn
                state.failure = .refresh
            }
        }
    }

    public func signIn(shareUsage: Bool) {
        guard !state.busy else {
            state.failure = .inProgress
            return
        }
        generation += 1
        let current = generation
        state.phase = .signingIn
        state.failure = nil
        Task {
            do {
                let discovery = try await discover()
                let verifier = OIDC.verifier()
                let stateToken = OIDC.verifier()
                guard let url = OIDC.authorizeURL(discovery, config, state: stateToken, challenge: OIDC.challenge(verifier)) else { throw AccountError(failure: .discovery) }
                let callback = try await authenticate(url)
                guard current == generation else { return }
                let items = URLComponents(url: callback, resolvingAgainstBaseURL: false)?.queryItems ?? []
                if let error = items.first(where: { $0.name == "error" })?.value {
                    throw AccountError(failure: error == "access_denied" ? .denied : .authorization)
                }
                guard items.first(where: { $0.name == "state" })?.value == stateToken else { throw AccountError(failure: .authorization) }
                guard let code = items.first(where: { $0.name == "code" })?.value else { throw AccountError(failure: .authorization) }
                let tokens = try await token(discovery, form: [
                    "grant_type": "authorization_code", "code": code, "redirect_uri": config.redirect.absoluteString,
                    "client_id": config.clientID, "code_verifier": verifier, "resource": config.resource,
                ])
                guard current == generation else { return }
                try await accept(tokens, discovery)
                state.phase = .signedIn
                if shareUsage {
                    setUsageSharing(true)
                } else {
                    await readConsent()
                }
                reportUsage()
            } catch let error as AccountError {
                guard current == generation else { return }
                signedOut(failure: error.failure)
            } catch {
                guard current == generation else { return }
                signedOut(failure: .generic)
            }
        }
    }

    public func cancel() {
        generation += 1
        webSession?.cancel()
        webSession = nil
        if state.phase == .signingIn { signedOut(failure: nil) }
    }

    public func signOut() {
        generation += 1
        let refresh = (try? Keychain.read(Self.tokenAccount)).flatMap { String(data: $0, encoding: .utf8) }
        state.phase = .signingOut
        Task {
            var confirmed = refresh == nil
            if let refresh, let discovery = try? await discover(), let endpoint = discovery.revocation_endpoint {
                var request = URLRequest(url: endpoint, timeoutInterval: 15)
                request.httpMethod = "POST"
                request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
                request.httpBody = Self.form(["token": refresh, "token_type_hint": "refresh_token", "client_id": config.clientID])
                if let (_, response) = try? await http.data(for: request), (response as? HTTPURLResponse)?.statusCode == 200 { confirmed = true }
            }
            let deleted = (try? Keychain.delete(Self.tokenAccount)) != nil
            defaults.removeObject(forKey: Key.profile)
            defaults.removeObject(forKey: Key.sent)
            signedOut(failure: !deleted ? .store : (confirmed ? nil : .revokeUnconfirmed))
            state.profile = nil
        }
    }

    public func setUsageSharing(_ share: Bool) {
        guard state.signedIn else { return }
        state.usageSharing = .saving
        Task {
            do {
                let answer = try await consent(method: "PUT", accepted: share)
                state.usageSharing = answer ? .shared : .notShared
                state.failure = nil
                if answer { reportUsage() }
            } catch {
                state.failure = .usageConsent
                await readConsent()
            }
        }
    }

    /// Sends the days whose counts changed since they were last confirmed,
    /// within the platform's 31-day window, as the desktop's usage upload.
    public func reportUsage() {
        guard state.signedIn, state.usageSharing == .shared, !reporting else { return }
        reporting = true
        Task {
            defer { reporting = false }
            guard let token = try? await currentToken() else { return }
            var sent = defaults.dictionary(forKey: Key.sent) as? [String: [Int]] ?? [:]
            let today = Date.now
            for day in ledger.days(ending: today, count: 31) {
                let visited = day.sessions > 0 || day.date == UsageLedger.key(today)
                guard visited else { continue }
                let snapshot = [day.sessions, Int(day.speechMs), day.words, 1]
                guard sent[day.date] != snapshot else { continue }
                guard await report(day, token: token) else { break }
                sent[day.date] = snapshot
            }
            let oldest = UsageLedger.key(Calendar.current.date(byAdding: .day, value: -31, to: today) ?? today)
            sent = sent.filter { $0.key >= oldest }
            defaults.set(sent, forKey: Key.sent)
        }
    }

    // MARK: Protocol steps

    private func discover() async throws -> Discovery {
        let url = config.issuer.appending(path: ".well-known/openid-configuration")
        guard let (data, response) = try? await http.data(from: url) else { throw AccountError(failure: .discovery) }
        guard (response as? HTTPURLResponse)?.statusCode == 200, let discovery = try? JSONDecoder().decode(Discovery.self, from: data) else {
            throw AccountError(failure: .discovery)
        }
        var issuer = config.issuer.absoluteString
        while issuer.hasSuffix("/") { issuer.removeLast() }
        guard discovery.issuer.trimmingCharacters(in: CharacterSet(charactersIn: "/")) == issuer.trimmingCharacters(in: CharacterSet(charactersIn: "/")) else {
            throw AccountError(failure: .discovery)
        }
        if let methods = discovery.code_challenge_methods_supported, !methods.contains("S256") { throw AccountError(failure: .discovery) }
        return discovery
    }

    private func authenticate(_ url: URL) async throws -> URL {
        try await withCheckedThrowingContinuation { continuation in
            let session = ASWebAuthenticationSession(url: url, callback: .customScheme(config.redirect.scheme ?? "arion")) { callback, error in
                if let callback {
                    continuation.resume(returning: callback)
                } else if (error as? ASWebAuthenticationSessionError)?.code == .canceledLogin {
                    // Closing the sheet is not a failure.
                    continuation.resume(throwing: AccountError(failure: nil))
                } else {
                    continuation.resume(throwing: AccountError(failure: .authorization))
                }
            }
            session.presentationContextProvider = self
            session.prefersEphemeralWebBrowserSession = false
            webSession = session
            if !session.start() { continuation.resume(throwing: AccountError(failure: .generic)) }
        }
    }

    private func token(_ discovery: Discovery, form: [String: String]) async throws -> TokenResponse {
        var request = URLRequest(url: discovery.token_endpoint, timeoutInterval: 20)
        request.httpMethod = "POST"
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.httpBody = Self.form(form)
        guard let (data, response) = try? await http.data(for: request) else { throw AccountError(failure: .token) }
        let status = (response as? HTTPURLResponse)?.statusCode ?? 0
        if status == 400, let body = try? JSONSerialization.jsonObject(with: data) as? [String: Any], body["error"] as? String == "invalid_grant" {
            throw AccountError(failure: .sessionEnded)
        }
        guard status == 200, let tokens = try? JSONDecoder().decode(TokenResponse.self, from: data) else { throw AccountError(failure: .token) }
        return tokens
    }

    private func accept(_ tokens: TokenResponse, _ discovery: Discovery) async throws {
        accessToken = tokens.access_token
        expiry = tokens.expires_in.map { Date.now.addingTimeInterval($0) }
        if let refresh = tokens.refresh_token {
            do {
                try Keychain.write(Data(refresh.utf8), Self.tokenAccount)
            } catch {
                throw AccountError(failure: .store)
            }
        }
        let claims = OIDC.claims(tokens.id_token)
        var profile = AccountProfile(
            subject: claims["sub"] as? String ?? OIDC.claims(tokens.access_token)["sub"] as? String ?? "",
            name: claims["name"] as? String ?? "", email: claims["email"] as? String ?? "",
            picture: (claims["picture"] as? String).flatMap(URL.init(string:)))
        if let endpoint = discovery.userinfo_endpoint {
            var request = URLRequest(url: endpoint, timeoutInterval: 15)
            request.setValue("Bearer \(tokens.access_token)", forHTTPHeaderField: "Authorization")
            if let (data, response) = try? await http.data(for: request), (response as? HTTPURLResponse)?.statusCode == 200,
               let info = try? JSONSerialization.jsonObject(with: data) as? [String: Any], profile.subject.isEmpty || info["sub"] as? String == profile.subject {
                profile.subject = info["sub"] as? String ?? profile.subject
                profile.name = info["name"] as? String ?? profile.name
                profile.email = info["email"] as? String ?? profile.email
                profile.picture = (info["picture"] as? String).flatMap(URL.init(string:)) ?? profile.picture
            }
        }
        // The same subject keeps a known name and email when the new claims omit them.
        if let previous = state.profile, previous.subject == profile.subject {
            if profile.name.isEmpty { profile.name = previous.name }
            if profile.email.isEmpty { profile.email = previous.email }
        }
        state.profile = profile
        store(profile)
    }

    private func refresh() async throws {
        guard let data = try? Keychain.read(Self.tokenAccount), let refresh = String(data: data, encoding: .utf8) else {
            throw AccountError(failure: .sessionEnded)
        }
        let discovery = try await discover()
        do {
            let tokens = try await token(discovery, form: ["grant_type": "refresh_token", "refresh_token": refresh, "client_id": config.clientID, "resource": config.resource])
            try await accept(tokens, discovery)
        } catch let error as AccountError where error.failure == .sessionEnded {
            try? Keychain.delete(Self.tokenAccount)
            throw error
        } catch let error as AccountError {
            throw AccountError(failure: error.failure == .token ? .refresh : error.failure)
        }
    }

    private func currentToken() async throws -> String {
        if let accessToken, let expiry, expiry.timeIntervalSinceNow > 60 { return accessToken }
        if let accessToken, expiry == nil { return accessToken }
        try await refresh()
        guard let accessToken else { throw AccountError(failure: .refresh) }
        return accessToken
    }

    private func readConsent() async {
        state.usageSharing = .loading
        do {
            state.usageSharing = try await consent(method: "GET", accepted: nil) ? .shared : .notShared
        } catch let error as AccountError where error.failure == .usageConsent && pendingUndecided {
            pendingUndecided = false
            state.usageSharing = .undecided
        } catch {
            state.usageSharing = .unknown
            state.failure = .usageConsent
        }
    }

    private var pendingUndecided = false

    /// Reads or records the platform's answer; true when sharing.
    private func consent(method: String, accepted: Bool?) async throws -> Bool {
        let token = try await currentToken()
        var parts = URLComponents(url: config.issuer.appending(path: "api/v1/usage/consent"), resolvingAgainstBaseURL: false)
        if method == "GET" { parts?.queryItems = [URLQueryItem(name: "productSlug", value: "arion")] }
        guard let url = parts?.url else { throw AccountError(failure: .usageConsent) }
        var request = URLRequest(url: url, timeoutInterval: 15)
        request.httpMethod = method
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let accepted {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONSerialization.data(withJSONObject: ["productSlug": "arion", "accepted": accepted])
        }
        guard let (data, response) = try? await http.data(for: request), (response as? HTTPURLResponse)?.statusCode == 200,
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any], let answer = object["data"] as? [String: Any] else {
            throw AccountError(failure: .usageConsent)
        }
        let isAccepted = answer["accepted"] as? Bool ?? false
        let decided = answer["decided"] as? Bool ?? false
        if isAccepted && (answer["version"] as? String != "usage-v1" || !decided) { throw AccountError(failure: .usageConsent) }
        if let accepted, isAccepted != accepted || !decided { throw AccountError(failure: .usageConsent) }
        if accepted == nil && !decided {
            pendingUndecided = true
            throw AccountError(failure: .usageConsent)
        }
        return isAccepted
    }

    private func report(_ day: UsageDay, token: String) async -> Bool {
        var request = URLRequest(url: config.issuer.appending(path: "api/v1/usage"), timeoutInterval: 20)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "sourceId": sourceID, "productSlug": "arion", "date": day.date, "visited": true,
            "sessions": day.sessions, "speechMs": day.speechMs, "words": day.words,
        ]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        guard let (data, response) = try? await http.data(for: request), (response as? HTTPURLResponse)?.statusCode ?? 0 < 300,
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any], object["success"] as? Bool == true,
              let echo = object["data"] as? [String: Any] else { return false }
        // The platform must confirm at least what was sent.
        return echo["productSlug"] as? String == "arion" && echo["date"] as? String == day.date
            && (echo["sessions"] as? Int ?? -1) >= day.sessions && (echo["speechMs"] as? Int64 ?? (echo["speechMs"] as? Int).map(Int64.init) ?? -1) >= day.speechMs
            && (echo["words"] as? Int ?? -1) >= day.words
    }

    /// A random source ID per installation, 32 hex characters.
    private var sourceID: String {
        if let id = defaults.string(forKey: Key.source), id.count == 32 { return id }
        let id = UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
        defaults.set(id, forKey: Key.source)
        return id
    }

    // MARK: State

    private func signedOut(failure: AccountFailure?) {
        accessToken = nil
        expiry = nil
        webSession = nil
        state.phase = .signedOut
        state.usageSharing = .unknown
        state.failure = failure
    }

    private func store(_ profile: AccountProfile) {
        defaults.set(["sub": profile.subject, "name": profile.name, "email": profile.email, "picture": profile.picture?.absoluteString ?? ""], forKey: Key.profile)
    }

    private func storedProfile() -> AccountProfile? {
        guard let stored = defaults.dictionary(forKey: Key.profile) as? [String: String], let subject = stored["sub"] else { return nil }
        return AccountProfile(subject: subject, name: stored["name"] ?? "", email: stored["email"] ?? "", picture: stored["picture"].flatMap(URL.init(string:)))
    }

    private static func form(_ values: [String: String]) -> Data {
        var parts = URLComponents()
        parts.queryItems = values.map { URLQueryItem(name: $0.key, value: $0.value) }
        let encoded = (parts.percentEncodedQuery ?? "").replacingOccurrences(of: "+", with: "%2B")
        return Data(encoded.utf8)
    }
}
