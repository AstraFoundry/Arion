package si.astrafoundry.arion.core

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray
import kotlinx.serialization.json.putJsonObject
import java.net.URI

/** Endpoints, request bodies and default models per provider, as the desktop's Republic adapters. */
object CloudApi {
    fun base(provider: CloudProvider, custom: String?): String? = when (provider) {
        CloudProvider.OpenAI -> "https://api.openai.com/v1"
        CloudProvider.Anthropic -> "https://api.anthropic.com/v1"
        CloudProvider.Gemini -> "https://generativelanguage.googleapis.com/v1beta"
        CloudProvider.DeepSeek -> "https://api.deepseek.com/v1"
        CloudProvider.OpenRouter -> "https://openrouter.ai/api/v1"
        CloudProvider.OpenAICompatible, CloudProvider.AnthropicCompatible -> custom
    }

    /** Authentication headers for a provider. */
    fun headers(provider: CloudProvider, key: String): Map<String, String> = when (provider) {
        CloudProvider.Anthropic, CloudProvider.AnthropicCompatible -> mapOf("x-api-key" to key, "anthropic-version" to "2023-06-01")
        CloudProvider.Gemini -> mapOf("x-goog-api-key" to key)
        else -> mapOf("Authorization" to "Bearer $key")
    }

    /**
     * Validates a compatible endpoint: http or https with a host, no query,
     * fragment or user info, stored without a trailing slash.
     */
    fun address(text: String?): String {
        val trimmed = text.orEmpty().trim()
        val uri = runCatching { URI(trimmed) }.getOrNull() ?: throw CloudException(CloudFailure.Address)
        if (uri.scheme?.lowercase() !in setOf("http", "https") || uri.host.isNullOrEmpty() || uri.rawQuery != null || uri.rawFragment != null || uri.rawUserInfo != null) {
            throw CloudException(CloudFailure.Address)
        }
        return trimmed.trimEnd('/')
    }

    /** Model IDs from a list response: `data[].id`, or Gemini's `models[].name`. */
    fun parseModels(provider: CloudProvider, body: String): List<String> {
        val root = runCatching { Json.parseToJsonElement(body).jsonObject }.getOrNull() ?: throw CloudException(CloudFailure.Models)
        if (provider == CloudProvider.Gemini) {
            val models = (root["models"] as? JsonArray) ?: throw CloudException(CloudFailure.Models)
            return models.mapNotNull { element ->
                val model = element.jsonObject
                val methods = (model["supportedGenerationMethods"] as? JsonArray)?.map { it.jsonPrimitive.content } ?: listOf("generateContent")
                model["name"]?.jsonPrimitive?.contentOrNull?.takeIf { "generateContent" in methods }?.removePrefix("models/")
            }
        }
        val data = (root["data"] as? JsonArray) ?: throw CloudException(CloudFailure.Models)
        return data.mapNotNull { (it as? JsonObject)?.get("id")?.jsonPrimitive?.contentOrNull }
    }

    /** The completion URL and body: instructions as the system message, the prompt as the user message, greedy, thinking off. */
    fun completion(provider: CloudProvider, base: String, model: String, request: RewriteRequest): Pair<String, String> {
        val body: JsonObject = when (provider) {
            CloudProvider.Anthropic, CloudProvider.AnthropicCompatible -> buildJsonObject {
                put("model", model)
                put("max_tokens", request.maximumTokens)
                put("temperature", 0)
                put("system", request.instructions)
                putJsonArray("messages") { add(buildJsonObject { put("role", "user"); put("content", request.prompt) }) }
                putJsonObject("thinking") { put("type", "disabled") }
            }
            CloudProvider.Gemini -> buildJsonObject {
                putJsonObject("systemInstruction") { putJsonArray("parts") { add(buildJsonObject { put("text", request.instructions) }) } }
                putJsonArray("contents") {
                    add(buildJsonObject { put("role", "user"); putJsonArray("parts") { add(buildJsonObject { put("text", request.prompt) }) } })
                }
                putJsonObject("generationConfig") {
                    put("temperature", 0)
                    put("maxOutputTokens", request.maximumTokens)
                    putJsonObject("thinkingConfig") { put("thinkingBudget", 0) }
                }
            }
            else -> buildJsonObject {
                put("model", model)
                put("temperature", 0)
                put("messages", buildJsonArray {
                    add(buildJsonObject { put("role", "system"); put("content", request.instructions) })
                    add(buildJsonObject { put("role", "user"); put("content", request.prompt) })
                })
                when (provider) {
                    CloudProvider.OpenAI -> { put("max_completion_tokens", request.maximumTokens); put("reasoning_effort", "none") }
                    CloudProvider.DeepSeek -> { put("max_tokens", request.maximumTokens); putJsonObject("thinking") { put("type", "disabled") } }
                    CloudProvider.OpenRouter -> { put("max_tokens", request.maximumTokens); putJsonObject("reasoning") { put("effort", "none") } }
                    else -> { put("max_tokens", request.maximumTokens); put("reasoning_effort", "none") }
                }
            }
        }
        val url = when (provider) {
            CloudProvider.Anthropic, CloudProvider.AnthropicCompatible -> "$base/messages"
            CloudProvider.Gemini -> "$base/models/$model:generateContent"
            else -> "$base/chat/completions"
        }
        return url to body.toString()
    }

