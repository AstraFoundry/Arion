package si.astrafoundry.arion.ime

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Backspace
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Language
import androidx.compose.material.icons.outlined.Translate
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.widthIn
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.boundsInRoot
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.platform.LocalViewConfiguration
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.onClick
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.withTimeoutOrNull
import si.astrafoundry.arion.core.DictationPhase
import si.astrafoundry.arion.design.Face
import si.astrafoundry.arion.design.FaceKind
import si.astrafoundry.arion.design.Layer
import si.astrafoundry.arion.design.Material
import si.astrafoundry.arion.design.Sunbeam
import si.astrafoundry.arion.design.VoiceMark

/** What the panel shows beside the microphone and in its text line. */
data class PanelState(
    val phase: DictationPhase,
    val languageMark: String?,
    val languageLabel: String,
    val translateLabel: String,
    val returnLabel: String,
    val notice: String?,
    val showSwitch: Boolean,
    val micEnabled: Boolean,
)

interface PanelActions {
    fun micTap()
    fun holdStart()
    fun holdEnd(target: HoldTarget)
    fun leading()
    fun translate()
    fun delete()
    fun space()
    fun moveCaret(steps: Int)
    fun enter()
    fun switchKeyboard()
    fun pickKeyboard()
}

enum class HoldTarget { Mic, Cancel, Translate }

/**
 * Arion's keyboard: one microphone, the keys a voice keyboard still needs, and
 * the text being recognized. It is a raised layer, so every key is raised.
 */
