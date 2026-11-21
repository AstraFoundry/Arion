package si.astrafoundry.arion.design

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import kotlin.math.max
import kotlin.math.sin

/** Bar heights from the icon generator (`apps/desktop/cmd/arion-icon`), as fractions of the tallest bar. */
private val heights = floatArrayOf(230f, 420f, 560f, 360f, 250f).map { it / 560f }
private const val BAR = 64f / 560f
private const val GAP = 46f / 560f

/**
 * Arion's voice mark: the five rounded bars of the app icon. At rest it draws
 * the icon's proportions; given a level it follows the voice.
 *
 * @param level voice level in 0..1; zero keeps the mark's shape.
 * @param tick advances each update so the bars move on their own phases.
 */
@Composable
fun VoiceMark(color: Color, modifier: Modifier = Modifier, level: Float = 0f, tick: Int = 0) {
    val animated by animateFloatAsState(level.coerceIn(0f, 1f), tween(100), label = "level")
    Canvas(modifier) {
        val tallest = size.height
        val width = tallest * BAR
        val gap = tallest * GAP
        val total = width * heights.size + gap * (heights.size - 1)
        var x = (size.width - total) / 2
        heights.forEachIndexed { index, h ->
            var height = tallest * h
            if (animated > 0f) {
                val wobble = (sin(tick * 0.9f + index * 1.7f) + 1f) / 2f
                height = max(width, tallest * (0.25f + 0.75f * animated * (0.55f + 0.45f * wobble)))
            }
            drawRoundRect(color, Offset(x, (size.height - height) / 2), Size(width, height), CornerRadius(width / 2))
            x += width + gap
        }
    }
}
