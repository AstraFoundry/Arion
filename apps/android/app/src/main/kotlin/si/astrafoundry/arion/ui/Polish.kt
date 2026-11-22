package si.astrafoundry.arion.ui

import si.astrafoundry.arion.design.ArionIcon
import si.astrafoundry.arion.design.IconView

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import si.astrafoundry.arion.core.CloudException
import si.astrafoundry.arion.core.CloudFailure
import si.astrafoundry.arion.core.CloudProvider
import si.astrafoundry.arion.core.CloudServing
import si.astrafoundry.arion.core.CloudState
import si.astrafoundry.arion.core.PolishEngine
import si.astrafoundry.arion.design.Face
import si.astrafoundry.arion.design.FaceKind
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.Sunbeam
import si.astrafoundry.arion.polish.ModelAvailability
import si.astrafoundry.arion.storage.Preferences

/** The polishing choice of onboarding and Settings: off, this device or a cloud model. */
enum class PolishChoice(val title: Int) {
    Off(R.string.polish_off), Device(R.string.polish_device), Cloud(R.string.polish_cloud);

    fun apply(preferences: Preferences) {
        preferences.polishingEnabled = this != Off
        if (this != Off) preferences.polishEngine = if (this == Cloud) PolishEngine.Cloud else PolishEngine.Device
    }

    companion object {
        fun current(preferences: Preferences): PolishChoice = when {
            !preferences.polishingEnabled -> Off
            preferences.polishEngine == PolishEngine.Cloud -> Cloud
            else -> Device
        }
    }
}

@Composable
fun providerName(provider: CloudProvider): String = when (provider) {
    CloudProvider.OpenAI -> "OpenAI"
    CloudProvider.Anthropic -> "Anthropic"
    CloudProvider.Gemini -> "Google Gemini"
    CloudProvider.DeepSeek -> "DeepSeek"
    CloudProvider.OpenRouter -> "OpenRouter"
    CloudProvider.OpenAICompatible -> stringResource(R.string.cloud_openai_compatible)
    CloudProvider.AnthropicCompatible -> stringResource(R.string.cloud_anthropic_compatible)
}

/** The mark of a cloud provider; compatible endpoints use their vendor's. */
fun providerIcon(provider: CloudProvider): ArionIcon = when (provider) {
    CloudProvider.OpenAI, CloudProvider.OpenAICompatible -> ArionIcon.LobeOpenai
    CloudProvider.Anthropic, CloudProvider.AnthropicCompatible -> ArionIcon.LobeAnthropic
    CloudProvider.Gemini -> ArionIcon.LobeGeminiMono
    CloudProvider.DeepSeek -> ArionIcon.LobeDeepseek
    CloudProvider.OpenRouter -> ArionIcon.LobeOpenrouter
}

/** The provider and model in use, as one line. */
@Composable
fun cloudSummary(state: CloudState): String? =
    state.active?.let { listOfNotNull(providerName(it.provider), it.model).joinToString(" · ") }

@Composable
private fun choiceDetail(choice: PolishChoice, availability: ModelAvailability?, cloud: CloudState): String = when (choice) {
    PolishChoice.Off -> stringResource(R.string.polish_off_detail)
    PolishChoice.Cloud -> cloudSummary(cloud) ?: stringResource(R.string.polish_cloud_detail)
    PolishChoice.Device -> when (availability) {
        ModelAvailability.Available, null -> stringResource(R.string.polish_device_detail)
        ModelAvailability.Downloadable, ModelAvailability.Downloading -> stringResource(R.string.settings_polish_android_downloading)
        ModelAvailability.Unavailable -> stringResource(R.string.settings_polish_android_unavailable)
    }
}

/** The three options as radio rows on a card. */
@Composable
fun PolishChoices(selected: PolishChoice, availability: ModelAvailability?, cloud: CloudState, choose: (PolishChoice) -> Unit) {
    val c = Sunbeam.colors
    val shape = RoundedCornerShape(18.dp)
    Column(Modifier.fillMaxWidth().background(c.card, shape).border(Dp.Hairline, c.border, shape)) {
        PolishChoice.entries.forEachIndexed { index, choice ->
            if (index > 0) HorizontalDivider(thickness = Dp.Hairline, color = c.border)
            val enabled = choice != PolishChoice.Device || availability != ModelAvailability.Unavailable
            RadioRow(stringResource(choice.title), choiceDetail(choice, availability, cloud), choice == selected, enabled) { choose(choice) }
        }
    }
}

