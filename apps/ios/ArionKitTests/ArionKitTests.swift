@testable import ArionKit
import XCTest

final class CorrectionRulesTests: XCTestCase {
    func testCollapsesWhitespace() {
        XCTAssertEqual(CorrectionRules.apply("  hello   world \n"), "hello world")
    }

    func testJoinsRecognizerSpacesBetweenCJK() {
        XCTAssertEqual(CorrectionRules.apply("今天 天气 很好 。"), "今天天气很好。")
    }

    func testKeepsSpacesAroundLatinInChinese() {
        XCTAssertEqual(CorrectionRules.apply("打开 Xcode 项目"), "打开 Xcode 项目")
    }

    func testRemovesSpaceBeforeClosingPunctuation() {
        XCTAssertEqual(CorrectionRules.apply("Hello , world !"), "Hello, world!")
    }
}

final class RewriteRecipeTests: XCTestCase {
    private func recipe() throws -> RewriteRecipe { try RewriteRecipe.bundled() }

    func testBundlesTheDesktopRecipe() throws {
        XCTAssertEqual(try recipe().version, "contextual-editing-v1")
    }

    func testRoutesByScript() {
        XCTAssertEqual(RewriteRecipe.route("今天开会"), .han)
        XCTAssertEqual(RewriteRecipe.route("meet today"), .latin)
        XCTAssertEqual(RewriteRecipe.route("打开 Xcode"), .mixed)
    }

    func testHanSourceUsesChineseInstructions() throws {
        let request = try recipe().polish("明天三点开会", scene: .chat)
        XCTAssertTrue(request.instructions.hasPrefix("# Role\n\n你是 Arion"))
        XCTAssertTrue(request.instructions.contains("原文："))
        XCTAssertEqual(request.prompt, "<source>明天三点开会</source>")
        XCTAssertEqual(request.maximumTokens, 256)
    }

    func testListSceneOmitsProsePolicy() throws {
        let chat = try recipe().polish("buy milk and eggs", scene: .chat)
        let list = try recipe().polish("buy milk and eggs", scene: .list)
        XCTAssertTrue(chat.instructions.contains("keep ordinary narration in natural prose"))
        XCTAssertFalse(list.instructions.contains("keep ordinary narration in natural prose"))
    }

    func testSearchIsWrittenAsChat() throws {
        XCTAssertEqual(try recipe().polish("weather", scene: .search), try recipe().polish("weather", scene: .chat))
    }

    func testEveryTargetTranslates() throws {
        let recipe = try recipe()
        for target in TranslationTarget.allCases {
            let request = try XCTUnwrap(recipe.translate("你好", scene: .chat, target: target), target.rawValue)
            XCTAssertTrue(request.instructions.contains("<rules>"))
            XCTAssertEqual(request.maximumTokens, 256)
        }
    }

    func testBudgetScales() {
        XCTAssertEqual(RewriteRecipe.budget(String(repeating: "a", count: 1000), perCharacter: 2), 2048)
        XCTAssertEqual(RewriteRecipe.budget(String(repeating: "a", count: 500), perCharacter: 1), 628)
    }
}

final class RewriteValidationTests: XCTestCase {
    func testAcceptsPlainText() {
        XCTAssertTrue(RewriteValidation.accepts("Meet at 4:30."))
    }

    func testRejectsEmptyTruncatedAndMarkup() {
        XCTAssertFalse(RewriteValidation.accepts("  \n"))
        XCTAssertFalse(RewriteValidation.accepts("fine", truncated: true))
        XCTAssertFalse(RewriteValidation.accepts("<source>hi</source>"))
        XCTAssertFalse(RewriteValidation.accepts("[chat] hi"))
        XCTAssertFalse(RewriteValidation.accepts("<think>ok"))
    }
}

final class DraftTests: XCTestCase {
    func testDraftJoinsFinalizedAndVolatile() {
        let draft = Draft(finalized: "Hello ", volatile: "wor")
        XCTAssertEqual(draft.text, "Hello wor")
        XCTAssertFalse(draft.isEmpty)
        XCTAssertTrue(Draft(finalized: " ", volatile: "").isEmpty)
    }

    func testGenerationsOrder() {
        let first = DictationGeneration(0)
        XCTAssertLessThan(first, first.next())
    }
}

final class LanguageTests: XCTestCase {
    func testDefaultsFollowPreferredLanguages() {
        XCTAssertEqual(DictationLanguage.defaults(preferred: ["zh-Hans-CN", "en-US"]), ["zh-CN", "en-US"])
        XCTAssertEqual(DictationLanguage.defaults(preferred: ["fr-FR"]), ["en-US"])
    }
}

final class WebMicrophoneTests: XCTestCase {
    @MainActor
    func testDecodesLittleEndianPCM() throws {
        let samples: [Int16] = [0, 1000, -1000, 32767]
        let data = samples.withUnsafeBytes { Data($0) }
        let buffer = try XCTUnwrap(WebMicrophone.buffer(from: data))
        XCTAssertEqual(buffer.frameLength, 4)
        XCTAssertEqual(buffer.int16ChannelData?[0][3], 32767)
        XCTAssertNil(WebMicrophone.buffer(from: Data()))
    }
}

final class TextJoiningTests: XCTestCase {
    func testSpacesBetweenLatinWords() {
        XCTAssertEqual(TextJoining.joining("world", after: "hello"), " world")
        XCTAssertEqual(TextJoining.joining("world", after: "hello "), "world")
        XCTAssertEqual(TextJoining.joining("世界", after: "你好"), "世界")
        XCTAssertEqual(TextJoining.joining("Next", after: "Done."), " Next")
        XCTAssertEqual(TextJoining.joining("hi", after: nil), "hi")
    }
}

@MainActor
final class ModelDownloadTests: XCTestCase {
    func testDownloadPhasesLeaveTheKeyboardIdle() {
        XCTAssertFalse(DictationPhase.downloading(0.4).isActive)
        XCTAssertFalse(DictationPhase.downloaded.isActive)
    }

    func testKeyboardShowsTheSystemDownloadWithItsProgress() {
        let keyboard = VoiceKeyboardView(frame: .zero)
        keyboard.languageName = "中文"
        keyboard.render(.downloading(0.42))
        let line = keyboard.draftLabel.text ?? ""
        XCTAssertTrue(line.contains("中文"), line)
        XCTAssertTrue(line.contains("42"), line)
        keyboard.render(.downloading(0.9))
        XCTAssertTrue((keyboard.draftLabel.text ?? "").contains("90"))
    }

    func testDownloadFailureIsNamed() {
        XCTAssertEqual(VoiceKeyboardView.message(.modelDownloadFailed), L10n.failureDownload)
        XCTAssertNotEqual(VoiceKeyboardView.message(.modelDownloadFailed), L10n.failureInstalling)
    }
}
