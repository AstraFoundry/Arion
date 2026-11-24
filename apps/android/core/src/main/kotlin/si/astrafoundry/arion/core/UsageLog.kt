package si.astrafoundry.arion.core

import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import java.time.LocalDate

/**
 * Counts text as people count what they wrote, as `domain.CountWords`: each
 * Han, kana or Hangul character is one word, as are runs of other letters
 * and digits joined by apostrophes, hyphens or decimal points.
 */
fun countWords(text: String): Int {
    var words = 0
    var inWord = false
    val points = text.codePoints().toArray()
    fun cjk(c: Int): Boolean = Character.UnicodeScript.of(c).let {
        it == Character.UnicodeScript.HAN || it == Character.UnicodeScript.HIRAGANA ||
            it == Character.UnicodeScript.KATAKANA || it == Character.UnicodeScript.HANGUL
    } && Character.isLetter(c)
    fun mark(c: Int) = Character.getType(c).let {
        it == Character.NON_SPACING_MARK.toInt() || it == Character.COMBINING_SPACING_MARK.toInt() || it == Character.ENCLOSING_MARK.toInt()
    }
    for ((i, c) in points.withIndex()) {
        when {
            cjk(c) -> { words++; inWord = false }
            Character.isLetterOrDigit(c) || mark(c) -> { if (!inWord) words++; inWord = true }
            inWord && (c == '\''.code || c == '’'.code || c == '-'.code || c == '.'.code) && i + 1 < points.size &&
                Character.isLetterOrDigit(points[i + 1]) && !cjk(points[i + 1]) -> Unit // Joined: don't, e-mail, 3.5
            else -> inWord = false
        }
    }
    return words
}

@Serializable
private data class StoredDay(val date: String, val sessions: Int = 0, val speechMs: Long = 0, val words: Int = 0, val scenes: Map<String, Int> = emptyMap())

/** The per-day history, oldest first, one entry per day; numbers and scene ids only. */
object UsageLog {
    private val serializer = ListSerializer(StoredDay.serializer())
    private val json = Json { ignoreUnknownKeys = true }

    fun decode(text: String): List<UsageDay> =
        json.decodeFromString(serializer, text).map { UsageDay(it.date, it.sessions, it.speechMs, it.words, it.scenes) }

    fun encode(days: List<UsageDay>): String =
        json.encodeToString(serializer, days.map { StoredDay(it.date, it.sessions, it.speechMs, it.words, it.scenes) })

    /** The log with one committed dictation counted on its day. */
    fun add(log: List<UsageDay>, date: LocalDate, speechMs: Long, text: String, scene: Scene?): List<UsageDay> {
        val key = date.toString()
        val words = countWords(text)
        val day = log.firstOrNull { it.date == key } ?: UsageDay(key)
        val scenes = if (scene == null) day.scenes else day.scenes + (scene.id to (day.scenes[scene.id] ?: 0) + words)
        val updated = day.copy(sessions = day.sessions + 1, speechMs = day.speechMs + maxOf(speechMs, 0), words = day.words + words, scenes = scenes)
        return (log.filter { it.date != key } + updated).sortedBy { it.date }
    }

    fun total(log: List<UsageDay>, from: String?): UsageDay = log.filter { from == null || it.date >= from }
        .fold(UsageDay("")) { sum, day ->
            val scenes = (sum.scenes.keys + day.scenes.keys).associateWith { (sum.scenes[it] ?: 0) + (day.scenes[it] ?: 0) }
            sum.copy(sessions = sum.sessions + day.sessions, speechMs = sum.speechMs + day.speechMs, words = sum.words + day.words, scenes = scenes)
        }

    fun days(log: List<UsageDay>, end: String, count: Int): List<UsageDay> {
        val byDate = log.associateBy { it.date }
        val last = LocalDate.parse(end)
        return (count - 1 downTo 0).map { back -> last.minusDays(back.toLong()).toString().let { byDate[it] ?: UsageDay(it) } }
    }
}
