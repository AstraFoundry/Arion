@testable import ArionKit
import XCTest

/// The same cases as `TestCountWords` in packages/core/domain/usage_test.go.
final class WordCountTests: XCTestCase {
    func testCountsAsTheDesktop() {
        let cases: [String: Int] = [
            "": 0,
            "Hello, world!": 2,
            "don't use e-mail at 3.5x": 5,
            "今天天气不错": 6,
            "把 README 发给 Alice，谢谢。": 7,
            "こんにちは": 5,
            "   \n\t": 0,
            "v2 build-42 ready.": 3,
            "结束了.": 3,
            "end.": 1,
        ]
        for (text, words) in cases {
            XCTAssertEqual(WordCount.count(text), words, text)
        }
    }
}

final class UsageLedgerTests: XCTestCase {
    private func ledger() -> UsageLedger {
        UsageLedger(url: FileManager.default.temporaryDirectory.appending(path: "usage-\(UUID().uuidString).json"))
    }

    func testAddsByDayWithScenes() throws {
        let usage = ledger()
        let day = Calendar.current.date(from: DateComponents(year: 2026, month: 10, day: 14, hour: 9))!
        try usage.record(speechMs: 30000, text: String(repeating: "word ", count: 50), scene: nil, at: day)
        try usage.record(speechMs: 30000, text: String(repeating: "word ", count: 70), scene: .email, at: day.addingTimeInterval(3 * 3600))
        try usage.record(speechMs: 6000, text: String(repeating: "word ", count: 10), scene: .email, at: day.addingTimeInterval(-2 * 86400))
        let days = usage.days()
        XCTAssertEqual(days.map(\.date), ["2026-10-12", "2026-10-14"])
        XCTAssertEqual(days[1].sessions, 2)
        XCTAssertEqual(days[1].words, 120)
        XCTAssertEqual(days[1].scenes, ["email": 70])
        XCTAssertEqual(days[1].polishedWords, 70)
        XCTAssertEqual(days[1].wordsPerMinute, 120)
        let total = usage.total(since: nil)
        XCTAssertEqual(total.words, 130)
        XCTAssertEqual(total.scenes["email"], 80)
    }

    func testFillsEmptyDays() throws {
        let usage = ledger()
        let end = Calendar.current.date(from: DateComponents(year: 2026, month: 10, day: 14, hour: 12))!
        try usage.record(speechMs: 1000, text: "hi", scene: nil, at: end)
        let days = usage.days(ending: end, count: 3)
        XCTAssertEqual(days.map(\.date), ["2026-10-12", "2026-10-13", "2026-10-14"])
        XCTAssertEqual(days.map(\.words), [0, 0, 1])
    }
}

/// The same cases as `TestDefaultModel` in apps/desktop/infrastructure/cloudpolish.
final class CloudTests: XCTestCase {
    func testDefaultModel() {
        let cases: [(CloudProvider, [String], String?)] = [
            (.openai, ["gpt-6-sol", "gpt-5.6-luna", "gpt-6-luna-2026-07-01", "gpt-6-luna", "gpt-6-mini"], "gpt-6-luna"),
            (.openai, ["gpt-6-sol", "gpt-5-mini", "gpt-5-mini-realtime-preview"], "gpt-5-mini"),
            (.anthropic, ["claude-opus-5-5", "claude-haiku-4-5-20251001", "claude-haiku-4-5", "claude-sonnet-5-5"], "claude-haiku-4-5"),
            (.deepseek, ["deepseek-v4-pro", "deepseek-flash"], "deepseek-flash"),
            (.deepseek, ["deepseek-reasoner", "deepseek-chat"], "deepseek-chat"),
            (.gemini, ["gemini-3.6-flash", "gemini-3.5-flash-lite", "gemini-3.1-flash-lite-preview"], "gemini-3.5-flash-lite"),
            (.openrouter, ["anthropic/claude-haiku-4.5", "openai/gpt-6-luna", "meta/llama"], "openai/gpt-6-luna"),
            (.openai, ["model-a", "model-b"], nil),
            (.openaiCompatible, ["gpt-6-luna", "custom"], "gpt-6-luna"),
            (.anthropicCompatible, ["claude-haiku-4-5", "local"], "claude-haiku-4-5"),
            (.anthropicCompatible, ["local-model"], nil),
        ]
        for (provider, models, want) in cases {
            XCTAssertEqual(CloudAPI.defaultModel(provider, models), want, "\(provider) \(models)")
        }
    }

