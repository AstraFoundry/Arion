package si.astrafoundry.arion.design

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp
import androidx.compose.foundation.clickable

// Sunbeam on Compose. Tokens come from SunbeamTokens.kt, generated from the
// Sunbeam package the desktop pins. Text uses the system sans at Sunbeam's
// sizes and weights; headlines use the system serif in place of Fraunces.

fun SunbeamArgb.color(): Color = Color(value.toInt())

/** Sunbeam's semantic colors resolved for one appearance. */
@Immutable
data class SunbeamColors(
    val dark: Boolean,
    val background: Color,
    val foreground: Color,
    val card: Color,
    val primary: Color,
    val primaryForeground: Color,
    val secondary: Color,
    val secondaryForeground: Color,
    val mutedForeground: Color,
    val border: Color,
    val faint: Color,
    val danger: Color,
    val flatFace: Color,
    val flatFacePressed: Color,
    val raisedFaceFrom: Color,
    val raisedFaceTo: Color,
    val raisedFacePressed: Color,
    val raisedEdge: Color,
    val raisedHighlight: Color,
    val raisedPrimaryFrom: Color,
    val raisedPrimaryTo: Color,
    val raisedPrimaryEdge: Color,
    val panelFaceFrom: Color,
    val panelFaceTo: Color,
    val substrate: Color,
    val substrateHi: Color,
)

private fun colors(dark: Boolean): SunbeamColors {
    fun SunbeamColorToken.pick() = (if (dark) this.dark else light).color()
    val c = SunbeamTokens.Color
    return SunbeamColors(
        dark = dark,
        background = c.background.pick(),
        foreground = c.foreground.pick(),
        card = c.card.pick(),
        primary = c.primary.pick(),
        primaryForeground = c.primaryForeground.pick(),
        secondary = c.secondary.pick(),
        secondaryForeground = c.secondaryForeground.pick(),
        mutedForeground = c.mutedForeground.pick(),
        border = c.border.pick(),
        faint = c.faint.pick(),
        danger = c.danger.pick(),
        flatFace = c.flatFace.pick(),
        flatFacePressed = c.flatFacePressed.pick(),
        raisedFaceFrom = c.raisedFaceFrom.pick(),
        raisedFaceTo = c.raisedFaceTo.pick(),
        raisedFacePressed = c.raisedFacePressed.pick(),
        raisedEdge = c.raisedEdge.pick(),
        raisedHighlight = c.raisedHighlight.pick(),
        raisedPrimaryFrom = c.raisedPrimaryFrom.pick(),
        raisedPrimaryTo = c.raisedPrimaryTo.pick(),
        raisedPrimaryEdge = c.raisedPrimaryEdge.pick(),
        panelFaceFrom = c.panelFaceFrom.pick(),
        panelFaceTo = c.panelFaceTo.pick(),
        substrate = c.substrate.pick(),
        substrateHi = c.substrateHi.pick(),
    )
}

/** Sunbeam's text styles. */
@Immutable
data class SunbeamType(
    val headline: TextStyle = TextStyle(fontFamily = FontFamily.Serif, fontSize = 34.sp, lineHeight = 40.sp, letterSpacing = (-0.02).em),
    val title: TextStyle = TextStyle(fontSize = 24.sp, fontWeight = FontWeight.Medium, lineHeight = 31.sp, letterSpacing = (-0.01).em),
    val body: TextStyle = TextStyle(fontSize = 16.sp, lineHeight = 25.sp),
    val bodySmall: TextStyle = TextStyle(fontSize = 14.sp, lineHeight = 21.sp),
    val label: TextStyle = TextStyle(fontSize = 14.sp, fontWeight = FontWeight.Medium, lineHeight = 21.sp),
    val caption: TextStyle = TextStyle(fontSize = 13.sp, lineHeight = 18.sp),
)

private val LocalColors = staticCompositionLocalOf { colors(false) }
private val LocalType = staticCompositionLocalOf { SunbeamType() }

/** The surface material of a layer: flat on the page, raised on a floating layer such as the keyboard. */
enum class Material { Flat, Raised }

private val LocalMaterial = staticCompositionLocalOf { Material.Flat }

object Sunbeam {
    val colors: SunbeamColors @Composable get() = LocalColors.current
    val type: SunbeamType @Composable get() = LocalType.current
    val material: Material @Composable get() = LocalMaterial.current
}

