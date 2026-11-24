package si.astrafoundry.arion.core

/**
 * The dictations committed on one local calendar day: numbers and scene
 * names only, never text (005). Mirrors `domain.UsageDay`.
 */
data class UsageDay(
    /** The local day as yyyy-MM-dd. */
    val date: String,
    val sessions: Int = 0,
    val speechMs: Long = 0,
    val words: Int = 0,
    /** Words of polished dictations by scene id. */
    val scenes: Map<String, Int> = emptyMap(),
) {
    val polishedWords: Int get() = scenes.values.sum()

    /** Average speed, 0 before any speech. */
    val wordsPerMinute: Int
        get() = if (speechMs > 0) Math.round(words * 60000.0 / speechMs).toInt() else 0
}

/** Reads the per-day counts the keyboard records. */
interface UsageReading {
    /** Every day with a dictation, oldest first. */
    fun days(): List<UsageDay>
    /** The sum of the days from [from] (yyyy-MM-dd) on; null sums every day. */
    fun total(from: String?): UsageDay
    /** The [count] days ending on [end] (yyyy-MM-dd), oldest first, zero-filled. */
    fun days(end: String, count: Int): List<UsageDay>
}
