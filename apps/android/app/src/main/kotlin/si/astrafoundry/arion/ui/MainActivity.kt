package si.astrafoundry.arion.ui

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.provider.Settings
import android.view.inputmethod.InputMethodManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.launch
import si.astrafoundry.arion.core.DictationLanguage
import si.astrafoundry.arion.core.TranslationTarget
import si.astrafoundry.arion.design.Face
import si.astrafoundry.arion.design.FaceKind
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.Sunbeam
import si.astrafoundry.arion.design.SunbeamTheme
import si.astrafoundry.arion.polish.ModelAvailability
import si.astrafoundry.arion.polish.OnDeviceModel
import si.astrafoundry.arion.storage.History
import si.astrafoundry.arion.storage.HistoryEntry
import si.astrafoundry.arion.storage.Preferences
import si.astrafoundry.arion.storage.Vocabulary

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            SunbeamTheme {
                // Wide screens keep a readable column, as on a phone.
                Box(Modifier.fillMaxSize().background(Sunbeam.colors.background), contentAlignment = androidx.compose.ui.Alignment.TopCenter) {
                    Box(Modifier.widthIn(max = 600.dp)) { App() }
                }
            }
        }
    }
}

private enum class Screen { Onboarding, Home, Languages, Target, Vocabulary, History }

/** Whether Arion is among the enabled keyboards, and whether it is the current one. */
private data class KeyboardStatus(val enabled: Boolean, val selected: Boolean)

private fun keyboardStatus(context: Context): KeyboardStatus {
    val manager = context.getSystemService(InputMethodManager::class.java)
    val ours = manager.enabledInputMethodList.firstOrNull { it.packageName == context.packageName }
    val current = Settings.Secure.getString(context.contentResolver, Settings.Secure.DEFAULT_INPUT_METHOD)
    return KeyboardStatus(ours != null, ours != null && ours.id == current)
}

@Composable
private fun App() {
    val context = LocalContext.current
    val preferences = remember { Preferences(context) }
    var screen by remember { mutableStateOf(if (preferences.onboardingCompleted) Screen.Home else Screen.Onboarding) }
    BackHandler(screen != Screen.Home && screen != Screen.Onboarding) { screen = Screen.Home }
    AnimatedContent(screen, transitionSpec = { fadeIn() togetherWith fadeOut() }, label = "screen") { current ->
        when (current) {
            Screen.Onboarding -> Onboarding(preferences) { preferences.onboardingCompleted = true; screen = Screen.Home }
            Screen.Home -> Home(preferences) { screen = it }
            Screen.Languages -> Detail(stringResource(R.string.settings_languages), { screen = Screen.Home }) { Languages(preferences) }
            Screen.Target -> Detail(stringResource(R.string.settings_translate_into), { screen = Screen.Home }) { Target(preferences) }
            Screen.Vocabulary -> Detail(stringResource(R.string.settings_vocabulary), { screen = Screen.Home }) { VocabularyList() }
            Screen.History -> Detail(stringResource(R.string.settings_history), { screen = Screen.Home }) { HistoryList(preferences) }
        }
    }
}

/** Re-reads system state each time the activity resumes, such as after returning from Settings. */
@Composable
private fun onResumeCount(): Int {
    var count by remember { mutableIntStateOf(0) }
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(lifecycle) { lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) { count++ } }
    return count
}

private enum class Step { Welcome, Microphone, Enable, Select, Practice }

