package si.astrafoundry.arion.ui

import si.astrafoundry.arion.design.ArionIcon
import si.astrafoundry.arion.design.IconView

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.spring
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.compositeOver
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import si.astrafoundry.arion.core.Scene
import si.astrafoundry.arion.core.UsageDay
import si.astrafoundry.arion.core.UsageReading
import si.astrafoundry.arion.design.R
import si.astrafoundry.arion.design.Sunbeam
import java.text.NumberFormat
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.time.format.TextStyle as DateTextStyle
import java.util.Locale
import kotlin.math.ceil
import kotlin.math.min

/**
 * What the user has dictated, as the desktop's Overview: totals, words per
 * day, where polishing went and the latest vocabulary. Before the first
 * dictation, a first-run card with a field to try the keyboard in.
 */
@Composable
fun Overview(usage: UsageReading, refresh: Int, terms: List<String>, editVocabulary: () -> Unit) {
    val days = remember(refresh) { usage.days() }
    val total = remember(refresh) { usage.total(null) }
    var period by rememberSaveable { mutableStateOf(Period.Week) }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        PageTitle(stringResource(R.string.tab_overview), start = 0.dp)
        if (total.sessions == 0) {
            Rise(0, refresh) { FirstRun() }
            Rise(1, refresh) { VocabularyCard(terms, editVocabulary) }
        } else {
            val (speech, speechUnit) = duration(total.speechMs)
            val stats = listOf(
                Stat(ArionIcon.Clock, stringResource(R.string.overview_speech_time), speech, speechUnit),
                Stat(ArionIcon.Type, stringResource(R.string.overview_words), count(total.words), stringResource(R.string.unit_words)),
                Stat(ArionIcon.Sparkles, stringResource(R.string.overview_polished), count(total.polishedWords), stringResource(R.string.unit_words)),
                Stat(ArionIcon.Gauge, stringResource(R.string.overview_speed), count(total.wordsPerMinute), stringResource(R.string.unit_words_per_minute)),
            )
            stats.chunked(2).forEachIndexed { row, pair ->
                Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    pair.forEachIndexed { column, stat ->
                        Box(Modifier.weight(1f).fillMaxHeight()) { Rise(row * 2 + column, refresh) { StatCard(stat) } }
                    }
                }
            }
            Rise(4, refresh) { Activity(days) }
            Rise(5, refresh) { Scenes(usage, refresh, period) { period = it } }
            Rise(6, refresh) { VocabularyCard(terms, editVocabulary) }
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
fun PageTitle(text: String, start: Dp = 16.dp) {
    // A tab's title starts on the left edge of its cards.
    Text(text, style = Sunbeam.type.headline, color = Sunbeam.colors.foreground,
        modifier = Modifier.padding(start = start, top = 24.dp, bottom = 4.dp).semantics { heading() })
}

/** A flat Sunbeam card. */
@Composable
fun Card(modifier: Modifier = Modifier, padding: Dp = 16.dp, content: @Composable ColumnScope.() -> Unit) {
    val c = Sunbeam.colors
    val shape = RoundedCornerShape(18.dp)
    Column(modifier.fillMaxWidth().background(c.card, shape).border(Dp.Hairline, c.border, shape).padding(padding), content = content)
}

@Composable
private fun CardHeader(title: String, trailing: @Composable () -> Unit = {}) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(title, style = Sunbeam.type.label, color = Sunbeam.colors.foreground, modifier = Modifier.weight(1f).semantics { heading() })
        trailing()
    }
}

/** Cards rise into place one after another when the page is entered. */
@Composable
private fun Rise(order: Int, key: Any, content: @Composable () -> Unit) {
    val progress = remember(key) { Animatable(0f) }
    LaunchedEffect(key) {
        kotlinx.coroutines.delay(order * 45L)
        progress.animateTo(1f, spring(dampingRatio = 1f, stiffness = 300f))
    }
    val shift = with(LocalDensity.current) { 8.dp.toPx() }
    Box(Modifier.graphicsLayer { alpha = progress.value; translationY = (1 - progress.value) * shift }) { content() }
}

private data class Stat(val icon: ArionIcon, val label: String, val value: String, val unit: String)