@Composable
fun KeyboardPanel(state: PanelState, actions: PanelActions, labels: PanelLabels) {
    val c = Sunbeam.colors
    var leadingBounds by remember { mutableStateOf(Rect.Zero) }
    var trailingBounds by remember { mutableStateOf(Rect.Zero) }
    var holdTarget by remember { mutableStateOf<HoldTarget?>(null) }
    val phase = state.phase
    val listening = phase is DictationPhase.Starting || phase is DictationPhase.Listening
    val working = phase is DictationPhase.Finishing || phase is DictationPhase.Polishing

    Layer(Material.Raised) {
        Column(
            Modifier
                .fillMaxWidth()
                .background(Brush.verticalGradient(listOf(c.panelFaceFrom, c.panelFaceTo)))
                .windowInsetsPadding(WindowInsets.navigationBars)
                .padding(horizontal = 6.dp, vertical = 6.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
          // On tablets and unfolded phones the keys keep a phone's reach.
          Column(Modifier.widthIn(max = 640.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            // The recognized text, or a short notice; drafts never reach the editor.
            val draft = when (phase) {
                is DictationPhase.Listening -> phase.draft
                is DictationPhase.Finishing -> phase.draft
                is DictationPhase.Polishing -> phase.draft
                else -> null
            }
            val line = buildAnnotatedString {
                if (draft != null) {
                    pushStyle(SpanStyle(color = c.foreground)); append(draft.finalized); pop()
                    pushStyle(SpanStyle(color = c.mutedForeground)); append(draft.volatile); pop()
                } else if (state.notice != null) {
                    pushStyle(SpanStyle(color = c.mutedForeground)); append(state.notice); pop()
                }
            }
            Box(Modifier.fillMaxWidth().height(64.dp).padding(horizontal = 14.dp, vertical = 4.dp), contentAlignment = Alignment.Center) {
                Text(
                    line,
                    style = if (draft != null) Sunbeam.type.body else Sunbeam.type.bodySmall,
                    textAlign = TextAlign.Center,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
                )
            }

            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(40.dp), modifier = Modifier.padding(top = 6.dp)) {
                Box(Modifier.size(56.dp, 44.dp).onGloballyPositioned { leadingBounds = it.boundsInRoot() }) {
                    when {
                        listening || working -> SideKey(labels.cancel, highlighted = holdTarget == HoldTarget.Cancel, onClick = actions::leading) { ink ->
                            Icon(Icons.Outlined.Close, null, tint = ink)
                        }
                        state.languageMark != null -> SideKey(state.languageLabel, onClick = actions::leading) { ink ->
                            Text(state.languageMark, style = Sunbeam.type.label, color = ink)
                        }
                    }
                }
                MicButton(
                    listening = listening,
                    working = working,
                    level = (phase as? DictationPhase.Listening)?.level ?: 0f,
                    enabled = state.micEnabled,
                    label = if (listening) labels.done else labels.dictate,
                    onTap = actions::micTap,
                    onHoldStart = { holdTarget = HoldTarget.Mic; actions.holdStart() },
                    onHoldMove = { point ->
                        val next = when {
                            leadingBounds.inflate(40f).contains(point) -> HoldTarget.Cancel
                            trailingBounds.inflate(40f).contains(point) -> HoldTarget.Translate
                            else -> HoldTarget.Mic
                        }
                        holdTarget = next
                    },
                    onHoldEnd = { actions.holdEnd(holdTarget ?: HoldTarget.Mic); holdTarget = null },
                )
                Box(Modifier.size(56.dp, 44.dp).onGloballyPositioned { trailingBounds = it.boundsInRoot() }) {
                    when {
                        listening -> SideKey(state.translateLabel, highlighted = holdTarget == HoldTarget.Translate, onClick = actions::translate) { ink ->
                            Icon(Icons.Outlined.Translate, null, tint = ink)
                        }
                        !working -> RepeatKey(labels.delete, onRepeat = actions::delete) { ink ->
                            Icon(Icons.AutoMirrored.Outlined.Backspace, null, tint = ink)
                        }
                    }
                }
            }

            Spacer(Modifier.height(14.dp))
            Row(Modifier.fillMaxWidth().height(44.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                val keyShape = RoundedCornerShape(10.dp)
                if (state.showSwitch) {
                    Face(FaceKind.Secondary, onClick = actions::switchKeyboard, shape = keyShape, contentDescription = labels.nextKeyboard,
                        modifier = Modifier.width(48.dp).height(44.dp).semantics { contentDescription = labels.nextKeyboard }
                            .pointerInput(Unit) { awaitEachGesture { awaitFirstDown(); val long = withTimeoutOrNull(viewConfiguration.longPressTimeoutMillis) { waitForUpOrCancellation() }; if (long == null) actions.pickKeyboard() } }) { ink ->
                        Icon(Icons.Outlined.Language, null, tint = ink)
                    }
                }
                SpaceBar(labels.space, actions, Modifier.weight(1f), keyShape)
                Face(FaceKind.Secondary, onClick = actions::enter, shape = keyShape, contentDescription = state.returnLabel, modifier = Modifier.width(92.dp).height(44.dp)) { ink ->
                    Text(state.returnLabel, style = Sunbeam.type.label, color = ink)
                }
            }
          }
        }
    }
}

/** Accessibility names the panel needs, resolved by the service. */
data class PanelLabels(val dictate: String, val done: String, val cancel: String, val delete: String, val space: String, val nextKeyboard: String)

@Composable
private fun SideKey(label: String, highlighted: Boolean = false, onClick: () -> Unit, content: @Composable (androidx.compose.ui.graphics.Color) -> Unit) {
    Face(FaceKind.Plain, onClick = onClick, highlighted = highlighted, contentDescription = label,
        modifier = Modifier.size(56.dp, 44.dp).semantics { contentDescription = label }) { ink -> content(ink) }
}

/** A key that repeats while held, as delete does on every keyboard. */
@Composable
private fun RepeatKey(label: String, onRepeat: () -> Unit, content: @Composable (androidx.compose.ui.graphics.Color) -> Unit) {
    val c = Sunbeam.colors
    var pressed by remember { mutableStateOf(false) }
    val face by animateColorAsState(if (pressed) c.flatFace else androidx.compose.ui.graphics.Color.Transparent, tween(120), label = "repeat")
    Box(
        Modifier.size(56.dp, 44.dp).clip(RoundedCornerShape(50)).background(face)
            .semantics { contentDescription = label; role = Role.Button; onClick { onRepeat(); true } }
            .pointerInput(Unit) {
                awaitEachGesture {
                    awaitFirstDown()
                    pressed = true
                    onRepeat()
                    var delay = 450L
                    while (withTimeoutOrNull(delay) { waitForUpOrCancellation() } == null) {
                        onRepeat()
                        delay = 80L
                    }
                    pressed = false
                }
            },
        contentAlignment = Alignment.Center,
    ) { content(c.foreground) }
}

/** Space inserts a space; dragging along it moves the caret. */
@Composable
private fun SpaceBar(label: String, actions: PanelActions, modifier: Modifier, shape: androidx.compose.ui.graphics.Shape) {
    Face(FaceKind.Secondary, onClick = actions::space, shape = shape, contentDescription = label,
        modifier = modifier.height(44.dp).pointerInput(Unit) {
            var carry = 0f
            detectHorizontalDragGestures(onDragStart = { carry = 0f }) { _, delta ->
                carry += delta
                val steps = (carry / 24f).toInt()
                if (steps != 0) {
                    actions.moveCaret(steps)
                    carry -= steps * 24f
                }
            }
        }) { ink ->
        Text(label, style = Sunbeam.type.label, color = ink)
    }
}

@Composable
private fun MicButton(
    listening: Boolean,
    working: Boolean,
    level: Float,
    enabled: Boolean,
    label: String,
    onTap: () -> Unit,
    onHoldStart: () -> Unit,
    onHoldMove: (androidx.compose.ui.geometry.Offset) -> Unit,
    onHoldEnd: () -> Unit,
) {
    // Each new level moves the bars to a new phase.
    var tick by remember { mutableIntStateOf(0) }
    androidx.compose.runtime.LaunchedEffect(level) { tick++ }
    val c = Sunbeam.colors
    var origin by remember { mutableStateOf(androidx.compose.ui.geometry.Offset.Zero) }
    val dark = listening || working
    val longPress = LocalViewConfiguration.current.longPressTimeoutMillis
    Box(
        Modifier
            .size(72.dp)
            .onGloballyPositioned { origin = it.positionInRoot() }
            .semantics { contentDescription = label; role = Role.Button; onClick { onTap(); true } }
            .alpha(if (enabled) 1f else 0.4f)
            .pointerInput(enabled, listening, working) {
                if (!enabled) return@pointerInput
                awaitEachGesture {
                    val down = awaitFirstDown()
                    val up = withTimeoutOrNull(minOf(longPress, 300L)) { waitForUpOrCancellation() }
                    if (up != null) {
                        onTap()
                        return@awaitEachGesture
                    }
                    if (listening || working) return@awaitEachGesture
                    onHoldStart()
                    while (true) {
                        val event = awaitPointerEvent()
                        val change = event.changes.firstOrNull { it.id == down.id } ?: break
                        onHoldMove(origin + change.position)
                        if (!change.pressed) break
                    }
                    onHoldEnd()
                }
            },
        contentAlignment = Alignment.Center,
    ) {
        if (dark) {
            // Ink on a light keyboard; on a dark one the raised dark face, which
            // stays visible where the substrate would disappear.
            Box(Modifier.size(72.dp).clip(CircleShape).background(if (c.dark) Brush.verticalGradient(listOf(c.raisedFaceFrom, c.raisedFaceTo)) else Brush.linearGradient(listOf(c.substrate, c.substrate))))
        } else {
            // The raised primary face, drawn without its own touch handling so
            // this button's gestures receive every touch.
            Box(
                Modifier.size(72.dp).shadow(2.dp, CircleShape, ambientColor = c.raisedPrimaryEdge, spotColor = c.raisedPrimaryEdge)
                    .clip(CircleShape).background(Brush.verticalGradient(listOf(c.raisedPrimaryFrom, c.raisedPrimaryTo)))
                    .border(androidx.compose.ui.unit.Dp.Hairline, c.raisedPrimaryEdge, CircleShape),
            )
        }
        VoiceMark(
            color = if (dark) c.primary else c.primaryForeground,
            level = if (listening) maxOf(level, 0.08f) else 0f,
            tick = tick,
            modifier = Modifier.size(36.dp, 30.dp).alpha(if (working) 0.5f else 1f),
        )
    }
}
