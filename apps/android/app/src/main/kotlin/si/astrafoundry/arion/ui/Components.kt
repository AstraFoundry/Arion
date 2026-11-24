package si.astrafoundry.arion.ui

import si.astrafoundry.arion.design.ArionIcon
import si.astrafoundry.arion.design.IconView

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import si.astrafoundry.arion.core.TranslationTarget
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.Sunbeam
import si.astrafoundry.arion.design.VoiceMark

fun targetName(context: Context, target: TranslationTarget): String = context.getString(
    when (target) {
        TranslationTarget.En -> R.string.target_en
        TranslationTarget.ZhCN -> R.string.target_zh_cn
        TranslationTarget.ZhTW -> R.string.target_zh_tw
        TranslationTarget.Ja -> R.string.target_ja
        TranslationTarget.Ko -> R.string.target_ko
        TranslationTarget.Es -> R.string.target_es
        TranslationTarget.Fr -> R.string.target_fr
        TranslationTarget.De -> R.string.target_de
        TranslationTarget.Pt -> R.string.target_pt
        TranslationTarget.It -> R.string.target_it
        TranslationTarget.Ru -> R.string.target_ru
        TranslationTarget.Ar -> R.string.target_ar
    },
)

/** A section of settings: an optional header, a flat card of rows, an optional footer. */
@Composable
fun Section(title: String? = null, footer: String? = null, content: @Composable ColumnScope.() -> Unit) {
    val c = Sunbeam.colors
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp)) {
        if (title != null) {
            Text(title, style = Sunbeam.type.caption, color = c.mutedForeground,
                modifier = Modifier.padding(start = 16.dp, bottom = 6.dp).semantics { heading() })
        }
        Column(
            Modifier.fillMaxWidth().background(c.card, RoundedCornerShape(18.dp))
                .border(Dp.Hairline, c.border, RoundedCornerShape(18.dp)),
            content = content,
        )
        if (footer != null) {
            Text(footer, style = Sunbeam.type.caption, color = c.mutedForeground, modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 6.dp))
        }
    }
}

/** One settings row: a label, an optional value, and a trailing control. */
@Composable
fun SettingRow(
    label: String,
    value: String? = null,
    icon: ArionIcon? = null,
    divider: Boolean = false,
    valueEmphasis: Boolean = false,
    onClick: (() -> Unit)? = null,
    trailing: @Composable (() -> Unit)? = null,
) {
    val c = Sunbeam.colors
    // A row with an icon starts its divider at the text, as the system lists do.
    if (divider) HorizontalDivider(Modifier.padding(start = if (icon != null) 48.dp else 16.dp), thickness = Dp.Hairline, color = c.border)
    Row(
        Modifier.fillMaxWidth().heightIn(min = 52.dp)
            .then(if (onClick != null) Modifier.clickable(role = Role.Button, onClick = onClick) else Modifier)
            .padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) {
            IconView(icon, 18.dp)
            Spacer(Modifier.width(14.dp))
        }
        Text(label, style = Sunbeam.type.body, color = c.foreground, modifier = Modifier.weight(1f).padding(vertical = 12.dp))
        if (value != null) {
            Text(value, style = Sunbeam.type.body, color = if (valueEmphasis) c.foreground else c.mutedForeground, maxLines = 1)
        }
        when {
            trailing != null -> { Spacer(Modifier.width(8.dp)); trailing() }
            onClick != null -> IconView(ArionIcon.ChevronRight, 16.dp, c.faint, modifier = Modifier.padding(start = 4.dp))
        }
    }
}

@Composable
fun Check(visible: Boolean) {
    if (visible) IconView(ArionIcon.Check, 18.dp)
}

@Composable
fun Toggle(checked: Boolean, enabled: Boolean = true, onChange: (Boolean) -> Unit) {
    val c = Sunbeam.colors
    Switch(checked, onChange, enabled = enabled, colors = SwitchDefaults.colors(
        checkedTrackColor = c.foreground, checkedThumbColor = c.background,
        uncheckedTrackColor = c.secondary, uncheckedThumbColor = c.card, uncheckedBorderColor = c.border,
    ))
}

/** The app icon in miniature: the voice mark on Sunbeam yellow. */
@Composable
fun AppMark() {
    val c = Sunbeam.colors
    Box(Modifier.size(64.dp).background(c.primary, RoundedCornerShape(16.dp)), contentAlignment = Alignment.Center) {
        VoiceMark(c.primaryForeground, Modifier.size(38.dp, 34.dp))
    }
}

@Composable
fun Heading(text: String, headline: Boolean = false) {
    Text(text, style = if (headline) Sunbeam.type.headline else Sunbeam.type.title, color = Sunbeam.colors.foreground,
        modifier = Modifier.semantics { heading() })
}

@Composable
fun Body(text: String, muted: Boolean = false) {
    Text(text, style = if (muted) Sunbeam.type.bodySmall else Sunbeam.type.body,
        color = if (muted) Sunbeam.colors.mutedForeground else Sunbeam.colors.secondaryForeground)
}

@Composable
fun Steps(vararg steps: String) {
    val c = Sunbeam.colors
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        steps.forEachIndexed { index, text ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(28.dp).background(c.secondary, RoundedCornerShape(14.dp)), contentAlignment = Alignment.Center) {
                    Text("${index + 1}", style = Sunbeam.type.label, color = c.foreground)
                }
                Spacer(Modifier.width(14.dp))
                Text(text, style = Sunbeam.type.body, color = c.foreground)
            }
        }
    }
}
