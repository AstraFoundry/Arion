package si.astrafoundry.arion.ui

import si.astrafoundry.arion.design.ArionIcon
import si.astrafoundry.arion.design.IconView

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
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.slideInHorizontally
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import si.astrafoundry.arion.account.AccountService
import si.astrafoundry.arion.cloud.CloudService
import si.astrafoundry.arion.core.AccountPhase
import si.astrafoundry.arion.core.AccountServing
import si.astrafoundry.arion.core.AccountState
import si.astrafoundry.arion.core.CloudProvider
import si.astrafoundry.arion.core.CloudServing
import si.astrafoundry.arion.core.CloudState
import si.astrafoundry.arion.usage.UsageLedger
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

private enum class Screen { Onboarding, Overview, Settings, Account, Polish, Cloud, CloudProvider, CloudModels, Languages, Target, Vocabulary, History }

/** The screen a detail page returns to. */
private fun Screen.parent(): Screen = when (this) {
    Screen.Cloud -> Screen.Polish
    Screen.CloudProvider, Screen.CloudModels -> Screen.Cloud
    Screen.Vocabulary -> Screen.Overview
    else -> Screen.Settings
}

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
    val account = remember { AccountService.get(context) }
    val cloud = remember { CloudService.get(context) }
    val accountState by account.state.collectAsState()
    val cloudState by cloud.state.collectAsState()
    var screen by rememberSaveable { mutableStateOf(if (preferences.onboardingCompleted) Screen.Overview else Screen.Onboarding) }
    var provider by rememberSaveable { mutableStateOf(CloudProvider.OpenAI) }
    var tab by rememberSaveable { mutableStateOf(Screen.Overview) }
    // Vocabulary opened from the Overview returns there; from Settings, to Settings.
    var vocabularyFrom by rememberSaveable { mutableStateOf(Screen.Overview) }
    LaunchedEffect(Unit) { account.restore() }
    val resumes = onResumeCount()
    LaunchedEffect(resumes) { account.reportUsage() }
    val top = screen == Screen.Overview || screen == Screen.Settings || screen == Screen.Onboarding
    BackHandler(!top) { screen = if (screen == Screen.Vocabulary) vocabularyFrom else screen.parent() }
    val back = { screen = if (screen == Screen.Vocabulary) vocabularyFrom else screen.parent() }
    Column(Modifier.fillMaxSize()) {
        Box(Modifier.weight(1f)) {
            AnimatedContent(screen, transitionSpec = { fadeIn() togetherWith fadeOut() }, label = "screen") { current ->
                when (current) {
                    Screen.Onboarding -> Onboarding(preferences, account, accountState, cloud, cloudState) {
                        preferences.onboardingCompleted = true
                        screen = Screen.Overview
                    }
                    Screen.Overview -> Column(Modifier.fillMaxSize().safeDrawingPadding()) {
                        val terms = remember(resumes) { runCatching { Vocabulary(context).terms() }.getOrDefault(emptyList()) }
                        Overview(remember { UsageLedger(context) }, resumes, terms) { vocabularyFrom = Screen.Overview; screen = Screen.Vocabulary }
                    }
                    Screen.Settings -> SettingsHome(preferences, accountState, cloudState) {
                        if (it == Screen.Vocabulary) vocabularyFrom = Screen.Settings
                        screen = it
                    }
                    Screen.Account -> Detail(stringResource(R.string.account_title), back) { AccountScreen(account, accountState) }
                    Screen.Polish -> Detail(stringResource(R.string.polish_title), back) {
                        val model = remember { OnDeviceModel() }
                        var availability by remember { mutableStateOf<ModelAvailability?>(null) }
                        val scope = rememberCoroutineScope()
                        LaunchedEffect(resumes) { availability = model.availability() }
                        PolishScreen(preferences, availability, cloudState, openCloud = { screen = Screen.Cloud }) {
                            scope.launch { runCatching { model.download() }; availability = model.availability() }
                        }
                    }
                    Screen.Cloud -> Detail(stringResource(R.string.cloud_title), back) {
                        CloudScreen(cloud, cloudState, openProvider = { provider = it; screen = Screen.CloudProvider }, openModels = { screen = Screen.CloudModels })
                    }
                    Screen.CloudProvider -> Detail(providerName(provider), back) { CloudProviderScreen(cloud, provider) { screen = Screen.Cloud } }
                    Screen.CloudModels -> Detail(stringResource(R.string.cloud_model), back) { CloudModelsScreen(cloud, cloudState) { screen = Screen.Cloud } }
                    Screen.Languages -> Detail(stringResource(R.string.settings_languages), back) { Languages(preferences) }
                    Screen.Target -> Detail(stringResource(R.string.settings_translate_into), back) { Target(preferences) }
                    Screen.Vocabulary -> Detail(stringResource(R.string.settings_vocabulary), back) { VocabularyList() }
                    Screen.History -> Detail(stringResource(R.string.settings_history), back) { HistoryList(preferences) }
                }
            }
        }
        if (screen == Screen.Overview || screen == Screen.Settings) {
            Tabs(screen) { tab = it; screen = it }
        }
    }
}

