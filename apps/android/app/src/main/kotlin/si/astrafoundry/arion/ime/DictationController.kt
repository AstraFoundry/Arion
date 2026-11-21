package si.astrafoundry.arion.ime

import android.content.Context
import androidx.core.content.ContextCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import si.astrafoundry.arion.core.CorrectionRules
import si.astrafoundry.arion.core.Delivery
import si.astrafoundry.arion.core.DictationFailure
import si.astrafoundry.arion.core.DictationIntent
import si.astrafoundry.arion.core.DictationPhase
import si.astrafoundry.arion.core.Draft
import si.astrafoundry.arion.core.RewriteRecipe
import si.astrafoundry.arion.core.Scene
import si.astrafoundry.arion.polish.ModelAvailability
import si.astrafoundry.arion.polish.OnDeviceModel
import si.astrafoundry.arion.speech.OnDeviceRecognizer
import si.astrafoundry.arion.storage.Preferences

/**
 * Runs one keyboard dictation at a time: on-device recognition, rules, then
 * polishing or translation. Every callback is checked against the current
 * generation, so a cancelled or replaced dictation cannot deliver text.
 */
class DictationController(
    context: Context,
    private val scope: CoroutineScope,
    private val preferences: Preferences,
    private val deliver: (Delivery) -> Unit,
) {
    private val recognizer = OnDeviceRecognizer(context, ContextCompat.getMainExecutor(context))
    private val model = OnDeviceModel()
    private val recipe: RewriteRecipe? = runCatching {
        context.assets.open("format.json").bufferedReader().use { RewriteRecipe(it.readText()) }
    }.getOrNull()

    private val _phase = MutableStateFlow<DictationPhase>(DictationPhase.Idle)
    val phase: StateFlow<DictationPhase> = _phase

    /** Increments on every start and cancel; older callbacks are ignored. */
    private var generation = 0L
    private var draft = Draft()
    private var intent: DictationIntent = DictationIntent.Dictate
    private var scene = Scene.Chat
    private var finishing: Job? = null

    fun start(language: String, vocabulary: List<String>, scene: Scene) {
        if (_phase.value.isActive) return
        val current = ++generation
        draft = Draft()
        intent = DictationIntent.Dictate
        this.scene = scene
        _phase.value = DictationPhase.Starting
        recognizer.start(language, vocabulary, object : OnDeviceRecognizer.Listener {
            override fun onReady() {
                if (current == generation && _phase.value == DictationPhase.Starting) _phase.value = DictationPhase.Listening(draft, 0f)
            }

            override fun onDraft(draft: Draft) {
                if (current != generation || _phase.value !is DictationPhase.Listening && _phase.value != DictationPhase.Starting) return
                this@DictationController.draft = draft
                _phase.value = DictationPhase.Listening(draft, (_phase.value as? DictationPhase.Listening)?.level ?: 0f)
            }

            override fun onLevel(level: Float) {
                val listening = _phase.value as? DictationPhase.Listening ?: return
                if (current == generation) _phase.value = listening.copy(level = level)
            }

            override fun onFinal(text: String) {
                if (current == generation) complete(text, current)
            }

            override fun onFailure(failure: DictationFailure) {
                if (current == generation) _phase.value = DictationPhase.Failed(failure)
            }
        })
    }

    /** Stops listening; the delivery follows when recognition completes. */
    fun finish(intent: DictationIntent) {
        val phase = _phase.value
        if (phase !is DictationPhase.Listening && phase != DictationPhase.Starting) return
        this.intent = intent
        _phase.value = DictationPhase.Finishing(draft)
        recognizer.stop()
    }

    /** Drops the dictation in flight; nothing it produced is delivered. */
    fun cancel() {
        generation++
        recognizer.cancel()
        finishing?.cancel()
        finishing = null
        _phase.value = DictationPhase.Idle
    }

    fun dismissFailure() {
        if (_phase.value is DictationPhase.Failed) _phase.value = DictationPhase.Idle
    }

    private fun complete(raw: String, current: Long) {
        val source = CorrectionRules.apply(raw)
        if (source.isEmpty()) {
            _phase.value = DictationPhase.Failed(DictationFailure.NothingHeard)
            return
        }
        val intent = intent
        finishing = scope.launch {
            val delivery = produce(source, intent)
            if (current != generation) return@launch
            _phase.value = DictationPhase.Idle
            deliver(delivery)
        }
    }

    private suspend fun produce(source: String, intent: DictationIntent): Delivery {
        when (intent) {
            is DictationIntent.Translate -> {
                val request = recipe?.translate(source, scene, intent.target)
                    ?: return Delivery(source, source, Delivery.Outcome.TranslationUnavailable)
                _phase.value = DictationPhase.Polishing(Draft(source), translating = true)
                return runCatching { model.run(request) }
                    .map { Delivery(it, source, Delivery.Outcome.Translated) }
                    .getOrElse { Delivery(source, source, Delivery.Outcome.TranslationUnavailable) }
            }
            DictationIntent.Dictate -> {
                if (!preferences.polishingEnabled) return Delivery(source, source, Delivery.Outcome.Cleaned)
                val recipe = recipe
                if (recipe == null || model.availability() != ModelAvailability.Available) {
                    return Delivery(source, source, Delivery.Outcome.PolishUnavailable)
                }
                _phase.value = DictationPhase.Polishing(Draft(source), translating = false)
                return runCatching { model.run(recipe.polish(source, scene)) }
                    .map { Delivery(it, source, Delivery.Outcome.Polished) }
                    .getOrElse { Delivery(source, source, Delivery.Outcome.PolishUnavailable) }
            }
        }
    }
}