private fun materialScheme(c: SunbeamColors): ColorScheme {
    val base = if (c.dark) darkColorScheme() else lightColorScheme()
    // Sunbeam checks and switches in ink; yellow is reserved for the one primary action.
    return base.copy(
        primary = c.foreground,
        onPrimary = c.background,
        primaryContainer = c.secondary,
        onPrimaryContainer = c.foreground,
        secondary = c.secondaryForeground,
        background = c.background,
        onBackground = c.foreground,
        surface = c.background,
        onSurface = c.foreground,
        surfaceVariant = c.secondary,
        onSurfaceVariant = c.mutedForeground,
        surfaceContainer = c.card,
        surfaceContainerHigh = c.card,
        surfaceContainerHighest = c.secondary,
        outline = c.border,
        outlineVariant = c.border,
        error = c.danger,
    )
}

@Composable
fun SunbeamTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val palette = remember(dark) { colors(dark) }
    val type = remember { SunbeamType() }
    CompositionLocalProvider(LocalColors provides palette, LocalType provides type) {
        MaterialTheme(
            colorScheme = materialScheme(palette),
            typography = Typography(bodyLarge = type.body, bodyMedium = type.bodySmall, labelLarge = type.label, titleLarge = type.title),
            content = content,
        )
    }
}

/** Builds content in a layer of the given material. */
@Composable
fun Layer(material: Material, content: @Composable () -> Unit) {
    CompositionLocalProvider(LocalMaterial provides material, content = content)
}

enum class FaceKind { Primary, Secondary, Plain }

/**
 * A control drawn with a Sunbeam face, taking its material from its layer:
 * one translucent fill when flat; gradient, hairline edge and soft shadow when
 * raised.
 */
@Composable
fun Face(
    kind: FaceKind,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    shape: Shape = RoundedCornerShape(50),
    enabled: Boolean = true,
    highlighted: Boolean = false,
    contentDescription: String? = null,
    interaction: MutableInteractionSource = remember { MutableInteractionSource() },
    content: @Composable BoxScope.(ink: Color) -> Unit,
) {
    val c = Sunbeam.colors
    val raised = Sunbeam.material == Material.Raised
    val pressed by interaction.collectIsPressedAsState()
    val down = pressed || highlighted
    val scale by animateFloatAsState(if (down) 0.97f else 1f, tween(120), label = "press")
    val fill: Brush = when (kind) {
        FaceKind.Primary -> if (raised) Brush.verticalGradient(listOf(c.raisedPrimaryFrom, c.raisedPrimaryTo)) else Brush.linearGradient(listOf(c.primary, c.primary))
        FaceKind.Secondary -> when {
            raised && down -> Brush.linearGradient(listOf(c.raisedFacePressed, c.raisedFacePressed))
            raised -> Brush.verticalGradient(listOf(c.raisedFaceFrom, c.raisedFaceTo))
            down -> Brush.linearGradient(listOf(c.flatFacePressed, c.flatFacePressed))
            else -> Brush.linearGradient(listOf(c.flatFace, c.flatFace))
        }
        FaceKind.Plain -> {
            val face by animateColorAsState(if (down) c.flatFace else Color.Transparent, tween(120), label = "plain")
            Brush.linearGradient(listOf(face, face))
        }
    }
    val ink = if (kind == FaceKind.Primary) c.primaryForeground else c.foreground
    var m = modifier.scale(scale)
    if (raised && kind != FaceKind.Plain) {
        m = m.shadow(if (down) 0.5.dp else 2.dp, shape, ambientColor = c.raisedEdge, spotColor = c.raisedEdge)
    }
    m = m.clip(shape).background(fill)
    if (raised && kind != FaceKind.Plain) {
        m = m.border(Dp.Hairline, if (kind == FaceKind.Primary) c.raisedPrimaryEdge else c.raisedEdge, shape)
    }
    Box(
        m.clickable(interactionSource = interaction, indication = null, enabled = enabled, role = Role.Button, onClickLabel = contentDescription, onClick = onClick)
            .defaultMinSize(minWidth = 44.dp, minHeight = 44.dp),
        contentAlignment = Alignment.Center,
    ) {
        content(if (enabled) ink else ink.copy(alpha = 0.4f))
    }
}