@Composable
private fun StatCard(stat: Stat) {
    val c = Sunbeam.colors
    Card(Modifier.fillMaxHeight().semantics(mergeDescendants = true) {}) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconView(stat.icon, 14.dp, c.mutedForeground)
            Spacer(Modifier.width(6.dp))
            Text(stat.label, style = Sunbeam.type.caption, color = c.mutedForeground, maxLines = 1)
        }
        Spacer(Modifier.height(10.dp))
        Row(verticalAlignment = Alignment.Bottom) {
            Text(stat.value, style = TextStyle(fontSize = 28.sp, fontWeight = FontWeight.SemiBold, fontFeatureSettings = "tnum"), color = c.foreground, maxLines = 1)
            Spacer(Modifier.width(4.dp))
            Text(stat.unit, style = Sunbeam.type.caption, color = c.mutedForeground, maxLines = 1, modifier = Modifier.padding(bottom = 4.dp))
        }
    }
}

private const val MAX_WEEKS = 26

/** The busiest quarter of the visible days is level 4, as the desktop. */
internal fun level(words: Int, active: List<Int>): Int {
    if (words <= 0 || active.isEmpty()) return 0
    val rank = active.indexOfFirst { it > words }.let { if (it < 0) active.size else it }
    return min(4, maxOf(1, ceil(4.0 * rank / active.size).toInt()))
}

