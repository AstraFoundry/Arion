package si.astrafoundry.arion.storage

import android.content.Context
import java.io.File
import java.util.UUID
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.KSerializer
import si.astrafoundry.arion.core.DictationLanguage
import si.astrafoundry.arion.core.TranslationTarget

/** The user's dictation preferences. The app and the keyboard share one process. */
class Preferences(context: Context) {
    private val store = context.getSharedPreferences("arion", Context.MODE_PRIVATE)

    /** Polishing with the on-device model is on by default and never leaves the phone. */
    var polishingEnabled: Boolean
        get() = store.getBoolean("polishing.enabled", true)
        set(value) = store.edit().putBoolean("polishing.enabled", value).apply()

    /** Recognition languages the language key cycles through. */
    var languages: List<String>
        get() = store.getString("dictation.languages", null)?.split(',')?.filter { it in DictationLanguage.all }?.ifEmpty { null }
            ?: DictationLanguage.defaults(android.os.LocaleList.getDefault().toLanguageTags().split(','))
        set(value) = store.edit().putString("dictation.languages", value.joinToString(",")).apply()

    var language: String
        get() = store.getString("dictation.language", null)?.takeIf { it in languages } ?: languages.first()
        set(value) = store.edit().putString("dictation.language", value).apply()

    /** By default the other of the first two languages: English speakers translate into Chinese. */
    var translationTarget: TranslationTarget
        get() = TranslationTarget.of(store.getString("translation.target", null))
            ?: if (languages.first() == DictationLanguage.ENGLISH) TranslationTarget.ZhCN else TranslationTarget.En
        set(value) = store.edit().putString("translation.target", value.tag).apply()

    /** Keeping recent dictations is opt-in and stays on this phone. */
    var historyEnabled: Boolean
        get() = store.getBoolean("history.enabled", false)
        set(value) = store.edit().putBoolean("history.enabled", value).apply()

    var onboardingCompleted: Boolean
        get() = store.getBoolean("onboarding.completed", false)
        set(value) = store.edit().putBoolean("onboarding.completed", value).apply()
}

/** A JSON list in the app's private files. */
private class JsonList<T>(context: Context, name: String, private val serializer: KSerializer<T>) {
    private val file = File(context.filesDir, name)
    private val list = ListSerializer(serializer)

    fun load(): List<T> = if (file.exists()) Json.decodeFromString(list, file.readText()) else emptyList()

    fun save(values: List<T>) {
        val temp = File(file.parentFile, file.name + ".tmp")
        temp.writeText(Json.encodeToString(list, values))
        if (!temp.renameTo(file)) throw java.io.IOException("Could not replace ${file.name}")
    }

    fun delete() {
        if (file.exists() && !file.delete()) throw java.io.IOException("Could not delete ${file.name}")
    }
}

/** Names and terms the user wants recognized and kept as written. */
class Vocabulary(context: Context) {
    private val file = JsonList(context, "vocabulary.json", String.serializer())

    fun terms(): List<String> = file.load()

    /** Adds a term once, ignoring case; false for empty or duplicate terms. */
    fun add(term: String): Boolean {
        val trimmed = term.trim()
        val all = terms()
        if (trimmed.isEmpty() || all.size >= LIMIT || all.any { it.equals(trimmed, ignoreCase = true) }) return false
        file.save(listOf(trimmed) + all)
        return true
    }

    fun remove(term: String) = file.save(terms() - term)

    companion object { const val LIMIT = 200 }
}

@Serializable
data class HistoryEntry(val id: String = UUID.randomUUID().toString(), val time: Long = System.currentTimeMillis(), val text: String, val original: String? = null)

/** Recent dictations, recorded only when history is on; turning it off erases them. */
class History(context: Context) {
    private val file = JsonList(context, "history.json", HistoryEntry.serializer())

    fun entries(): List<HistoryEntry> = file.load()

    fun record(text: String, original: String) {
        val entry = HistoryEntry(text = text, original = original.takeIf { it != text })
        file.save((listOf(entry) + entries()).take(LIMIT))
    }

    fun remove(id: String) = file.save(entries().filter { it.id != id })

    fun erase() = file.delete()

    companion object { const val LIMIT = 100 }
}