/** The two top-level pages, as a flat bar in Sunbeam ink. */
@Composable
private fun Tabs(selected: Screen, choose: (Screen) -> Unit) {
    val c = Sunbeam.colors
    NavigationBar(containerColor = c.background, tonalElevation = 0.dp) {
        listOf(
            Triple(Screen.Overview, ArionIcon.ChartLine, R.string.tab_overview),
            Triple(Screen.Settings, ArionIcon.Settings, R.string.tab_settings),
        ).forEach { (screen, icon, label) ->
            NavigationBarItem(
                selected = selected == screen, onClick = { choose(screen) },
                icon = { IconView(icon, 22.dp, if (selected == screen) Sunbeam.colors.foreground else Sunbeam.colors.mutedForeground) }, label = { Text(stringResource(label), style = Sunbeam.type.caption) },
                colors = NavigationBarItemDefaults.colors(
                    selectedIconColor = c.foreground, selectedTextColor = c.foreground, indicatorColor = c.secondary,
                    unselectedIconColor = c.mutedForeground, unselectedTextColor = c.mutedForeground,
                ),
            )
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

private enum class Step { Welcome, Microphone, Enable, Select, Polish, Practice }

@Composable
private fun Onboarding(
    preferences: Preferences,
    account: AccountServing,
    accountState: AccountState,
    cloud: CloudServing,
    cloudState: CloudState,
    done: () -> Unit,
) {
    val context = LocalContext.current
    val resumes = onResumeCount()
    var step by rememberSaveable { mutableStateOf(Step.Welcome) }
    var terms by rememberSaveable { mutableStateOf(true) }
    var shareUsage by rememberSaveable { mutableStateOf(false) }
    var polish by rememberSaveable { mutableStateOf(PolishChoice.current(preferences)) }
    var cloudSetup by rememberSaveable { mutableStateOf<String?>(null) }
    val model = remember { OnDeviceModel() }
    var availability by remember { mutableStateOf<ModelAvailability?>(null) }
    LaunchedEffect(resumes) { availability = model.availability() }
    val micGranted = remember(resumes, step) { context.checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED }
    val status = remember(resumes, step) { keyboardStatus(context) }
    val permission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) step = Step.Enable
    }
    // Steps already satisfied are skipped when the user comes back from Settings.
    LaunchedEffect(resumes, step) {
        if (step == Step.Enable && status.enabled) step = Step.Select
        if (step == Step.Select && status.selected) step = Step.Polish
    }
    val previous = Step.entries.getOrNull(step.ordinal - 1)
    BackHandler(previous != null || cloudSetup != null) {
        when {
            cloudSetup == "provider" -> cloudSetup = "list"
            cloudSetup != null -> cloudSetup = null
            previous != null -> step = previous
        }
    }
    // Choosing Cloud opens the provider pages inside the step; connecting one moves on.
    if (cloudSetup != null) {
        var provider by rememberSaveable { mutableStateOf(CloudProvider.OpenAI) }
        val finish = { cloudSetup = null; step = Step.Practice }
        Column(Modifier.fillMaxSize().safeDrawingPadding().imePadding()) {
            Row(Modifier.fillMaxWidth().padding(start = 8.dp, end = 16.dp, top = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                IconButton(onClick = { cloudSetup = if (cloudSetup == "provider") "list" else null }) {
                    IconView(ArionIcon.ChevronLeft, 24.dp, contentDescription = stringResource(R.string.onboarding_back))
                }
                Spacer(Modifier.weight(1f))
                Face(FaceKind.Plain, onClick = finish) { ink -> Text(stringResource(R.string.onboarding_skip_cloud), style = Sunbeam.type.label, color = ink, modifier = Modifier.padding(horizontal = 12.dp)) }
            }
            Column(Modifier.verticalScroll(rememberScrollState())) {
                if (cloudSetup == "provider") {
                    Text(providerName(provider), style = Sunbeam.type.title, color = Sunbeam.colors.foreground, modifier = Modifier.padding(start = 24.dp, bottom = 8.dp))
                    CloudProviderScreen(cloud, provider) { if (cloud.state.value.active?.model != null) finish() else cloudSetup = "list" }
                } else {
                    Text(stringResource(R.string.cloud_title), style = Sunbeam.type.title, color = Sunbeam.colors.foreground, modifier = Modifier.padding(start = 32.dp, bottom = 8.dp))
                    CloudScreen(cloud, cloudState, openProvider = { provider = it; cloudSetup = "provider" }, openModels = {}, connected = finish)
                }
            }
        }
        return
    }
    Column(Modifier.fillMaxSize().safeDrawingPadding().imePadding().padding(horizontal = 24.dp)) {
        Row(Modifier.fillMaxWidth().height(48.dp), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(40.dp), contentAlignment = Alignment.CenterStart) {
                if (previous != null) {
                    IconButton(onClick = { step = previous }, modifier = Modifier.offset(x = (-12).dp)) {
                        IconView(ArionIcon.ChevronLeft, 24.dp, contentDescription = stringResource(R.string.onboarding_back))
                    }
                }
            }
            StepProgress(step.ordinal, Step.entries.size, Modifier.weight(1f))
        }
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(top = 24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            AnimatedContent(step, transitionSpec = {
                val forward = targetState.ordinal > initialState.ordinal
                (slideInHorizontally { if (forward) it / 8 else -it / 8 } + fadeIn()) togetherWith fadeOut()
            }, label = "step") { current ->
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    when (current) {
                        Step.Welcome -> {
                            AppMark()
                            Spacer(Modifier.height(16.dp))
                            Heading(stringResource(R.string.onboarding_welcome_title), headline = true)
                            Body(stringResource(R.string.onboarding_welcome_body))
                            Spacer(Modifier.height(20.dp))
                            if (accountState.signedIn || accountState.phase == AccountPhase.Restoring) {
                                AccountRowCompact(accountState)
                            } else {
                                Text(stringResource(R.string.account_sign_in_title), style = Sunbeam.type.label, color = Sunbeam.colors.foreground)
                                if (accountState.phase == AccountPhase.SigningIn) {
                                    Body(stringResource(R.string.account_waiting), muted = true)
                                    OutlineButton(stringResource(R.string.account_cancel), null) { account.cancel() }
                                } else {
                                    SignInMethods(enabled = terms) { account.signIn(shareUsage) }
                                    Spacer(Modifier.height(4.dp))
                                    ConsentRow(stringResource(R.string.account_terms_accept), listOf(
                                        stringResource(R.string.account_terms) to SiteLinks.terms,
                                        stringResource(R.string.account_privacy) to SiteLinks.privacy,
                                    ), terms) { terms = it }
                                    ConsentRow(stringResource(R.string.account_usage_agree), listOf(
                                        stringResource(R.string.account_usage_data) to SiteLinks.usageData,
                                    ), shareUsage) { shareUsage = it }
                                }
                                accountState.failure?.let { Text(stringResource(failureText(it)), style = Sunbeam.type.caption, color = Sunbeam.colors.danger) }
                            }
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
                        Step.Polish -> {
                            Heading(stringResource(R.string.onboarding_polish_title))
                            Body(stringResource(R.string.onboarding_polish_body))
                            Spacer(Modifier.height(12.dp))
                            PolishChoices(polish, availability, cloudState) { polish = it }
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
        // Signed out, the welcome step continues without an account in a quiet button.
        val local = step == Step.Welcome && !accountState.signedIn && accountState.phase != AccountPhase.Restoring
        val label = when {
            local -> stringResource(R.string.account_continue_locally)
            step == Step.Welcome || step == Step.Polish -> stringResource(R.string.onboarding_continue)
            step == Step.Microphone -> stringResource(R.string.onboarding_microphone_allow)
            step == Step.Enable -> stringResource(R.string.android_onboarding_enable_open)
            step == Step.Select -> stringResource(R.string.android_onboarding_select_open)
            else -> stringResource(R.string.onboarding_finish)
        }
        Face(if (local) FaceKind.Plain else FaceKind.Primary, onClick = {
            when (step) {
                Step.Welcome -> step = if (micGranted) Step.Enable else Step.Microphone
                Step.Microphone -> permission.launch(Manifest.permission.RECORD_AUDIO)
                Step.Enable -> context.startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS))
                Step.Select -> context.getSystemService(InputMethodManager::class.java).showInputMethodPicker()
                Step.Polish -> {
                    polish.apply(preferences)
                    if (polish == PolishChoice.Device && availability == ModelAvailability.Downloadable) {
                        CoroutineScope(Dispatchers.Default).launch { runCatching { model.download() } }
                    }
                    if (polish == PolishChoice.Cloud && cloudState.active?.model == null) cloudSetup = "list" else step = Step.Practice
                }
                Step.Practice -> done()
            }
        }, contentDescription = label, modifier = Modifier.fillMaxWidth().height(52.dp)) { ink ->
            Text(label, style = Sunbeam.type.label, color = if (local) Sunbeam.colors.mutedForeground else ink)
        }
        Spacer(Modifier.height(16.dp))
    }
}

/** Segments for the steps of a flow; the current and earlier ones in ink. */
@Composable
private fun StepProgress(current: Int, total: Int, modifier: Modifier = Modifier) {
    val c = Sunbeam.colors
    val description = stringResource(R.string.onboarding_progress, "${current + 1}", "$total")
    Row(modifier.semantics { contentDescription = description }, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        repeat(total) { index ->
            val color by animateColorAsState(if (index <= current) c.foreground else c.border, label = "segment")
            Box(Modifier.weight(1f).height(4.dp).background(color, RoundedCornerShape(2.dp)))
        }
    }
}

/** The signed-in account on the welcome step. */
@Composable
private fun AccountRowCompact(state: AccountState) {
    val c = Sunbeam.colors
    Card(padding = 14.dp) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Avatar(state.profile, 40.dp)
            Spacer(Modifier.width(12.dp))
            Column {
                Text(state.profile?.let { it.name.ifEmpty { it.email } }.orEmpty(), style = Sunbeam.type.body, color = c.foreground)
                Text(stringResource(if (state.phase == AccountPhase.Restoring) R.string.account_restoring else R.string.account_signed_in),
                    style = Sunbeam.type.caption, color = c.mutedForeground)
            }
        }
    }
}

@Composable
fun PracticeField(placeholder: String, autofocus: Boolean = false, height: Dp = 140.dp, fill: Color? = null) {
    val c = Sunbeam.colors
    var text by remember { mutableStateOf("") }
    val focus = remember { FocusRequester() }
    LaunchedEffect(autofocus) { if (autofocus) focus.requestFocus() }
    val shape = RoundedCornerShape(18.dp)
    Box(
        Modifier.fillMaxWidth().height(height)
            .background(fill ?: c.card, shape).border(androidx.compose.ui.unit.Dp.Hairline, c.border, shape)
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
private fun SettingsHome(preferences: Preferences, accountState: AccountState, cloudState: CloudState, open: (Screen) -> Unit) {
    val context = LocalContext.current
    val resumes = onResumeCount()
    val status = remember(resumes) { keyboardStatus(context) }
    val vocabularyCount = remember(resumes) { runCatching { Vocabulary(context).terms().size }.getOrDefault(0) }
    val polish = remember(resumes, cloudState) { PolishChoice.current(preferences) }
    Column(Modifier.fillMaxSize().safeDrawingPadding().verticalScroll(rememberScrollState())) {
        PageTitle(stringResource(R.string.tab_settings))
        AccountRow(accountState) { open(Screen.Account) }
        Section(stringResource(R.string.settings_section_dictation), stringResource(R.string.settings_translate_footer)) {
            SettingRow(stringResource(R.string.settings_languages), preferences.languages.joinToString(" · ") { languageMark(context, it) }, icon = ArionIcon.Globe, onClick = { open(Screen.Languages) })
            SettingRow(stringResource(R.string.settings_translate_into), targetName(context, preferences.translationTarget), icon = ArionIcon.Languages, divider = true, onClick = { open(Screen.Target) })
        }
        Section {
            val value = if (polish == PolishChoice.Cloud) cloudSummary(cloudState) ?: stringResource(R.string.cloud_not_connected) else stringResource(polish.title)
            SettingRow(stringResource(R.string.settings_polish), value, icon = ArionIcon.Sparkles, onClick = { open(Screen.Polish) })
        }
        Section(stringResource(R.string.settings_section_memory)) {
            SettingRow(stringResource(R.string.settings_vocabulary), vocabularyCount.toString(), icon = ArionIcon.BookOpen, onClick = { open(Screen.Vocabulary) })
            SettingRow(stringResource(R.string.settings_history), icon = ArionIcon.History, divider = true, onClick = { open(Screen.History) })
        }
        Section(stringResource(R.string.settings_section_keyboard)) {
            val ready = status.enabled
            SettingRow(stringResource(R.string.settings_keyboard_setup), stringResource(if (ready) R.string.settings_keyboard_ready else R.string.settings_keyboard_set_up),
                icon = ArionIcon.Keyboard, valueEmphasis = !ready, onClick = { context.startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS)) })
            SettingRow(stringResource(R.string.settings_version), context.packageManager.getPackageInfo(context.packageName, 0).versionName, icon = ArionIcon.Info, divider = true)
        }
        Spacer(Modifier.height(24.dp))
    }
}

private fun languageMark(context: Context, tag: String) =
    context.getString(if (tag == DictationLanguage.CHINESE) R.string.language_mark_chinese else R.string.language_mark_english)

private fun languageName(context: Context, tag: String) =
    context.getString(if (tag == DictationLanguage.CHINESE) R.string.language_name_chinese else R.string.language_name_english)

@Composable
private fun Detail(title: String, back: () -> Unit, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxSize().safeDrawingPadding().imePadding().verticalScroll(rememberScrollState())) {
        IconButton(onClick = back, modifier = Modifier.padding(start = 8.dp, top = 4.dp)) {
            IconView(ArionIcon.ChevronLeft, 24.dp, contentDescription = stringResource(R.string.onboarding_back))
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
                        IconView(ArionIcon.X, 18.dp, c.mutedForeground, stringResource(R.string.settings_history_delete))
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
                        IconView(ArionIcon.X, 18.dp, Sunbeam.colors.mutedForeground, stringResource(R.string.settings_history_delete))
                    }
                })
            }
        }
    }
}