@Composable
private fun Onboarding(preferences: Preferences, done: () -> Unit) {
    val context = LocalContext.current
    val resumes = onResumeCount()
    var step by remember { mutableStateOf(Step.Welcome) }
    val micGranted = remember(resumes, step) { context.checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED }
    val status = remember(resumes, step) { keyboardStatus(context) }
    val permission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) step = Step.Enable
    }
    // Steps already satisfied are skipped when the user comes back from Settings.
    LaunchedEffect(resumes, step) {
        if (step == Step.Enable && status.enabled) step = Step.Select
        if (step == Step.Select && status.selected) step = Step.Practice
    }
    Column(Modifier.fillMaxSize().safeDrawingPadding().imePadding().padding(horizontal = 24.dp)) {
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(top = 56.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            AnimatedContent(step, transitionSpec = { fadeIn() togetherWith fadeOut() }, label = "step") { current ->
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    when (current) {
                        Step.Welcome -> {
                            AppMark()
                            Spacer(Modifier.height(16.dp))
                            Heading(stringResource(R.string.onboarding_welcome_title), headline = true)
                            Body(stringResource(R.string.onboarding_welcome_body))
                        }
                        Step.Microphone -> {
                            Heading(stringResource(R.string.onboarding_microphone_title))
                            Body(stringResource(R.string.onboarding_microphone_body))
                        }
                        Step.Enable -> {
                            Heading(stringResource(R.string.android_onboarding_enable_title))
                            Body(stringResource(R.string.android_onboarding_enable_body))
                            Spacer(Modifier.height(8.dp))
                            Body(stringResource(R.string.android_onboarding_enable_note), muted = true)
                        }
                        Step.Select -> {
                            Heading(stringResource(R.string.android_onboarding_select_title))
                            Body(stringResource(R.string.android_onboarding_select_body))
                        }
                        Step.Practice -> {
                            Heading(stringResource(R.string.onboarding_try_title))
                            Body(stringResource(R.string.onboarding_try_body_android))
                            Spacer(Modifier.height(8.dp))
                            PracticeField(stringResource(R.string.onboarding_try_placeholder), autofocus = true)
                        }
                    }
                }
            }
        }
        val label = when (step) {
            Step.Welcome -> stringResource(R.string.onboarding_continue)
            Step.Microphone -> stringResource(R.string.onboarding_microphone_allow)
            Step.Enable -> stringResource(R.string.android_onboarding_enable_open)
            Step.Select -> stringResource(R.string.android_onboarding_select_open)
            Step.Practice -> stringResource(R.string.onboarding_finish)
        }
        Face(FaceKind.Primary, onClick = {
            when (step) {
                Step.Welcome -> step = if (micGranted) Step.Enable else Step.Microphone
                Step.Microphone -> permission.launch(Manifest.permission.RECORD_AUDIO)
                Step.Enable -> context.startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS))
                Step.Select -> context.getSystemService(InputMethodManager::class.java).showInputMethodPicker()
                Step.Practice -> done()
            }
        }, contentDescription = label, modifier = Modifier.fillMaxWidth().height(52.dp).padding(bottom = 0.dp)) { ink ->
            Text(label, style = Sunbeam.type.label, color = ink)
        }
        Spacer(Modifier.height(16.dp))
    }
}

@Composable
private fun PracticeField(placeholder: String, autofocus: Boolean = false, bordered: Boolean = true) {
    val c = Sunbeam.colors
    var text by remember { mutableStateOf("") }
    val focus = remember { FocusRequester() }
    LaunchedEffect(autofocus) { if (autofocus) focus.requestFocus() }
    val shape = RoundedCornerShape(18.dp)
    Box(
        Modifier.fillMaxWidth().height(140.dp)
            .then(if (bordered) Modifier.background(c.card, shape).border(androidx.compose.ui.unit.Dp.Hairline, c.border, shape) else Modifier)
            .padding(16.dp),
    ) {
        if (text.isEmpty()) Text(placeholder, style = Sunbeam.type.body, color = c.faint)
        BasicTextField(
            text, { text = it },
            textStyle = Sunbeam.type.body.copy(color = c.foreground),
            cursorBrush = SolidColor(c.foreground),
            modifier = Modifier.fillMaxSize().focusRequester(focus).semantics { contentDescription = placeholder },
        )
    }
}

