package si.astrafoundry.arion.core

import kotlinx.coroutines.flow.StateFlow

/** Where polishing and translation run. */
enum class PolishEngine(val id: String) { Device("device"), Cloud("cloud") }

/**
 * Cloud providers the user connects with their own API key, as on the
 * desktop (docs/project/cloud-polishing.md). The ChatGPT subscription
 * sign-in is desktop-only: its callback is a fixed loopback port.
 */
enum class CloudProvider(val id: String, val needsBaseUrl: Boolean = false) {
    OpenAI("openai"),
    Anthropic("anthropic"),
    Gemini("gemini"),
    DeepSeek("deepseek"),
    OpenRouter("openrouter"),
    OpenAICompatible("openai-compatible", needsBaseUrl = true),
    AnthropicCompatible("anthropic-compatible", needsBaseUrl = true),
    ;

    companion object {
        fun of(id: String): CloudProvider? = entries.firstOrNull { it.id == id }
    }
}

/** Content-free connection failures, as `cloudpolish.Code` on the desktop. */
enum class CloudFailure { Rejected, Network, Models, Store, Address }

class CloudException(val failure: CloudFailure) : Exception(failure.name)

data class CloudConnection(val provider: CloudProvider, val model: String?, val baseUrl: String?)

data class CloudState(
    /** The provider in use, if any. */
    val active: CloudConnection? = null,
    /** Providers with a stored key, the active one included. */
    val saved: Set<CloudProvider> = emptySet(),
)

/** The connected providers. Keys are encrypted with an Android Keystore key. */
interface CloudServing {
    val state: StateFlow<CloudState>
    /**
     * Lists the provider's models with the key; stores it and makes the
     * provider active only when that works. Throws [CloudException].
     */
    suspend fun connect(provider: CloudProvider, key: String, baseUrl: String?): List<String>
    /** Makes a provider with a stored key active. */
    fun activate(provider: CloudProvider)
    /** Deletes the active provider's key. */
    fun disconnect()
    /** The active provider's models. Throws [CloudException]. */
    suspend fun models(): List<String>
    fun choose(model: String)
}