@Composable
fun RadioRow(title: String, detail: String?, selected: Boolean, enabled: Boolean = true, onClick: () -> Unit) {
    val c = Sunbeam.colors
    Row(
        Modifier.fillMaxWidth().alpha(if (enabled) 1f else 0.45f).clickable(enabled = enabled, role = Role.RadioButton, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, style = Sunbeam.type.body, color = c.foreground)
            if (detail != null) Text(detail, style = Sunbeam.type.caption, color = c.mutedForeground)
        }
        Spacer(Modifier.width(12.dp))
        IconView(if (selected) ArionIcon.CircleCheck else ArionIcon.Circle, 22.dp, if (selected) c.foreground else c.faint)
    }
}

/** Settings › Polish: the choice, and the cloud model in use while Cloud is chosen. */
@Composable
fun PolishScreen(preferences: Preferences, availability: ModelAvailability?, cloud: CloudState, openCloud: () -> Unit, download: () -> Unit) {
    var choice by remember { mutableStateOf(PolishChoice.current(preferences)) }
    Section {
        PolishChoice.entries.forEachIndexed { index, option ->
            if (index > 0) HorizontalDivider(Modifier.padding(start = 16.dp), thickness = Dp.Hairline, color = Sunbeam.colors.border)
            val enabled = option != PolishChoice.Device || availability != ModelAvailability.Unavailable
            RadioRow(stringResource(option.title), if (option == PolishChoice.Cloud) stringResource(R.string.polish_cloud_detail) else choiceDetail(option, availability, cloud),
                option == choice, enabled) {
                choice = option
                option.apply(preferences)
                if (option == PolishChoice.Device && availability == ModelAvailability.Downloadable) download()
                if (option == PolishChoice.Cloud && cloud.active == null) openCloud()
            }
        }
    }
    if (choice == PolishChoice.Cloud) {
        Section {
            SettingRow(stringResource(R.string.cloud_title), cloudSummary(cloud) ?: stringResource(R.string.cloud_not_connected),
                icon = cloud.active?.provider?.let(::providerIcon) ?: ArionIcon.Server, onClick = openCloud)
        }
    }
}

/** The cloud providers: the one in use with its model, then every provider that takes a key. */
@Composable
fun CloudScreen(cloud: CloudServing, state: CloudState, openProvider: (CloudProvider) -> Unit, openModels: () -> Unit, connected: (() -> Unit)? = null) {
    val c = Sunbeam.colors
    var failure by remember { mutableStateOf<CloudFailure?>(null) }
    val active = state.active
    if (active != null) {
        Section(stringResource(R.string.cloud_in_use), failure?.let { stringResource(cloudFailure(it)) }) {
            SettingRow(providerName(active.provider), active.baseUrl?.let { android.net.Uri.parse(it).host }, icon = providerIcon(active.provider), trailing = { Check(true) })
            SettingRow(stringResource(R.string.cloud_model), active.model ?: stringResource(R.string.cloud_model_placeholder), icon = ArionIcon.Cpu, divider = true, onClick = openModels)
            HorizontalDivider(Modifier.padding(start = 16.dp), thickness = Dp.Hairline, color = c.border)
            Row(Modifier.fillMaxWidth().height(52.dp).clickable(role = Role.Button) {
                failure = runCatching { cloud.disconnect() }.exceptionOrNull()?.let { (it as? CloudException)?.failure ?: CloudFailure.Store }
            }.padding(horizontal = 16.dp), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                Text(stringResource(R.string.cloud_disconnect), style = Sunbeam.type.body, color = c.danger)
            }
        }
    }
    Section(stringResource(R.string.cloud_api_keys)) {
        CloudProvider.entries.filter { it != active?.provider }.forEachIndexed { index, provider ->
            val saved = provider in state.saved
            SettingRow(providerName(provider), if (saved) stringResource(R.string.cloud_saved) else null, icon = providerIcon(provider), divider = index > 0, onClick = { openProvider(provider) },
                trailing = if (saved) ({
                    Face(FaceKind.Secondary, onClick = {
                        val error = runCatching { cloud.activate(provider) }.exceptionOrNull()
                        failure = error?.let { (it as? CloudException)?.failure ?: CloudFailure.Store }
                        if (error == null && cloud.state.value.active?.model != null) connected?.invoke()
                    }, modifier = Modifier.height(32.dp)) { ink ->
                        Text(stringResource(R.string.cloud_use), style = Sunbeam.type.label, color = ink, modifier = Modifier.padding(horizontal = 12.dp))
                    }
                }) else null)
        }
    }
}

