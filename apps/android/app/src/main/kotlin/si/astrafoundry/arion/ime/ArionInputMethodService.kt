package si.astrafoundry.arion.ime

import android.Manifest
import android.content.pm.PackageManager
import android.inputmethodservice.InputMethodService
import android.text.InputType
import android.view.HapticFeedbackConstants
import android.view.KeyEvent
import android.view.View
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputMethodManager
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.ComposeView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.LifecycleRegistry
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.setViewTreeLifecycleOwner
import androidx.savedstate.SavedStateRegistry
import androidx.savedstate.SavedStateRegistryController
import androidx.savedstate.SavedStateRegistryOwner
import androidx.savedstate.setViewTreeSavedStateRegistryOwner
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import si.astrafoundry.arion.core.Delivery
import si.astrafoundry.arion.core.DictationFailure
import si.astrafoundry.arion.core.DictationIntent
import si.astrafoundry.arion.core.DictationLanguage
import si.astrafoundry.arion.core.DictationPhase
import si.astrafoundry.arion.core.Scene
import si.astrafoundry.arion.core.joinAfter
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.SunbeamTheme
import si.astrafoundry.arion.storage.History
import si.astrafoundry.arion.storage.Preferences
import si.astrafoundry.arion.storage.Vocabulary
import si.astrafoundry.arion.ui.targetName

/**
 * Arion's voice keyboard. Drafts stay in the panel; the editor receives one
 * commit when the dictation ends. Password fields get no microphone, and
 * fields that ask for no personalized learning are never kept in history.
 */
class ArionInputMethodService : InputMethodService(), LifecycleOwner, SavedStateRegistryOwner {
    private val lifecycleRegistry = LifecycleRegistry(this)
    private val savedState = SavedStateRegistryController.create(this)
    override val lifecycle: Lifecycle get() = lifecycleRegistry
    override val savedStateRegistry: SavedStateRegistry get() = savedState.savedStateRegistry

    private lateinit var preferences: Preferences
    private lateinit var controller: DictationController
    private var notice by mutableStateOf<String?>(null)
    private var languageRevision by mutableStateOf(0)
    private var editor by mutableStateOf<EditorInfo?>(null)
    private var noticeJob: Job? = null
    private var root: View? = null

