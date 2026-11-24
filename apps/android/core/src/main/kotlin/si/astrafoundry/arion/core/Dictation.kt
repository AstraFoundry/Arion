package si.astrafoundry.arion.core

/**
 * Text recognized so far: the finalized part and the tail the recognizer may
 * still revise. A draft is never target text; only a commit writes it.
 */
data class Draft(val finalized: String = "", val volatile: String = "") {
    val text: String get() = finalized + volatile
    val isEmpty: Boolean get() = text.isBlank()
}

/** Why a dictation could not start or finish. Each is shown with its remedy. */
enum class DictationFailure {
    MicrophoneDenied,
    RecognizerUnavailable,
    LanguageUnavailable,
    RecognizerBusy,
    CaptureEnded,
    RecognitionFailed,
    NothingHeard,
}

/** One dictation's lifecycle as the keyboard shows it. */
sealed interface DictationPhase {
    data object Idle : DictationPhase
    data object Starting : DictationPhase
    data class Listening(val draft: Draft, val level: Float) : DictationPhase
    data class Finishing(val draft: Draft) : DictationPhase
    data class Polishing(val draft: Draft, val translating: Boolean) : DictationPhase
    data class Failed(val failure: DictationFailure) : DictationPhase

    val isActive: Boolean
        get() = this is Starting || this is Listening || this is Finishing || this is Polishing
}

/** What to do with the speech once it ends. */
sealed interface DictationIntent {
    data object Dictate : DictationIntent
    data class Translate(val target: TranslationTarget) : DictationIntent
}

/** What a finished dictation delivers and how it was produced. */
data class Delivery(val text: String, val source: String, val outcome: Outcome) {
    enum class Outcome { Cleaned, Polished, Translated, PolishUnavailable, TranslationUnavailable }
}

/** The recognition languages Arion offers, as BCP 47 tags. */
object DictationLanguage {
    const val ENGLISH = "en-US"
    const val CHINESE = "zh-CN"
    val all = listOf(ENGLISH, CHINESE)

    /** The device's preferred languages among those offered; English when none match. */
    fun defaults(preferred: List<String>): List<String> {
        val result = mutableListOf<String>()
        for (tag in preferred) {
            val language = tag.substringBefore('-').substringBefore('_').lowercase()
            val match = all.firstOrNull { it.substringBefore('-') == language }
            if (match != null && match !in result) result += match
        }
        return result.ifEmpty { listOf(ENGLISH) }
    }
}
