package si.astrafoundry.arion.speech

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.speech.RecognitionListener
import android.speech.RecognitionSupport
import android.speech.RecognitionSupportCallback
import android.speech.RecognizerIntent
import android.speech.SpeechRecognizer
import java.util.concurrent.Executor
import si.astrafoundry.arion.core.DictationFailure
import si.astrafoundry.arion.core.Draft

/**
 * Android's on-device speech recognizer. Audio is captured and recognized by
 * the system's on-device service; it is never sent to a server.
 */
class OnDeviceRecognizer(private val context: Context, private val executor: Executor) {
    interface Listener {
        fun onReady()
        fun onDraft(draft: Draft)
        fun onLevel(level: Float)
        /** The recognizer ended: on stop, or after a long pause. */
        fun onFinal(text: String)
        fun onFailure(failure: DictationFailure)
    }

    private var recognizer: SpeechRecognizer? = null

    val isAvailable: Boolean get() = SpeechRecognizer.isOnDeviceRecognitionAvailable(context)

    fun start(language: String, vocabulary: List<String>, listener: Listener) {
        cancel()
        if (!isAvailable) {
            listener.onFailure(DictationFailure.RecognizerUnavailable)
            return
        }
        val recognizer = SpeechRecognizer.createOnDeviceSpeechRecognizer(context)
        this.recognizer = recognizer
        var finalized = ""
        recognizer.setRecognitionListener(object : RecognitionListener {
            override fun onReadyForSpeech(params: Bundle?) = listener.onReady()
            override fun onBeginningOfSpeech() {}
            override fun onRmsChanged(rmsdB: Float) = listener.onLevel(((rmsdB + 2f) / 12f).coerceIn(0f, 1f))
            override fun onBufferReceived(buffer: ByteArray?) {}
            override fun onEndOfSpeech() {}
            override fun onEvent(eventType: Int, params: Bundle?) {}

            override fun onPartialResults(partialResults: Bundle?) {
                val text = partialResults.best() ?: return
                listener.onDraft(Draft(finalized, text))
            }

            override fun onSegmentResults(segmentResults: Bundle) {
                val text = segmentResults.best() ?: return
                finalized = join(finalized, text)
                listener.onDraft(Draft(finalized, ""))
            }

            override fun onEndOfSegmentedSession() {
                this@OnDeviceRecognizer.recognizer = null
                recognizer.destroy()
                listener.onFinal(finalized)
            }

            override fun onResults(results: Bundle?) {
                this@OnDeviceRecognizer.recognizer = null
                recognizer.destroy()
                listener.onFinal(join(finalized, results.best().orEmpty()))
            }

            override fun onError(error: Int) {
                this@OnDeviceRecognizer.recognizer = null
                recognizer.destroy()
                if ((error == SpeechRecognizer.ERROR_NO_MATCH || error == SpeechRecognizer.ERROR_SPEECH_TIMEOUT) && finalized.isNotBlank()) {
                    listener.onFinal(finalized)
                    return
                }
                if (error == SpeechRecognizer.ERROR_LANGUAGE_UNAVAILABLE) download(language)
                listener.onFailure(failure(error))
            }
        })
        recognizer.startListening(intent(language, vocabulary))
    }

    /** Ends listening; the final result arrives through the listener. */
    fun stop() {
        recognizer?.stopListening()
    }

    /** Drops the dictation; nothing more is reported. */
    fun cancel() {
        recognizer?.let {
            it.setRecognitionListener(null)
            it.cancel()
            it.destroy()
        }
        recognizer = null
    }

    /** Asks the system to download the on-device model for a language. */
    fun download(language: String) {
        val recognizer = SpeechRecognizer.createOnDeviceSpeechRecognizer(context)
        recognizer.triggerModelDownload(intent(language, emptyList()))
        recognizer.destroy()
    }

    /** Whether the on-device model for the language is installed. */
    fun checkInstalled(language: String, result: (Boolean?) -> Unit) {
        if (!isAvailable) return result(null)
        val recognizer = SpeechRecognizer.createOnDeviceSpeechRecognizer(context)
        recognizer.checkRecognitionSupport(intent(language, emptyList()), executor, object : RecognitionSupportCallback {
            override fun onSupportResult(support: RecognitionSupport) {
                recognizer.destroy()
                result(language in support.installedOnDeviceLanguages || support.installedOnDeviceLanguages.any { it.equals(language, ignoreCase = true) })
            }

            override fun onError(error: Int) {
                recognizer.destroy()
                result(null)
            }
        })
    }

    private fun intent(language: String, vocabulary: List<String>) = Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply {
        putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM)
        putExtra(RecognizerIntent.EXTRA_LANGUAGE, language)
        putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, true)
        putExtra(RecognizerIntent.EXTRA_PREFER_OFFLINE, true)
        putExtra(RecognizerIntent.EXTRA_ENABLE_FORMATTING, RecognizerIntent.FORMATTING_OPTIMIZE_QUALITY)
        if (vocabulary.isNotEmpty()) putExtra(RecognizerIntent.EXTRA_BIASING_STRINGS, ArrayList(vocabulary.take(50)))
    }

    private fun Bundle?.best(): String? = this?.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION)?.firstOrNull()?.takeIf { it.isNotBlank() }

    private fun join(a: String, b: String) = when {
        a.isEmpty() -> b
        b.isEmpty() -> a
        else -> "$a $b"
    }

    private fun failure(error: Int): DictationFailure = when (error) {
        SpeechRecognizer.ERROR_INSUFFICIENT_PERMISSIONS -> DictationFailure.MicrophoneDenied
        SpeechRecognizer.ERROR_NO_MATCH, SpeechRecognizer.ERROR_SPEECH_TIMEOUT -> DictationFailure.NothingHeard
        SpeechRecognizer.ERROR_RECOGNIZER_BUSY, SpeechRecognizer.ERROR_TOO_MANY_REQUESTS -> DictationFailure.RecognizerBusy
        SpeechRecognizer.ERROR_LANGUAGE_NOT_SUPPORTED, SpeechRecognizer.ERROR_LANGUAGE_UNAVAILABLE,
        SpeechRecognizer.ERROR_NETWORK, SpeechRecognizer.ERROR_NETWORK_TIMEOUT -> DictationFailure.LanguageUnavailable
        SpeechRecognizer.ERROR_AUDIO -> DictationFailure.CaptureEnded
        else -> DictationFailure.RecognitionFailed
    }
}