    func testCompatibleAddress() throws {
        XCTAssertEqual(try CloudAPI.address(" https://example.com/v1/ ").absoluteString, "https://example.com/v1")
        for bad in ["ftp://example.com", "example.com", "https://user@example.com", "https://example.com/v1?x=1", "https://example.com#a", ""] {
            XCTAssertThrowsError(try CloudAPI.address(bad), bad)
        }
    }

    private func body(_ provider: CloudProvider) throws -> [String: Any] {
        let request = RewriteRequest(instructions: "system", prompt: "user", maximumTokens: 64)
        let url = try CloudAPI.completionRequest(provider, base: URL(string: "https://example.com/v1")!, key: "k", model: "m", request: request)
        return try XCTUnwrap(JSONSerialization.jsonObject(with: XCTUnwrap(url.httpBody)) as? [String: Any])
    }

    func testRequestsTurnThinkingOff() throws {
        XCTAssertEqual(try body(.openai)["reasoning_effort"] as? String, "none")
        XCTAssertEqual((try body(.anthropic)["thinking"] as? [String: String])?["type"], "disabled")
        XCTAssertEqual((try body(.deepseek)["thinking"] as? [String: String])?["type"], "disabled")
        XCTAssertEqual((try body(.openrouter)["reasoning"] as? [String: String])?["effort"], "none")
        let config = try XCTUnwrap(try body(.gemini)["generationConfig"] as? [String: Any])
        XCTAssertEqual((config["thinkingConfig"] as? [String: Int])?["thinkingBudget"], 0)
        XCTAssertEqual(try body(.anthropic)["system"] as? String, "system")
    }

    func testParsesCompletionsAndRefusesTruncation() {
        let chat = #"{"choices":[{"finish_reason":"stop","message":{"content":"Done."}}]}"#
        XCTAssertEqual(CloudAPI.parseCompletion(.openai, Data(chat.utf8)), "Done.")
        let cut = #"{"choices":[{"finish_reason":"length","message":{"content":"Do"}}]}"#
        XCTAssertNil(CloudAPI.parseCompletion(.openai, Data(cut.utf8)))
        let anthropic = #"{"stop_reason":"end_turn","content":[{"type":"text","text":"Hi."}]}"#
        XCTAssertEqual(CloudAPI.parseCompletion(.anthropic, Data(anthropic.utf8)), "Hi.")
        let models = #"{"models":[{"name":"models/gemini-3.5-flash-lite","supportedGenerationMethods":["generateContent"]},{"name":"models/embedding","supportedGenerationMethods":["embedContent"]}]}"#
        XCTAssertEqual(try CloudAPI.parseModels(.gemini, Data(models.utf8)), ["gemini-3.5-flash-lite"])
    }
}

final class AccountTests: XCTestCase {
    func testPKCEChallengeIsTheS256OfTheVerifier() {
        // RFC 7636 appendix B.
        XCTAssertEqual(OIDC.challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"), "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")
    }

    func testAuthorizeURLCarriesTheMobileClient() throws {
        let discovery = try JSONDecoder().decode(Discovery.self, from: Data(#"{"issuer":"https://astrafoundry.si","authorization_endpoint":"https://astrafoundry.si/oauth/authorize","token_endpoint":"https://astrafoundry.si/oauth/token"}"#.utf8))
        let url = try XCTUnwrap(OIDC.authorizeURL(discovery, AccountConfig(), state: "s", challenge: "c"))
        let items = Dictionary(uniqueKeysWithValues: (URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems ?? []).map { ($0.name, $0.value ?? "") })
        XCTAssertEqual(items["client_id"], "arion-ios")
        XCTAssertEqual(items["redirect_uri"], "arion://auth/callback")
        XCTAssertEqual(items["code_challenge_method"], "S256")
        XCTAssertEqual(items["resource"], "https://api.arion.astrafoundry.si")
        XCTAssertEqual(items["scope"], "openid profile offline_access arion.sync")
    }
}