@Composable
private fun Home(preferences: Preferences, open: (Screen) -> Unit) {
    val context = LocalContext.current
    val resumes = onResumeCount()
    val scope = rememberCoroutineScope()
    val model = remember { OnDeviceModel() }
    var availability by remember { mutableStateOf<ModelAvailability?>(null) }
    var polishing by remember { mutableStateOf(preferences.polishingEnabled) }
    val status = remember(resumes) { keyboardStatus(context) }
    val vocabularyCount = remember(resumes) { runCatching { Vocabulary(context).terms().size }.getOrDefault(0) }
    LaunchedEffect(resumes) { availability = model.availability() }
    Column(Modifier.fillMaxSize().safeDrawingPadding().verticalScroll(rememberScrollState())) {
        Text(stringResource(R.string.app_name), style = Sunbeam.type.headline, color = Sunbeam.colors.foreground,
            modifier = Modifier.padding(start = 32.dp, top = 24.dp, bottom = 8.dp))
        Section { PracticeField(stringResource(R.string.home_practice), bordered = false) }
        Section(stringResource(R.string.settings_section_dictation), stringResource(R.string.settings_translate_footer)) {
            SettingRow(stringResource(R.string.settings_languages), preferences.languages.joinToString(", ") { languageName(context, it) }, onClick = { open(Screen.Languages) })
            SettingRow(stringResource(R.string.settings_translate_into), targetName(context, preferences.translationTarget), divider = true, onClick = { open(Screen.Target) })
        }
        val footer = when (availability) {
            ModelAvailability.Available, null -> stringResource(R.string.settings_polish_footer)
            ModelAvailability.Downloadable, ModelAvailability.Downloading -> stringResource(R.string.settings_polish_android_downloading)
            ModelAvailability.Unavailable -> stringResource(R.string.settings_polish_android_unavailable)
        }
        Section(footer = footer) {
            SettingRow(stringResource(R.string.settings_polish), trailing = {
                // Without a model the switch shows off; the saved choice returns with one.
                Toggle(polishing && availability != ModelAvailability.Unavailable, enabled = availability != ModelAvailability.Unavailable) {
                    polishing = it
                    preferences.polishingEnabled = it
                    if (it && availability == ModelAvailability.Downloadable) scope.launch { runCatching { model.download() }; availability = model.availability() }
                }
            })
        }
        Section(stringResource(R.string.settings_section_memory)) {
            SettingRow(stringResource(R.string.settings_vocabulary), vocabularyCount.toString(), onClick = { open(Screen.Vocabulary) })
            SettingRow(stringResource(R.string.settings_history), divider = true, onClick = { open(Screen.History) })
        }
        Section(stringResource(R.string.settings_section_keyboard)) {
            val ready = status.enabled
            SettingRow(stringResource(R.string.settings_keyboard_setup), stringResource(if (ready) R.string.settings_keyboard_ready else R.string.settings_keyboard_set_up),
                valueEmphasis = !ready, onClick = { context.startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS)) })
            SettingRow(stringResource(R.string.settings_version), context.packageManager.getPackageInfo(context.packageName, 0).versionName, divider = true)
        }
        Spacer(Modifier.height(24.dp))
    }
}

private fun languageName(context: Context, tag: String) =
    context.getString(if (tag == DictationLanguage.CHINESE) R.string.language_name_chinese else R.string.language_name_english)

@Composable
private fun Detail(title: String, back: () -> Unit, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxSize().safeDrawingPadding().imePadding().verticalScroll(rememberScrollState())) {
        IconButton(onClick = back, modifier = Modifier.padding(start = 8.dp, top = 4.dp)) {
            Icon(Icons.AutoMirrored.Outlined.ArrowBack, stringResource(R.string.keyboard_cancel), tint = Sunbeam.colors.foreground)
        }
        Text(title, style = Sunbeam.type.title, color = Sunbeam.colors.foreground, modifier = Modifier.padding(start = 32.dp, bottom = 8.dp))
        content()
    }
}