/** Words per day for recent weeks, a week per column with Monday on top. */
@Composable
private fun Activity(log: List<UsageDay>) {
    val c = Sunbeam.colors
    val today = LocalDate.now()
    val firstMonday = today.minusDays((today.dayOfWeek.value - 1).toLong()).minusWeeks((MAX_WEEKS - 1).toLong())
    val byDate = remember(log) { log.associateBy { it.date } }
    val days = remember(log, today) {
        generateSequence(firstMonday) { it.plusDays(1) }.takeWhile { !it.isAfter(today) }.map { date -> date to (byDate[date.toString()]?.words ?: 0) }.toList()
    }
    var selected by remember { mutableStateOf<Int?>(null) }
    // An empty well, then the accent growing to full strength and deepening to its ink.
    val palette = listOf(c.border, c.primary.copy(alpha = 0.3f).compositeOver(c.card), c.primary.copy(alpha = 0.62f).compositeOver(c.card), c.primary, c.accentForeground)
    val measurer = rememberTextMeasurer()
    val caption = Sunbeam.type.caption.copy(fontSize = 11.sp, color = c.mutedForeground)
    val locale = Locale.getDefault()
    val detail = selected?.let { index ->
        val (date, words) = days[index]
        stringResource(R.string.overview_day_words, date.format(DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM)), count(words))
    }
    val weekWords = days.takeLast(7).sumOf { it.second }
    val summary = "${stringResource(R.string.overview_range_week)}: ${count(weekWords)} ${stringResource(R.string.unit_words)}"
    Card {
        CardHeader(stringResource(R.string.overview_activity)) {
            if (detail != null) {
                Text(detail, style = Sunbeam.type.caption, color = c.mutedForeground)
            } else {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(3.dp)) {
                    Text(stringResource(R.string.overview_less), style = Sunbeam.type.caption, color = c.mutedForeground, modifier = Modifier.padding(end = 3.dp))
                    palette.forEach { Box(Modifier.size(10.dp).background(it, RoundedCornerShape(2.dp))) }
                    Text(stringResource(R.string.overview_more), style = Sunbeam.type.caption, color = c.mutedForeground, modifier = Modifier.padding(start = 3.dp))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
        BoxWithConstraints(Modifier.fillMaxWidth()) {
            val density = LocalDensity.current
            val widthPx = with(density) { maxWidth.toPx() }
            val dayColumn = with(density) { 22.dp.toPx() }
            val monthRow = with(density) { 18.dp.toPx() }
            val gap = with(density) { 3.dp.toPx() }
            val minCell = with(density) { 11.dp.toPx() }
            val maxPitch = with(density) { 18.dp.toPx() }
            val room = widthPx - dayColumn + gap
            val weeks = min(MAX_WEEKS, maxOf(1, (room / (minCell + gap)).toInt()))
            val pitch = min(room / weeks, maxPitch)
            val cell = pitch - gap
            val x0 = dayColumn + room - pitch * weeks
            val skip = (MAX_WEEKS - weeks) * 7
            val active = days.drop(skip).map { it.second }.filter { it > 0 }.sorted()
            val height = with(density) { (monthRow + 7 * pitch - gap).toDp() }
            Canvas(
                Modifier.fillMaxWidth().height(height)
                    .semantics { contentDescription = summary }
                    .pointerInput(weeks, pitch) {
                        detectTapGestures { point ->
                            val week = ((point.x - x0) / pitch).toInt()
                            val day = ((point.y - monthRow) / pitch).toInt()
                            val index = skip + week * 7 + day
                            selected = if (point.x >= x0 && point.y >= monthRow && week < weeks && day < 7 && index < days.size && selected != index) index else null
                        }
                    },
            ) {
                for (day in 0 until 7 step 2) {
                    val name = DayOfWeek.of(day + 1).getDisplayName(DateTextStyle.NARROW, locale)
                    // A weekday ends just before the grid, however wide the card.
                    val width = measurer.measure(name, caption).size.width
                    drawText(measurer, name, Offset(x0 - 6.dp.toPx() - width, monthRow + day * pitch + (cell - 13.sp.toPx()) / 2), caption)
                }
                var previous = -1
                var labelEnd = -1f
                for (week in 0 until weeks) {
                    val index = skip + week * 7
                    if (index >= days.size) break
                    val month = days[index].first.monthValue
                    if (month == previous) continue
                    previous = month
                    val text = days[index].first.month.getDisplayName(DateTextStyle.SHORT, locale)
                    val x = x0 + week * pitch
                    val width = measurer.measure(text, caption).size.width
                    if (x > labelEnd + 6 && x + width <= size.width) {
                        drawText(measurer, text, Offset(x, 0f), caption)
                        labelEnd = x + width
                    }
                }
                val radius = CornerRadius(min(3.dp.toPx(), cell / 4))
                for (week in 0 until weeks) {
                    for (day in 0 until 7) {
                        val index = skip + week * 7 + day
                        if (index >= days.size) continue
                        val topLeft = Offset(x0 + week * pitch, monthRow + day * pitch)
                        drawRoundRect(palette[level(days[index].second, active)], topLeft, Size(cell, cell), radius)
                        if (index == selected) drawRoundRect(c.foreground.copy(alpha = 0.55f), topLeft, Size(cell, cell), radius, style = Stroke(1.dp.toPx()))
                    }
                }
            }
        }
    }
}

enum class Period(val label: Int) {
    Today(R.string.overview_range_today), Week(R.string.overview_range_week), All(R.string.overview_range_all);

    fun start(today: LocalDate): String? = when (this) {
        Today -> today.toString()
        Week -> today.minusDays(6).toString()
        All -> null
    }
}

/** Where a period's words went: polished words by scene, then those left as recognized. */
@Composable
private fun Scenes(usage: UsageReading, refresh: Int, period: Period, choose: (Period) -> Unit) {
    val c = Sunbeam.colors
    val total = remember(refresh, period) { usage.total(period.start(LocalDate.now())) }
    var open by remember { mutableStateOf(false) }
    Card {
        CardHeader(stringResource(R.string.overview_scenes)) {
            Box {
                Row(
                    Modifier.padding(vertical = 2.dp).pointerInput(Unit) { detectTapGestures { open = true } },
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(stringResource(period.label), style = Sunbeam.type.caption, color = c.mutedForeground)
                    IconView(ArionIcon.ChevronsUpDown, 14.dp, c.mutedForeground)
                }
                DropdownMenu(open, { open = false }, containerColor = c.card) {
                    Period.entries.forEach { option ->
                        DropdownMenuItem({ Text(stringResource(option.label), style = Sunbeam.type.body, color = c.foreground) }, onClick = {
                            choose(option)
                            open = false
                        }, trailingIcon = { Check(option == period) })
                    }
                }
            }
        }
        Spacer(Modifier.height(14.dp))
        if (total.words == 0) {
            Text(stringResource(R.string.overview_scenes_empty), style = Sunbeam.type.bodySmall, color = c.mutedForeground)
            return@Card
        }
        val shares = total.scenes.entries.sortedWith(compareBy({ -it.value }, { it.key }))
            .mapNotNull { (key, words) -> Scene.entries.firstOrNull { it.id == key }?.let { Triple(sceneIcon(it), stringResource(sceneLabel(it)), words) } }
            .toMutableList()
        val plain = total.words - total.polishedWords
        if (plain > 0) shares += Triple(ArionIcon.Type, stringResource(R.string.overview_unpolished), plain)
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            shares.forEach { (icon, label, words) ->
                Column(Modifier.semantics(mergeDescendants = true) {}, verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        IconView(icon, 14.dp, c.mutedForeground)
                        Spacer(Modifier.width(6.dp))
                        Text(label, style = Sunbeam.type.bodySmall, color = c.foreground, maxLines = 1, modifier = Modifier.weight(1f))
                        Text(count(words), style = Sunbeam.type.caption.copy(fontFeatureSettings = "tnum"), color = c.mutedForeground)
                    }
                    Box(Modifier.fillMaxWidth().height(6.dp).background(c.border, RoundedCornerShape(3.dp))) {
                        Box(Modifier.fillMaxWidth(maxOf(words.toFloat() / total.words, 0.02f)).height(6.dp).background(c.primary, RoundedCornerShape(3.dp)))
                    }
                }
            }
        }
    }
}