    /** The completion text; null when the response is not a complete answer. */
    fun parseCompletion(provider: CloudProvider, body: String): String? {
        val root = runCatching { Json.parseToJsonElement(body).jsonObject }.getOrNull() ?: return null
        return when (provider) {
            CloudProvider.Anthropic, CloudProvider.AnthropicCompatible -> {
                if (root["stop_reason"]?.jsonPrimitive?.contentOrNull == "max_tokens") return null
                (root["content"] as? JsonArray)?.filter { it.jsonObject["type"]?.jsonPrimitive?.contentOrNull == "text" }
                    ?.joinToString("") { it.jsonObject["text"]?.jsonPrimitive?.contentOrNull.orEmpty() }
            }
            CloudProvider.Gemini -> {
                val candidate = (root["candidates"] as? JsonArray)?.firstOrNull()?.jsonObject ?: return null
                val finish = candidate["finishReason"]?.jsonPrimitive?.contentOrNull
                if (finish != null && finish != "STOP") return null
                (candidate["content"]?.jsonObject?.get("parts") as? JsonArray)
                    ?.filter { it.jsonObject["thought"]?.jsonPrimitive?.booleanOrNull != true }
                    ?.joinToString("") { it.jsonObject["text"]?.jsonPrimitive?.contentOrNull.orEmpty() }
            }
            else -> {
                val choice = (root["choices"] as? JsonArray)?.firstOrNull()?.jsonObject ?: return null
                if (choice["finish_reason"]?.jsonPrimitive?.contentOrNull == "length") return null
                choice["message"]?.jsonObject?.get("content")?.let { (it as? JsonPrimitive)?.contentOrNull }
            }
        }
    }

    // Default model, as `cloudpolish.DefaultModel`.
    private val tiers: Map<CloudProvider, List<List<String>>> = mapOf(
        CloudProvider.OpenAI to listOf(listOf("luna"), listOf("mini")),
        CloudProvider.OpenAICompatible to listOf(listOf("luna"), listOf("mini")),
        CloudProvider.Anthropic to listOf(listOf("haiku")),
        CloudProvider.AnthropicCompatible to listOf(listOf("haiku")),
        CloudProvider.Gemini to listOf(listOf("flash-lite"), listOf("flash")),
        CloudProvider.DeepSeek to listOf(listOf("flash"), listOf("deepseek-chat")),
        CloudProvider.OpenRouter to listOf(listOf("openai/", "luna"), listOf("anthropic/", "haiku"), listOf("deepseek/", "flash"), listOf("google/", "flash-lite")),
    )
    private val unsuited = listOf("preview", "exp", "realtime", "audio", "tts", "image", "transcribe", "embed", "search", "vision")
    private val dated = Regex("""\d{4}-\d{2}-\d{2}|\d{4,}""")

    /** The newest model of the provider's first tier that has one; null leaves the choice to the user. */
    fun defaultModel(provider: CloudProvider, models: List<String>): String? {
        for (tier in tiers[provider].orEmpty()) {
            var best: String? = null
            for (id in models) {
                val lower = id.lowercase()
                if (unsuited.none { it in lower } && tier.all { it in lower } && (best == null || newer(id, best))) best = id
            }
            if (best != null) return best
        }
        return null
    }

    private fun newer(a: String, b: String): Boolean {
        val va = version(a)
        val vb = version(b)
        for (i in 0 until minOf(va.size, vb.size)) if (va[i] != vb[i]) return va[i] > vb[i]
        if (va.size != vb.size) return va.size > vb.size
        return a.length < b.length
    }

    private fun version(id: String): List<Int> =
        dated.replace(id, "").split(Regex("""\D+""")).filter { it.isNotEmpty() }.map { it.toInt() }
}

