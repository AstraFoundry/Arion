package si.astrafoundry.arion.core

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test
import java.net.URI
import java.time.LocalDate

class BackendTest {
    /** The same cases as `TestCountWords` in packages/core/domain/usage_test.go. */
    @Test
    fun countsWordsAsTheDesktop() {
        mapOf(
            "" to 0, "Hello, world!" to 2, "don't use e-mail at 3.5x" to 5, "今天天气不错" to 6,
            "把 README 发给 Alice，谢谢。" to 7, "こんにちは" to 5, "   \n\t" to 0, "v2 build-42 ready." to 3,
            "结束了." to 3, "end." to 1,
        ).forEach { (text, words) -> assertEquals(text, words, countWords(text)) }
    }

    @Test
    fun usageLogAddsByDayWithScenes() {
        val day = LocalDate.of(2026, 10, 14)
        var log = UsageLog.add(emptyList(), day, 30000, "word ".repeat(50), null)
        log = UsageLog.add(log, day, 30000, "word ".repeat(70), Scene.Email)
        log = UsageLog.add(log, day.minusDays(2), 6000, "word ".repeat(10), Scene.Email)
        assertEquals(listOf("2026-10-12", "2026-10-14"), log.map { it.date })
        assertEquals(2, log[1].sessions)
        assertEquals(120, log[1].words)
        assertEquals(mapOf("email" to 70), log[1].scenes)
        assertEquals(120, log[1].wordsPerMinute)
        assertEquals(80, UsageLog.total(log, null).scenes["email"])
        assertEquals(listOf(10, 0, 120), UsageLog.days(log, "2026-10-14", 3).map { it.words })
        assertEquals(log, UsageLog.decode(UsageLog.encode(log)))
    }

    /** The same cases as `TestDefaultModel` in apps/desktop/infrastructure/cloudpolish. */
    @Test
    fun defaultModel() {
        listOf(
            Triple(CloudProvider.OpenAI, listOf("gpt-6-sol", "gpt-5.6-luna", "gpt-6-luna-2026-07-01", "gpt-6-luna", "gpt-6-mini"), "gpt-6-luna"),
            Triple(CloudProvider.OpenAI, listOf("gpt-6-sol", "gpt-5-mini", "gpt-5-mini-realtime-preview"), "gpt-5-mini"),
            Triple(CloudProvider.Anthropic, listOf("claude-opus-5-5", "claude-haiku-4-5-20251001", "claude-haiku-4-5", "claude-sonnet-5-5"), "claude-haiku-4-5"),
            Triple(CloudProvider.DeepSeek, listOf("deepseek-v4-pro", "deepseek-flash"), "deepseek-flash"),
            Triple(CloudProvider.DeepSeek, listOf("deepseek-reasoner", "deepseek-chat"), "deepseek-chat"),
            Triple(CloudProvider.Gemini, listOf("gemini-3.6-flash", "gemini-3.5-flash-lite", "gemini-3.1-flash-lite-preview"), "gemini-3.5-flash-lite"),
            Triple(CloudProvider.OpenRouter, listOf("anthropic/claude-haiku-4.5", "openai/gpt-6-luna", "meta/llama"), "openai/gpt-6-luna"),
            Triple(CloudProvider.OpenAI, listOf("model-a", "model-b"), null),
            Triple(CloudProvider.OpenAICompatible, listOf("gpt-6-luna", "custom"), "gpt-6-luna"),
            Triple(CloudProvider.AnthropicCompatible, listOf("claude-haiku-4-5", "local"), "claude-haiku-4-5"),
            Triple(CloudProvider.AnthropicCompatible, listOf("local-model"), null),
        ).forEach { (provider, models, want) -> assertEquals("$provider $models", want, CloudApi.defaultModel(provider, models)) }
    }

    @Test
    fun compatibleAddress() {
        assertEquals("https://example.com/v1", CloudApi.address(" https://example.com/v1/ "))
        listOf("ftp://example.com", "example.com", "https://user@example.com", "https://example.com/v1?x=1", "https://example.com#a", "").forEach {
            assertThrows(it, CloudException::class.java) { CloudApi.address(it) }
        }
    }