@Composable
private fun Languages(preferences: Preferences) {
    val context = LocalContext.current
    var selected by remember { mutableStateOf(preferences.languages) }
    Section(footer = stringResource(R.string.settings_languages_footer)) {
        DictationLanguage.all.forEachIndexed { index, tag ->
            SettingRow(languageName(context, tag), divider = index > 0, onClick = {
                val next = if (tag in selected) (selected - tag).ifEmpty { selected } else selected + tag
                selected = next
                preferences.languages = next
            }, trailing = { Check(tag in selected) })
        }
    }
}

@Composable
private fun Target(preferences: Preferences) {
    val context = LocalContext.current
    var selected by remember { mutableStateOf(preferences.translationTarget) }
    Section {
        TranslationTarget.entries.forEachIndexed { index, target ->
            SettingRow(targetName(context, target), divider = index > 0, onClick = {
                selected = target
                preferences.translationTarget = target
            }, trailing = { Check(target == selected) })
        }
    }
}

@Composable
private fun VocabularyList() {
    val context = LocalContext.current
    val vocabulary = remember { Vocabulary(context) }
    var terms by remember { mutableStateOf(runCatching { vocabulary.terms() }.getOrDefault(emptyList())) }
    var draft by remember { mutableStateOf("") }
    val c = Sunbeam.colors
    Section(footer = stringResource(R.string.settings_vocabulary_footer)) {
        Box(Modifier.fillMaxWidth().padding(16.dp)) {
            if (draft.isEmpty()) Text(stringResource(R.string.settings_vocabulary_add), style = Sunbeam.type.body, color = c.faint)
            BasicTextField(draft, { draft = it }, singleLine = true,
                textStyle = Sunbeam.type.body.copy(color = c.foreground), cursorBrush = SolidColor(c.foreground),
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = {
                    if (runCatching { vocabulary.add(draft) }.getOrDefault(false)) terms = vocabulary.terms()
                    draft = ""
                }),
                modifier = Modifier.fillMaxWidth())
        }
    }
    if (terms.isNotEmpty()) {
        Section {
            terms.forEachIndexed { index, term ->
                SettingRow(term, divider = index > 0, trailing = {
                    IconButton(onClick = { runCatching { vocabulary.remove(term) }; terms = vocabulary.terms() }) {
                        Icon(Icons.Outlined.Close, stringResource(R.string.settings_history_delete), tint = c.mutedForeground)
                    }
                })
            }
        }
    }
}

@Composable
private fun HistoryList(preferences: Preferences) {
    val context = LocalContext.current
    val history = remember { History(context) }
    var enabled by remember { mutableStateOf(preferences.historyEnabled) }
    var entries by remember { mutableStateOf<List<HistoryEntry>>(runCatching { history.entries() }.getOrDefault(emptyList())) }
    val clipboard = context.getSystemService(android.content.ClipboardManager::class.java)
    Section(footer = stringResource(R.string.settings_history_footer)) {
        SettingRow(stringResource(R.string.settings_history_keep), trailing = {
            Toggle(enabled) {
                enabled = it
                preferences.historyEnabled = it
                if (!it) {
                    runCatching { history.erase() }
                    entries = emptyList()
                }
            }
        })
    }
    if (enabled) {
        Section {
            if (entries.isEmpty()) {
                Text(stringResource(R.string.settings_history_empty), style = Sunbeam.type.body, color = Sunbeam.colors.mutedForeground, modifier = Modifier.padding(16.dp))
            }
            entries.forEachIndexed { index, entry ->
                SettingRow(entry.text, divider = index > 0, onClick = {
                    clipboard.setPrimaryClip(android.content.ClipData.newPlainText(null, entry.text))
                }, trailing = {
                    IconButton(onClick = { runCatching { history.remove(entry.id) }; entries = history.entries() }) {
                        Icon(Icons.Outlined.Close, stringResource(R.string.settings_history_delete), tint = Sunbeam.colors.mutedForeground)
                    }
                })
            }
        }
    }
}
