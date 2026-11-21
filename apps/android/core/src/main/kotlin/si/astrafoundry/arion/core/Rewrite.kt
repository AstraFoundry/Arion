package si.astrafoundry.arion.core

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * The conservative rules applied to every transcript before delivery: the
 * spacing rules of the desktop's `domain.Correct` (`arion-rules-v4`). They
 * collapse whitespace and drop recognizer spaces between CJK runes and
 * before closing punctuation, and never add words.
 */
object CorrectionRules {
    fun apply(raw: String): String = joinCjkSpacing(collapseWhitespace(raw))

    internal fun collapseWhitespace(s: String): String = s.trim().split(Regex("\\s+")).filter { it.isNotEmpty() }.joinToString(" ")

    private fun isCjkPunctuation(c: Int) = c in 0x3000..0x303F || c in 0xFF00..0xFFEF

    private fun isCjk(c: Int): Boolean {
        val han = c in 0x3400..0x9FFF || c in 0xF900..0xFAFF || c in 0x20000..0x3134F || c in 0x2E80..0x2FDF
        val kana = c in 0x3040..0x30FF || c in 0x31F0..0x31FF
        val hangul = c in 0xAC00..0xD7AF || c in 0x1100..0x11FF || c in 0x3130..0x318F
        return han || kana || hangul || isCjkPunctuation(c)
    }

    private fun isClosingPunctuation(c: Int) =
        ",.!?;:%)]}".codePoints().anyMatch { it == c } ||
            (isCjkPunctuation(c) && "「『（【《〈".codePoints().noneMatch { it == c })

    internal fun joinCjkSpacing(s: String): String {
        val points = s.codePoints().toArray()
        val out = StringBuilder()
        for ((i, c) in points.withIndex()) {
            if (c == ' '.code && i > 0 && i + 1 < points.size) {
                val prev = points[i - 1]
                val next = points[i + 1]
                if ((isCjk(prev) && isCjk(next)) || isClosingPunctuation(next) || isCjkPunctuation(prev)) continue
            }
            out.appendCodePoint(c)
        }
        return out.toString()
    }
}

/** Where the text goes, which guides layout but never content. */
enum class Scene(val id: String) { Chat("chat"), Work("work"), Email("email"), Prompt("prompt"), List("list"), Search("search") }

/** A translation target the shared recipe supports. */
enum class TranslationTarget(val tag: String) {
    En("en"), ZhCN("zh-CN"), ZhTW("zh-TW"), Ja("ja"), Ko("ko"), Es("es"), Fr("fr"), De("de"), Pt("pt"), It("it"), Ru("ru"), Ar("ar");

    companion object {
        fun of(tag: String?): TranslationTarget? = entries.firstOrNull { it.tag == tag }
    }
}

/** One on-device rewrite: trusted instructions and the untrusted dictation. */
data class RewriteRequest(val instructions: String, val prompt: String, val maximumTokens: Int)

/**
 * Builds polishing and translation requests from the desktop's formatting
 * recipe (`apps/desktop/infrastructure/llm/prompt/format.json`, bundled
 * unchanged), following `SelectedTemplates.Format` and `FormatTranslation`.
 */
class RewriteRecipe(json: String) {
    private val root: JsonObject = Json.parseToJsonElement(json).jsonObject

    private fun string(key: String) = root.getValue(key).jsonPrimitive.content
    private fun obj(key: String) = root.getValue(key).jsonObject
    private data class Example(val source: String, val output: String)

    private fun examples(element: kotlinx.serialization.json.JsonElement?) =
        element?.jsonArray?.map {
            Example(it.jsonObject.getValue("source").jsonPrimitive.content, it.jsonObject.getValue("output").jsonPrimitive.content)
        } ?: emptyList()

    val version: String get() = string("version")

    enum class Route { Han, Latin, Mixed }

    private fun formatScene(scene: Scene) = if (scene == Scene.Search) Scene.Chat.id else scene.id

    private fun sceneInstruction(scene: String, han: Boolean): String {
        val recipe = obj("format")[scene]?.jsonObject ?: obj("format").getValue("chat").jsonObject
        return recipe.getValue(if (han) "instructionHan" else "instruction").jsonPrimitive.content
    }