    override fun onCreate() {
        super.onCreate()
        savedState.performRestore(null)
        lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_CREATE)
        preferences = Preferences(this)
        controller = DictationController(this, lifecycleScope, preferences, ::insert)
    }

    override fun onCreateInputView(): View {
        window.window?.decorView?.let {
            it.setViewTreeLifecycleOwner(this)
            it.setViewTreeSavedStateRegistryOwner(this)
        }
        val labels = PanelLabels(
            dictate = getString(R.string.keyboard_dictate),
            done = getString(R.string.keyboard_done),
            cancel = getString(R.string.keyboard_cancel),
            delete = getString(R.string.keyboard_delete),
            space = getString(R.string.keyboard_space),
            nextKeyboard = getString(R.string.keyboard_next_keyboard),
        )
        return ComposeView(this).apply {
            setViewTreeLifecycleOwner(this@ArionInputMethodService)
            setViewTreeSavedStateRegistryOwner(this@ArionInputMethodService)
            setContent {
                val phase by controller.phase.collectAsState()
                languageRevision
                SunbeamTheme {
                    KeyboardPanel(panelState(phase), actions, labels)
                }
            }
            root = this
        }
    }

    override fun onStartInputView(info: EditorInfo?, restarting: Boolean) {
        super.onStartInputView(info, restarting)
        editor = info
        lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_RESUME)
    }

    override fun onFinishInputView(finishingInput: Boolean) {
        // Leaving the field ends the dictation; nothing arrives later.
        controller.cancel()
        lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_PAUSE)
        super.onFinishInputView(finishingInput)
    }

    override fun onDestroy() {
        controller.cancel()
        lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_DESTROY)
        super.onDestroy()
    }

    private val isPasswordField: Boolean
        get() {
            val type = editor?.inputType ?: return false
            val variation = type and InputType.TYPE_MASK_VARIATION
            return when (type and InputType.TYPE_MASK_CLASS) {
                InputType.TYPE_CLASS_TEXT -> variation == InputType.TYPE_TEXT_VARIATION_PASSWORD ||
                    variation == InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD || variation == InputType.TYPE_TEXT_VARIATION_WEB_PASSWORD
                InputType.TYPE_CLASS_NUMBER -> variation == InputType.TYPE_NUMBER_VARIATION_PASSWORD
                else -> false
            }
        }

    /** The scene follows the field, as the desktop's scene rules use the target. */
    private val scene: Scene
        get() {
            val info = editor ?: return Scene.Chat
            val action = info.imeOptions and EditorInfo.IME_MASK_ACTION
            val variation = info.inputType and InputType.TYPE_MASK_VARIATION
            return when {
                action == EditorInfo.IME_ACTION_SEARCH -> Scene.Search
                variation == InputType.TYPE_TEXT_VARIATION_URI -> Scene.Search
                variation == InputType.TYPE_TEXT_VARIATION_EMAIL_SUBJECT -> Scene.Email
                else -> Scene.Chat
            }
        }

    private fun panelState(phase: DictationPhase): PanelState {
        val languages = preferences.languages
        val language = preferences.language
        val info = editor
        val action = info?.let { it.imeOptions and EditorInfo.IME_MASK_ACTION } ?: EditorInfo.IME_ACTION_NONE
        val noEnterAction = info != null && info.imeOptions and EditorInfo.IME_FLAG_NO_ENTER_ACTION != 0
        val returnLabel = getString(
            when {
                noEnterAction -> R.string.return_default
                action == EditorInfo.IME_ACTION_GO -> R.string.return_go
                action == EditorInfo.IME_ACTION_SEARCH -> R.string.return_search
                action == EditorInfo.IME_ACTION_SEND -> R.string.return_send
                action == EditorInfo.IME_ACTION_NEXT -> R.string.return_next
                action == EditorInfo.IME_ACTION_DONE -> R.string.return_done
                else -> R.string.return_default
            },
        )
        val failure = (phase as? DictationPhase.Failed)?.failure
        return PanelState(
            phase = phase,
            languageMark = if (languages.size > 1) getString(if (language == DictationLanguage.CHINESE) R.string.language_mark_chinese else R.string.language_mark_english) else null,
            languageLabel = getString(R.string.keyboard_language, getString(if (language == DictationLanguage.CHINESE) R.string.language_name_chinese else R.string.language_name_english)),
            translateLabel = getString(R.string.keyboard_translate, targetName(this, preferences.translationTarget)),
            returnLabel = returnLabel,
            notice = failure?.let(::message) ?: notice,
            showSwitch = shouldOfferSwitchingToNextInputMethod(),
            micEnabled = !isPasswordField,
        )
    }

    private fun message(failure: DictationFailure): String = getString(
        when (failure) {
            DictationFailure.MicrophoneDenied -> R.string.android_failure_microphone
            DictationFailure.RecognizerUnavailable -> R.string.android_failure_recognizer
            DictationFailure.LanguageUnavailable -> R.string.android_failure_network
            DictationFailure.RecognizerBusy -> R.string.android_failure_busy
            DictationFailure.CaptureEnded -> R.string.failure_capture
            DictationFailure.RecognitionFailed -> R.string.failure_recognition
            DictationFailure.NothingHeard -> R.string.failure_nothing_heard
        },
    )

    private fun begin() {
        if (checkSelfPermission(Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) {
            controller.cancel()
            showNotice(message(DictationFailure.MicrophoneDenied), autoClear = false)
            return
        }
        haptic()
        noticeJob?.cancel()
        notice = null
        controller.start(preferences.language, runCatching { Vocabulary(this).terms() }.getOrDefault(emptyList()), scene)
    }

    private fun insert(delivery: Delivery) {
        val connection = currentInputConnection ?: return
        val before = connection.getTextBeforeCursor(1, 0)
        connection.commitText(joinAfter(delivery.text, before), 1)
        val noLearning = editor?.let { it.imeOptions and EditorInfo.IME_FLAG_NO_PERSONALIZED_LEARNING != 0 } ?: false
        if (preferences.historyEnabled && !noLearning) {
            runCatching { History(this).record(delivery.text, delivery.source) }
        }
        when (delivery.outcome) {
            Delivery.Outcome.TranslationUnavailable -> showNotice(getString(R.string.notice_untranslated), autoClear = true)
            Delivery.Outcome.PolishUnavailable -> showNotice(getString(R.string.notice_unpolished), autoClear = true)
            else -> Unit
        }
    }

    private fun showNotice(text: String, autoClear: Boolean) {
        noticeJob?.cancel()
        notice = text
        if (autoClear) {
            noticeJob = lifecycleScope.launch {
                delay(2500)
                notice = null
            }
        }
    }

    private fun haptic() {
        root?.performHapticFeedback(HapticFeedbackConstants.KEYBOARD_TAP)
    }

    private val actions = object : PanelActions {
        override fun micTap() {
            when (controller.phase.value) {
                DictationPhase.Idle, is DictationPhase.Failed -> begin()
                DictationPhase.Starting, is DictationPhase.Listening -> { haptic(); controller.finish(DictationIntent.Dictate) }
                else -> Unit
            }
        }

        override fun holdStart() = begin()

        override fun holdEnd(target: HoldTarget) {
            haptic()
            when (target) {
                HoldTarget.Cancel -> controller.cancel()
                HoldTarget.Translate -> controller.finish(DictationIntent.Translate(preferences.translationTarget))
                HoldTarget.Mic -> controller.finish(DictationIntent.Dictate)
            }
        }

        override fun leading() {
            if (controller.phase.value.isActive) {
                controller.cancel()
                return
            }
            val languages = preferences.languages
            if (languages.size < 2) return
            preferences.language = languages[(languages.indexOf(preferences.language) + 1) % languages.size]
            languageRevision++
            haptic()
        }

        override fun translate() {
            haptic()
            controller.finish(DictationIntent.Translate(preferences.translationTarget))
        }

        override fun delete() {
            controller.dismissFailure()
            val connection = currentInputConnection ?: return
            haptic()
            if (connection.getSelectedText(0).isNullOrEmpty()) connection.deleteSurroundingTextInCodePoints(1, 0) else connection.commitText("", 1)
        }

        override fun space() {
            haptic()
            currentInputConnection?.commitText(" ", 1)
        }

        override fun moveCaret(steps: Int) {
            val key = if (steps < 0) KeyEvent.KEYCODE_DPAD_LEFT else KeyEvent.KEYCODE_DPAD_RIGHT
            repeat(kotlin.math.abs(steps)) { sendDownUpKeyEvents(key) }
        }

        override fun enter() {
            haptic()
            val info = editor
            val action = info?.let { it.imeOptions and EditorInfo.IME_MASK_ACTION } ?: EditorInfo.IME_ACTION_NONE
            val noEnterAction = info != null && info.imeOptions and EditorInfo.IME_FLAG_NO_ENTER_ACTION != 0
            if (!noEnterAction && action != EditorInfo.IME_ACTION_NONE && action != EditorInfo.IME_ACTION_UNSPECIFIED) {
                currentInputConnection?.performEditorAction(action)
            } else {
                sendKeyChar('\n')
            }
        }

        override fun switchKeyboard() {
            controller.cancel()
            switchToPreviousInputMethod()
        }

        override fun pickKeyboard() {
            controller.cancel()
            getSystemService(InputMethodManager::class.java).showInputMethodPicker()
        }
    }
}
