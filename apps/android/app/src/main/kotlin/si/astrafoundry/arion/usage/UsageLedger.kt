package si.astrafoundry.arion.usage

import android.content.Context
import si.astrafoundry.arion.core.Scene
import si.astrafoundry.arion.core.UsageDay
import si.astrafoundry.arion.core.UsageLog
import si.astrafoundry.arion.core.UsageReading
import java.io.File
import java.time.LocalDate

/**
 * The per-day counts the keyboard records and the Overview shows, as JSON in
 * the app's no-backup files. Numbers and scene ids only, never text (005).
 */
class UsageLedger(context: Context) : UsageReading {
    private val file = File(context.noBackupFilesDir, "usage.json")

    @Synchronized
    fun record(speechMs: Long, text: String, scene: Scene?, at: LocalDate = LocalDate.now()) {
        val updated = UsageLog.add(days(), at, speechMs, text, scene)
        val temp = File(file.parentFile, "${file.name}.tmp")
        temp.writeText(UsageLog.encode(updated))
        if (!temp.renameTo(file)) throw java.io.IOException("Could not replace ${file.name}")
    }

    @Synchronized
    override fun days(): List<UsageDay> = if (file.exists()) UsageLog.decode(file.readText()) else emptyList()

    override fun total(from: String?): UsageDay = UsageLog.total(days(), from)

    override fun days(end: String, count: Int): List<UsageDay> = UsageLog.days(days(), end, count)
}