    fun polish(source: String, scene: Scene): RewriteRequest {
        val route = route(source)
        val han = route == Route.Han
        val sceneName = formatScene(scene)
        val sections = mutableListOf(
            string(if (han) "formatSystemHan" else "formatSystem"),
            string(if (han) "formatRulesHan" else "formatRules"),
            "# Scene\n\n" + sceneInstruction(sceneName, han),
        )
        if (sceneName != Scene.List.id) sections += string(if (han) "prosePolicyHan" else "prosePolicy")
        sections += obj("routeInstructions")[route.name.lowercase()]?.jsonPrimitive?.content ?: ""
        sections += "# Context\n\n" + string(if (han) "contextRulesHan" else "contextRules")
        val demonstrations = examples(root["editingExamples"])
            .filter { route == Route.Mixed || route(it.source) == route }
            .map { if (han) "原文：${it.source}\n输出：${it.output}" else "Input: ${it.source}\nOutput: ${it.output}" }
        sections += "# Examples\n\n" + demonstrations.joinToString("\n\n")
        sections += "# Output\n\n" + string(if (han) "outputRuleHan" else "outputRule")
        return RewriteRequest(sections.joinToString("\n\n"), wrap(source), budget(source, 1))
    }

    /**
     * Polishes and translates into the target. The desktop sends the scene's
     * demonstrations as chat turns; the on-device prompt carries them inside
     * its instructions in the same order.
     */
    fun translate(source: String, scene: Scene, target: TranslationTarget): RewriteRequest? {
        val translation = obj("translate")[target.tag]?.jsonObject ?: return null
        val sceneName = formatScene(scene)
        val sceneExamples = examples(translation["scenes"]?.jsonObject?.get(sceneName))
        if (sceneExamples.isEmpty()) return null
        val chinese = target == TranslationTarget.ZhCN
        val lines = translation.getValue("rules").jsonPrimitive.content.split("\n").toMutableList()
        lines += sceneInstruction(sceneName, chinese)
        if (sceneName != Scene.List.id) lines += string(if (chinese) "prosePolicyHan" else "prosePolicy")
        lines += translation.getValue("route").jsonPrimitive.content
        val groups = mutableListOf(sceneExamples, examples(translation["editingExamples"]))
        if (sceneName == Scene.List.id) groups.reverse()
        val demonstrations = groups.flatMapIndexed { index, group ->
            group.map {
                val output = if (sceneName == Scene.List.id && index == 0) "1. ${it.output}" else it.output
                "Input: ${wrap(it.source)}\nOutput: $output"
            }
        }
        val instructions = translation.getValue("system").jsonPrimitive.content +
            "\n\n<rules>\n" + lines.joinToString("\n") + "\n</rules>" +
            "\n\n# Context\n\n" + string(if (chinese) "contextRulesHan" else "contextRules") +
            "\n\n# Examples\n\n" + demonstrations.joinToString("\n\n")
        return RewriteRequest(instructions, wrap(source), budget(source, 2))
    }

    companion object {
        fun wrap(source: String) = "<source>$source</source>"

        fun budget(source: String, perCharacter: Int): Int =
            minOf(2048, maxOf(256, source.codePointCount(0, source.length) * perCharacter + 128))

        private fun isHan(c: Int) = c in 0x3400..0x4DBF || c in 0x4E00..0x9FFF || c in 0xF900..0xFAFF || c in 0x20000..0x3134F

        private fun isLatin(c: Int): Boolean = when {
            c in 0x41..0x5A || c in 0x61..0x7A -> true
            c in 0xC0..0x24F -> c != 0xD7 && c != 0xF7
            else -> c in 0x1E00..0x1EFF || c in 0xFF21..0xFF3A || c in 0xFF41..0xFF5A
        }

        /** Which script the source uses, as the desktop's `route`. */
        fun route(source: String): Route {
            val han = source.codePoints().anyMatch(::isHan)
            val latin = source.codePoints().anyMatch(::isLatin)
            return when {
                han && !latin -> Route.Han
                latin && !han -> Route.Latin
                else -> Route.Mixed
            }
        }
    }
}

/**
 * The delivery check every model proposal passes before insertion, as the
 * desktop's `domain.ValidateRewriteDelivery`.
 */
object RewriteValidation {
    private val markers: List<String> = listOf(
        "<think", "</think", "<source>", "</source>", "<|", "|>", "[terms", "\"transcript\"",
        "<context>", "</context>", "<screen>", "</screen>", "<before_cursor>", "</before_cursor>",
        "<after_cursor>", "</after_cursor>", "<history>", "</history>", "<dictation>", "</dictation>",
    ) + Scene.entries.map { "[${it.id}]" }

    fun accepts(candidate: String, truncated: Boolean = false): Boolean {
        if (truncated) return false
        val trimmed = candidate.trim()
        if (trimmed.isEmpty()) return false
        val lower = trimmed.lowercase()
        return markers.none { lower.contains(it) }
    }
}

/** Adds a space between Latin words, as a typing keyboard would. */
fun joinAfter(text: String, before: CharSequence?): String {
    val last = before?.lastOrNull() ?: return text
    val first = text.firstOrNull() ?: return text
    fun wordLike(c: Char) = (c.isLetter() && c.code < 0x2E80) || c.isDigit()
    return if ((wordLike(last) || last in ".,!?;:") && wordLike(first)) " $text" else text
}
