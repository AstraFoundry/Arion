package si.astrafoundry.arion.core

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CorrectionRulesTest {
    @Test fun collapsesWhitespace() = assertEquals("hello world", CorrectionRules.apply("  hello   world \n"))
    @Test fun joinsCjk() = assertEquals("今天天气很好。", CorrectionRules.apply("今天 天气 很好 。"))
    @Test fun keepsLatinInChinese() = assertEquals("打开 Xcode 项目", CorrectionRules.apply("打开 Xcode 项目"))
    @Test fun dropsSpaceBeforePunctuation() = assertEquals("Hello, world!", CorrectionRules.apply("Hello , world !"))
}

class RewriteRecipeTest {
    private val recipe = RewriteRecipe(File(System.getProperty("arion.recipe")).readText())

    @Test fun readsTheDesktopRecipe() = assertEquals("contextual-editing-v1", recipe.version)

    @Test fun routesByScript() {
        assertEquals(RewriteRecipe.Route.Han, RewriteRecipe.route("今天开会"))
        assertEquals(RewriteRecipe.Route.Latin, RewriteRecipe.route("meet today"))
        assertEquals(RewriteRecipe.Route.Mixed, RewriteRecipe.route("打开 Xcode"))
    }

    @Test fun hanSourceUsesChineseInstructions() {
        val request = recipe.polish("明天三点开会", Scene.Chat)
        assertTrue(request.instructions.startsWith("# Role\n\n你是 Arion"))
        assertTrue(request.instructions.contains("原文："))
        assertEquals("<source>明天三点开会</source>", request.prompt)
        assertEquals(256, request.maximumTokens)
    }

    @Test fun listOmitsProsePolicy() {
        assertTrue(recipe.polish("buy milk", Scene.Chat).instructions.contains("keep ordinary narration in natural prose"))
        assertFalse(recipe.polish("buy milk", Scene.List).instructions.contains("keep ordinary narration in natural prose"))
    }

    @Test fun searchIsWrittenAsChat() = assertEquals(recipe.polish("weather", Scene.Chat), recipe.polish("weather", Scene.Search))

    @Test fun everyTargetTranslates() {
        for (target in TranslationTarget.entries) {
            val request = recipe.translate("你好", Scene.Chat, target)
            assertNotNull(target.tag, request)
            assertTrue(request!!.instructions.contains("<rules>"))
        }
    }

    @Test fun budgetScales() {
        assertEquals(2048, RewriteRecipe.budget("a".repeat(1000), 2))
        assertEquals(628, RewriteRecipe.budget("a".repeat(500), 1))
    }
}

class RewriteValidationTest {
    @Test fun acceptsPlainText() = assertTrue(RewriteValidation.accepts("Meet at 4:30."))

    @Test fun rejectsEmptyTruncatedAndMarkup() {
        assertFalse(RewriteValidation.accepts("  \n"))
        assertFalse(RewriteValidation.accepts("fine", truncated = true))
        assertFalse(RewriteValidation.accepts("<source>hi</source>"))
        assertFalse(RewriteValidation.accepts("[chat] hi"))
    }
}

class DictationTest {
    @Test fun draftJoins() {
        assertEquals("Hello wor", Draft("Hello ", "wor").text)
        assertTrue(Draft(" ", "").isEmpty)
    }

    @Test fun languagesFollowPreference() {
        assertEquals(listOf("zh-CN", "en-US"), DictationLanguage.defaults(listOf("zh-Hans-CN", "en-US")))
        assertEquals(listOf("en-US"), DictationLanguage.defaults(listOf("fr-FR")))
    }

    @Test fun joinsLatinWords() {
        assertEquals(" world", joinAfter("world", "hello"))
        assertEquals("世界", joinAfter("世界", "你好"))
        assertEquals("world", joinAfter("world", "hello "))
    }
}