    @Test
    fun requestsTurnThinkingOff() {
        val request = RewriteRequest("system", "user", 64)
        fun body(provider: CloudProvider) = Json.parseToJsonElement(CloudApi.completion(provider, "https://example.com/v1", "m", request).second).jsonObject
        assertEquals("none", body(CloudProvider.OpenAI)["reasoning_effort"]!!.jsonPrimitive.content)
        assertEquals("disabled", body(CloudProvider.Anthropic)["thinking"]!!.jsonObject["type"]!!.jsonPrimitive.content)
        assertEquals("disabled", body(CloudProvider.DeepSeek)["thinking"]!!.jsonObject["type"]!!.jsonPrimitive.content)
        assertEquals("none", body(CloudProvider.OpenRouter)["reasoning"]!!.jsonObject["effort"]!!.jsonPrimitive.content)
        assertEquals("0", body(CloudProvider.Gemini)["generationConfig"]!!.jsonObject["thinkingConfig"]!!.jsonObject["thinkingBudget"]!!.jsonPrimitive.content)
        assertEquals("https://example.com/v1/messages", CloudApi.completion(CloudProvider.AnthropicCompatible, "https://example.com/v1", "m", request).first)
    }

    @Test
    fun parsesCompletionsAndRefusesTruncation() {
        assertEquals("Done.", CloudApi.parseCompletion(CloudProvider.OpenAI, """{"choices":[{"finish_reason":"stop","message":{"content":"Done."}}]}"""))
        assertNull(CloudApi.parseCompletion(CloudProvider.OpenAI, """{"choices":[{"finish_reason":"length","message":{"content":"Do"}}]}"""))
        assertEquals("Hi.", CloudApi.parseCompletion(CloudProvider.Anthropic, """{"stop_reason":"end_turn","content":[{"type":"text","text":"Hi."}]}"""))
        assertEquals(listOf("gemini-3.5-flash-lite"), CloudApi.parseModels(CloudProvider.Gemini,
            """{"models":[{"name":"models/gemini-3.5-flash-lite","supportedGenerationMethods":["generateContent"]},{"name":"models/e","supportedGenerationMethods":["embedContent"]}]}"""))
    }

    @Test
    fun pkceAndAuthorizeUrl() {
        // RFC 7636 appendix B.
        assertEquals("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", Oidc.challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
        val discovery = Discovery("https://astrafoundry.si", "https://astrafoundry.si/oauth/authorize", "https://astrafoundry.si/oauth/token")
        val query = URI(Oidc.authorizeUrl(discovery, AccountConfig(), "s", "c")).query.split('&').associate { it.substringBefore('=') to it.substringAfter('=') }
        assertEquals("arion-android", query["client_id"])
        assertEquals("arion://auth/callback", query["redirect_uri"])
        assertEquals("S256", query["code_challenge_method"])
        assertEquals("openid profile offline_access arion.sync", query["scope"])
        assertTrue(Oidc.accepts(discovery, AccountConfig()))
        assertFalse(Oidc.accepts(discovery.copy(issuer = "https://other.example"), AccountConfig()))
    }

    @Test
    fun consentAndReportAnswers() {
        assertEquals(true, Oidc.consent("""{"data":{"accepted":true,"decided":true,"version":"usage-v1"}}""", null))
        assertNull(Oidc.consent("""{"data":{"accepted":false,"decided":false}}""", null))
        assertThrows(IllegalStateException::class.java) { Oidc.consent("""{"data":{"accepted":true,"decided":true,"version":"usage-v0"}}""", null) }
        assertThrows(IllegalStateException::class.java) { Oidc.consent("""{"data":{"accepted":false,"decided":true}}""", true) }
        val day = UsageDay("2026-10-14", sessions = 2, speechMs = 1000, words = 10)
        assertTrue(Oidc.confirmsReport("""{"success":true,"data":{"productSlug":"arion","date":"2026-10-14","sessions":2,"speechMs":1000,"words":12}}""", day))
        assertFalse(Oidc.confirmsReport("""{"success":true,"data":{"productSlug":"arion","date":"2026-10-14","sessions":1,"speechMs":1000,"words":12}}""", day))
    }
}
