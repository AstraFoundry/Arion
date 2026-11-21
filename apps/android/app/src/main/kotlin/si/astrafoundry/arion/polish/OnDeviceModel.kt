package si.astrafoundry.arion.polish

import com.google.mlkit.genai.common.FeatureStatus
import com.google.mlkit.genai.common.GenAiException
import com.google.mlkit.genai.prompt.Generation
import com.google.mlkit.genai.prompt.SystemInstruction
import com.google.mlkit.genai.prompt.TextPart
import com.google.mlkit.genai.prompt.generateContentRequest
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout
import si.astrafoundry.arion.core.RewriteRequest
import si.astrafoundry.arion.core.RewriteValidation

/** Whether Gemini Nano can polish on this phone. */
enum class ModelAvailability { Available, Downloadable, Downloading, Unavailable }

/**
 * Gemini Nano through ML Kit's Prompt API, running the shared recipe on the
 * phone. ML Kit serves inference only to the foreground app; when the editor
 * is in front the request may be refused, and the dictation is delivered
 * without polishing.
 */
class OnDeviceModel {
    private val model by lazy { Generation.getClient() }

    suspend fun availability(): ModelAvailability = try {
        when (model.checkStatus()) {
            FeatureStatus.AVAILABLE -> ModelAvailability.Available
            FeatureStatus.DOWNLOADABLE -> ModelAvailability.Downloadable
            FeatureStatus.DOWNLOADING -> ModelAvailability.Downloading
            else -> ModelAvailability.Unavailable
        }
    } catch (error: GenAiException) {
        ModelAvailability.Unavailable
    }

    /** Starts the model download where the phone supports it. */
    suspend fun download() {
        model.download().collect { }
    }

    sealed class Failure(message: String) : Exception(message) {
        class Unavailable(cause: String) : Failure(cause)
        class TimedOut : Failure("timed out")
        class Rejected : Failure("rejected")
    }

    /** Runs one request and returns the validated proposal. */
    suspend fun run(request: RewriteRequest): String {
        val text = try {
            withTimeout(TIMEOUT_MILLIS) {
                model.generateContent(
                    generateContentRequest(TextPart(request.prompt)) {
                        systemInstruction = SystemInstruction(request.instructions)
                        temperature = 0f
                        topK = 1
                        maxOutputTokens = request.maximumTokens
                    },
                ).candidates.firstOrNull()?.text.orEmpty()
            }
        } catch (error: TimeoutCancellationException) {
            throw Failure.TimedOut()
        } catch (error: GenAiException) {
            throw Failure.Unavailable(error.errorCode.toString())
        }
        if (!RewriteValidation.accepts(text)) throw Failure.Rejected()
        return text.trim()
    }

    companion object {
        /** The desktop bounds the final rewrite to the same 15 seconds. */
        const val TIMEOUT_MILLIS = 15_000L
    }
}
