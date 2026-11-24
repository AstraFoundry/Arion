package si.astrafoundry.arion.cloud

import android.content.Context
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import si.astrafoundry.arion.core.CloudApi
import si.astrafoundry.arion.core.CloudConnection
import si.astrafoundry.arion.core.CloudException
import si.astrafoundry.arion.core.CloudFailure
import si.astrafoundry.arion.core.CloudProvider
import si.astrafoundry.arion.core.CloudServing
import si.astrafoundry.arion.core.CloudState
import si.astrafoundry.arion.core.RewriteRequest
import si.astrafoundry.arion.core.RewriteValidation
import si.astrafoundry.arion.storage.Secrets
import java.io.IOException

/** One provider's stored credential, as the desktop's `cloud-polish/<provider>` item. */
@Serializable
internal data class CloudCredential(val provider: String, val key: String, val model: String? = null, val baseUrl: String? = null)

/**
 * The connected providers, with one encrypted credential per provider and an
 * active-provider pointer, as the desktop's cloud polishing service. The
 * app and the input method share this process-wide instance.
 */
class CloudService private constructor(context: Context) : CloudServing {
    private val secrets = Secrets(context)
    private val json = Json { ignoreUnknownKeys = true }
    private val _state = MutableStateFlow(CloudState())
    override val state: StateFlow<CloudState> = _state

    init {
        reload()
    }

    private fun account(provider: CloudProvider) = "cloud-polish/${provider.id}"

    internal fun credential(provider: CloudProvider): CloudCredential? = try {
        secrets.read(account(provider))?.let { json.decodeFromString(CloudCredential.serializer(), it) }
    } catch (e: Exception) {
        throw CloudException(CloudFailure.Store)
    }

    private fun reload() {
        val saved = CloudProvider.entries.filter { secrets.has(account(it)) }.toSet()
        val active = runCatching { secrets.read(POINTER) }.getOrNull()?.let(CloudProvider::of)?.let { provider ->
            runCatching { credential(provider) }.getOrNull()?.let { CloudConnection(provider, it.model, it.baseUrl) }
        }
        _state.value = CloudState(active, saved)
    }

    override suspend fun connect(provider: CloudProvider, key: String, baseUrl: String?): List<String> {
        val custom = if (provider.needsBaseUrl) CloudApi.address(baseUrl) else null
        val base = CloudApi.base(provider, custom) ?: throw CloudException(CloudFailure.Address)
        val models = list(provider, base, key)
        val previous = runCatching { credential(provider) }.getOrNull()?.model?.takeIf { it in models }
        store(CloudCredential(provider.id, key, previous ?: CloudApi.defaultModel(provider, models), custom), provider)
        pointer(provider)
        reload()
        return models
    }

    override fun activate(provider: CloudProvider) {
        if (credential(provider) == null) throw CloudException(CloudFailure.Store)
        pointer(provider)
        reload()
    }

    override fun disconnect() {
        val provider = _state.value.active?.provider ?: return
        try {
            secrets.delete(account(provider))
            secrets.delete(POINTER)
        } catch (e: Exception) {
            throw CloudException(CloudFailure.Store)
        }
        reload()
    }

    override suspend fun models(): List<String> {
        val provider = _state.value.active?.provider ?: throw CloudException(CloudFailure.Store)
        val credential = credential(provider) ?: throw CloudException(CloudFailure.Store)
        val base = CloudApi.base(provider, credential.baseUrl) ?: throw CloudException(CloudFailure.Store)
        return list(provider, base, credential.key)
    }

    override fun choose(model: String) {
        val provider = _state.value.active?.provider ?: return
        val credential = credential(provider) ?: return
        store(credential.copy(model = model), provider)
        reload()
    }

    private suspend fun list(provider: CloudProvider, base: String, key: String): List<String> {
        val result = try {
            http("$base/models", headers = CloudApi.headers(provider, key))
        } catch (e: IOException) {
            throw CloudException(CloudFailure.Network)
        }
        if (result.status == 401) throw CloudException(CloudFailure.Rejected)
        if (result.status != 200) throw CloudException(CloudFailure.Models)
        return CloudApi.parseModels(provider, result.body)
    }

    private fun store(credential: CloudCredential, provider: CloudProvider) {
        try {
            secrets.write(account(provider), json.encodeToString(CloudCredential.serializer(), credential))
        } catch (e: Exception) {
            throw CloudException(CloudFailure.Store)
        }
    }

    private fun pointer(provider: CloudProvider) {
        try {
            secrets.write(POINTER, provider.id)
        } catch (e: Exception) {
            throw CloudException(CloudFailure.Store)
        }
    }

    /** Runs the shared recipe on the provider and model in use: 15 seconds, validated. */
    suspend fun run(request: RewriteRequest): String {
        val provider = _state.value.active?.provider ?: error("no cloud provider")
        val credential = credential(provider) ?: error("no cloud credential")
        val model = credential.model ?: error("no cloud model")
        val base = CloudApi.base(provider, credential.baseUrl) ?: error("no cloud address")
        val (url, body) = CloudApi.completion(provider, base, model, request)
        val result = try {
            withTimeout(TIMEOUT_MS) { http(url, "POST", CloudApi.headers(provider, credential.key), body, timeoutMs = TIMEOUT_MS.toInt()) }
        } catch (e: TimeoutCancellationException) {
            error("cloud polishing timed out")
        }
        check(result.status == 200) { "cloud polishing failed with HTTP ${result.status}" }
        val text = CloudApi.parseCompletion(provider, result.body) ?: error("incomplete cloud answer")
        check(RewriteValidation.accepts(text)) { "cloud answer failed the delivery check" }
        return text.trim()
    }

    /** Whether a provider with a chosen model is connected. */
    val ready: Boolean get() = _state.value.active?.model != null

    companion object {
        private const val POINTER = "cloud-polish"
        private const val TIMEOUT_MS = 15000L

        @Volatile private var instance: CloudService? = null

        fun get(context: Context): CloudService =
            instance ?: synchronized(this) { instance ?: CloudService(context.applicationContext).also { instance = it } }
    }
}