fun sceneLabel(scene: Scene): Int = when (scene) {
    Scene.Chat -> R.string.scene_chat
    Scene.Work -> R.string.scene_work
    Scene.Email -> R.string.scene_email
    Scene.Prompt -> R.string.scene_prompt
    Scene.List -> R.string.scene_list
    Scene.Search -> R.string.scene_search
}

private fun sceneIcon(scene: Scene): ArionIcon = when (scene) {
    Scene.Chat -> ArionIcon.MessageCircle
    Scene.Work -> ArionIcon.Briefcase
    Scene.Email -> ArionIcon.Mail
    Scene.Prompt -> ArionIcon.Code
    Scene.List -> ArionIcon.List
    Scene.Search -> ArionIcon.Search
}

/** Stands in for the totals until the first dictation. */
@Composable
private fun FirstRun() {
    val c = Sunbeam.colors
    Card(padding = 20.dp) {
        Text(stringResource(R.string.overview_first_title), style = Sunbeam.type.title, color = c.foreground)
        Text(stringResource(R.string.overview_first_prompt), style = Sunbeam.type.title, color = c.mutedForeground)
        Spacer(Modifier.height(20.dp))
        PracticeField(stringResource(R.string.overview_first_hint), height = 112.dp, fill = c.background)
    }
}

@Composable
private fun VocabularyCard(terms: List<String>, edit: () -> Unit) {
    val c = Sunbeam.colors
    Card {
        CardHeader(stringResource(R.string.overview_vocabulary)) {
            Text(stringResource(R.string.overview_vocabulary_edit), style = Sunbeam.type.label, color = c.foreground,
                modifier = Modifier.pointerInput(Unit) { detectTapGestures { edit() } }.padding(4.dp))
        }
        Spacer(Modifier.height(10.dp))
        if (terms.isEmpty()) {
            Text(stringResource(R.string.overview_vocabulary_empty_title), style = Sunbeam.type.bodySmall, color = c.mutedForeground)
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                terms.takeLast(5).reversed().forEach { Text(it, style = Sunbeam.type.body, color = c.foreground, maxLines = 1) }
            }
        }
    }
}

fun count(value: Int): String = NumberFormat.getIntegerInstance().format(value)

/** Seconds under a minute, minutes under an hour, then hours to one decimal. */
@Composable
private fun duration(milliseconds: Long): Pair<String, String> {
    val seconds = milliseconds / 1000.0
    return when {
        seconds < 60 -> count(Math.round(seconds).toInt()) to stringResource(R.string.unit_seconds)
        seconds < 3600 -> count(Math.round(seconds / 60).toInt()) to stringResource(R.string.unit_minutes)
        else -> NumberFormat.getNumberInstance().apply { maximumFractionDigits = 1 }.format(seconds / 3600) to stringResource(R.string.unit_hours)
    }
}