/** One provider: its key (and base URL for a compatible endpoint) and Connect. */
@Composable
fun CloudProviderScreen(cloud: CloudServing, provider: CloudProvider, done: () -> Unit) {
    val c = Sunbeam.colors
    val scope = rememberCoroutineScope()
    var key by remember { mutableStateOf("") }
    var address by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var failure by remember { mutableStateOf<CloudFailure?>(null) }
    Column(Modifier.padding(horizontal = 24.dp, vertical = 8.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        if (provider.needsBaseUrl) {
            FormField(stringResource(R.string.cloud_base_url), stringResource(R.string.cloud_base_url_placeholder), address, secret = false) { address = it; failure = null }
        }
        FormField(stringResource(R.string.cloud_key), stringResource(R.string.cloud_key_placeholder), key, secret = true) { key = it; failure = null }
        Spacer(Modifier.height(8.dp))
        val label = stringResource(R.string.cloud_connect)
        Face(FaceKind.Primary, enabled = !busy && key.isNotBlank(), onClick = {
            busy = true
            scope.launch {
                failure = try {
                    cloud.connect(provider, key.trim(), address.takeIf { provider.needsBaseUrl })
                    null
                } catch (e: CloudException) {
                    e.failure
                }
                busy = false
                if (failure == null) done()
            }
        }, contentDescription = label, modifier = Modifier.fillMaxWidth().height(50.dp)) { ink ->
            if (busy) CircularProgressIndicator(color = ink, strokeWidth = 2.dp, modifier = Modifier.size(18.dp))
            else Text(label, style = Sunbeam.type.label, color = ink)
        }
        failure?.let { Text(stringResource(cloudFailure(it)), style = Sunbeam.type.caption, color = c.danger) }
        Text(stringResource(R.string.cloud_footer), style = Sunbeam.type.caption, color = c.mutedForeground)
    }
}

/** The provider's models, the chosen one checked. */
@Composable
fun CloudModelsScreen(cloud: CloudServing, state: CloudState, done: () -> Unit) {
    var models by remember { mutableStateOf<List<String>?>(null) }
    var failure by remember { mutableStateOf<CloudFailure?>(null) }
    var attempt by remember { mutableStateOf(0) }
    LaunchedEffect(attempt) {
        failure = null
        models = null
        try {
            models = cloud.models()
        } catch (e: CloudException) {
            failure = e.failure
        }
    }
    Section(footer = failure?.let { stringResource(cloudFailure(it)) }) {
        when {
            failure != null -> SettingRow(stringResource(R.string.cloud_retry), onClick = { attempt++ })
            models == null -> Box(Modifier.fillMaxWidth().height(52.dp), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = Sunbeam.colors.mutedForeground, strokeWidth = 2.dp, modifier = Modifier.size(18.dp))
            }
            else -> models!!.forEachIndexed { index, model ->
                SettingRow(model, divider = index > 0, onClick = { cloud.choose(model); done() }, trailing = { Check(model == state.active?.model) })
            }
        }
    }
}

@Composable
private fun FormField(label: String, placeholder: String, value: String, secret: Boolean, onChange: (String) -> Unit) {
    val c = Sunbeam.colors
    val shape = RoundedCornerShape(14.dp)
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(label, style = Sunbeam.type.label, color = c.secondaryForeground)
        Box(Modifier.fillMaxWidth().height(48.dp).background(c.card, shape).border(Dp.Hairline, c.border, shape).padding(horizontal = 14.dp), contentAlignment = Alignment.CenterStart) {
            if (value.isEmpty()) Text(placeholder, style = Sunbeam.type.body, color = c.faint)
            BasicTextField(
                value, onChange, singleLine = true,
                textStyle = Sunbeam.type.body.copy(color = c.foreground),
                cursorBrush = SolidColor(c.foreground),
                visualTransformation = if (secret) PasswordVisualTransformation() else VisualTransformation.None,
                keyboardOptions = KeyboardOptions(keyboardType = if (secret) KeyboardType.Password else KeyboardType.Uri, autoCorrectEnabled = false),
                modifier = Modifier.fillMaxWidth().semantics { contentDescription = label },
            )
        }
    }
}

fun cloudFailure(failure: CloudFailure): Int = when (failure) {
    CloudFailure.Rejected -> R.string.cloud_fail_rejected
    CloudFailure.Network -> R.string.cloud_fail_network
    CloudFailure.Models -> R.string.cloud_fail_models
    CloudFailure.Store -> R.string.cloud_fail_store
    CloudFailure.Address -> R.string.cloud_fail_address
}
