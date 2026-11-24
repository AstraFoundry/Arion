@testable import ArionKit
import AVFAudio
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

final class KeyLayoutTests: XCTestCase {
    func testQwertyRowsByLayout() {
        let pinyin = KeyLayout.qwerty(.pinyin)
        XCTAssertEqual(pinyin.map(\.count), [10, 9, 9])
        XCTAssertEqual(pinyin[2].first?.key, .shift, "Shift sits on the left, as on the system keyboards")
        XCTAssertEqual(KeyLayout.qwerty(.english)[2].first?.key, .shift)
        XCTAssertEqual(pinyin[2].last?.key, .delete)
    }

    func testKeypadCoversEveryLetterOnce() {
        let groups = KeyLayout.keypad.flatMap { $0 }.compactMap { key -> String? in
            if case .letter = key.key { return key.label } else { return nil }
        }
        XCTAssertEqual(groups.joined().lowercased().sorted(), Array("abcdefghijklmnopqrstuvwxyz"))
    }

    func testSymbolsFollowTheScript() {
        XCTAssertTrue(KeyLayout.symbols(chinese: true, more: false).flatMap { $0 }.contains { $0.key == .text("，") })
        XCTAssertFalse(KeyLayout.symbols(chinese: false, more: false).flatMap { $0 }.contains { $0.key == .text("，") })
        for chinese in [false, true] {
            for more in [false, true] {
                let rows = KeyLayout.symbols(chinese: chinese, more: more)
                XCTAssertEqual(rows[0].count, 10)
                XCTAssertEqual(rows[1].count, 10)
                XCTAssertEqual(rows[2].first?.key, more ? .symbols : .moreSymbols, "page key on the left")
                XCTAssertEqual(rows[2].last?.key, .delete, "Delete stays at the right edge")
            }
        }
    }

    func testChineseLayoutsHaveSchemas() {
        XCTAssertEqual(TypingLayout.pinyin.schema, "arion_pinyin")
        XCTAssertEqual(TypingLayout.pinyin9.schema, "arion_pinyin_9key")
        XCTAssertNil(TypingLayout.english.schema)
    }
}

@MainActor
final class MicrophoneGainTests: XCTestCase {
    func testStopsMatchTheDesktop() {
        // packages/core/application/gain.go GainStops.
        XCTAssertEqual(MicrophoneGain.stops.map(\.decibels), [-6, -3, 0, 3, 6, 9, 12])
        XCTAssertEqual(MicrophoneGain.stops.map(\.centimeters), [15, 21, 30, 42, 60, 85, 120])
        XCTAssertEqual(MicrophoneGain.nearest(4.4).decibels, 3)
        XCTAssertEqual(MicrophoneGain.nearest(-9).decibels, -6)
    }

    func testLevelMapsDecibelsAsTheDesktop() {
        // packages/core/application/dictation.go Level: -60 dBFS is 0, -10 dBFS is 1.
        XCTAssertEqual(MicrophoneGain.level(rms: 0.001, decibels: 0), 0, accuracy: 0.001)
        XCTAssertEqual(MicrophoneGain.level(rms: 0.0316, decibels: 0), 0.6, accuracy: 0.001)
        XCTAssertEqual(MicrophoneGain.level(rms: 0.5, decibels: 0), 1)
        // +6 dB of gain lifts the meter by 6/50.
        XCTAssertEqual(MicrophoneGain.level(rms: 0.0316, decibels: 6), 0.72, accuracy: 0.002)
    }

    func testGainScalesAndClips() throws {
        let buffer = try XCTUnwrap(AVAudioPCMBuffer(pcmFormat: WebMicrophone.format, frameCapacity: 3))
        buffer.frameLength = 3
        let samples = try XCTUnwrap(buffer.int16ChannelData?[0])
        samples[0] = 1000; samples[1] = -1000; samples[2] = 30000
        MicrophoneGain.apply(buffer, decibels: 6)
        XCTAssertEqual(samples[0], 1995)
        XCTAssertEqual(samples[1], -1995)
        XCTAssertEqual(samples[2], 32767, "clips at full scale")
    }

    func testZeroLeavesAudioUntouched() throws {
        let buffer = try XCTUnwrap(AVAudioPCMBuffer(pcmFormat: WebMicrophone.format, frameCapacity: 1))
        buffer.frameLength = 1
        buffer.int16ChannelData![0][0] = 1234
        MicrophoneGain.apply(buffer, decibels: 0)
        XCTAssertEqual(buffer.int16ChannelData![0][0], 1234)
    }

    // The typewriter's cases mirror apps/desktop/ui/entities/dictation/typewriter_test.go.

    /// Advances in 16 ms frames until the text is fully shown and faded.
    private func type(_ w: inout Typewriter, _ session: UUID, _ text: String, final: Bool, from start: TimeInterval) -> (TimeInterval, [Typewriter.Frame]) {
        var now = start
        var frames: [Typewriter.Frame] = []
        for _ in 0..<1000 {
            let frame = w.update(session: session, text: text, final: final, now: now)
            frames.append(frame)
            if !frame.animating { return (now, frames) }
            now += 0.016
        }
        XCTFail("typewriter never settled")
        return (now, frames)
    }

    func testTypewriterTypesStreamingTextSmoothly() {
        var w = Typewriter()
        let (end, frames) = type(&w, UUID(), "今天下午三点开会", final: false, from: 0)
        XCTAssertEqual(frames.last?.text, "今天下午三点开会")
        for i in 1..<frames.count {
            XCTAssertLessThanOrEqual(frames[i].text.count - frames[i - 1].text.count, 2, "frame \(i) jumped")
        }
        XCTAssert((0.2...0.9).contains(end), "eight characters took \(end) s")
        let first = frames.first { !$0.text.isEmpty }!
        XCTAssertLessThan(first.opacity.last ?? 1, 1)
        XCTAssertEqual(frames.last?.opacity, [])
    }

    func testTypewriterSpringCatchesUpWithBacklog() {
        var short = Typewriter(), long = Typewriter()
        let (endShort, _) = type(&short, UUID(), "一二三四五六", final: false, from: 0)
        let (endLong, _) = type(&long, UUID(), String(repeating: "一二三四五六七八九十", count: 4), final: false, from: 0)
        XCTAssertLessThan(endLong / 40, endShort / 6)
    }

    func testTypewriterRevisesInPlaceWithoutRetreating() {
        var w = Typewriter()
        let session = UUID()
        let (now, _) = type(&w, session, "我们明天见面", final: false, from: 0)
        let revised = w.update(session: session, text: "我们明天再说", final: false, now: now + 0.016)
        XCTAssertEqual(revised.text, "我们明天再说")
        XCTAssertEqual(revised.opacity, [])
        var v = Typewriter()
        let stream = UUID()
        var at = 10.0, shown = 0
        for text in ["今天下午三点", "今天下午三点我们开", "今天下午三点，我们开会", "今天下午 3 点，我们开会讨论", "今天下午3点，我们开会讨论一下"] {
            for _ in 0..<6 {
                let count = v.update(session: stream, text: text, final: false, now: at).text.count
                XCTAssert(count >= shown || count >= text.count, "line retreated on \(text)")
                shown = count
                at += 0.016
            }
        }
        XCTAssertEqual(v.update(session: stream, text: "今天", final: false, now: at).text, "今天")
    }

    func testTypewriterFinalDrainsFastAndSessionResets() {
        var w = Typewriter()
        let text = "这是一段比较长的最终识别结果需要尽快显示出来"
        let (end, _) = type(&w, UUID(), text, final: true, from: 0)
        XCTAssertLessThanOrEqual(end, Double(text.count) * Typewriter.minInterval + Typewriter.fadeIn + 0.05)
        XCTAssertLessThanOrEqual(w.update(session: UUID(), text: "好", final: false, now: end).text.count, 1)
    }

    func testTypewriterKeepsEmojiWhole() {
        var w = Typewriter()
        let (_, frames) = type(&w, UUID(), "好👍🏽了", final: false, from: 0)
        XCTAssert(frames.allSatisfy { "好👍🏽了".hasPrefix($0.text) })
    }

    // MARK: Key targets

    /// Three 30-point keys 6 points apart, as one row of the keyboard.
    private let row: [KeyTargets.Key] = [
        .init(frame: CGRect(x: 0, y: 0, width: 30, height: 42), letter: "d"),
        .init(frame: CGRect(x: 36, y: 0, width: 30, height: 42), letter: "f"),
        .init(frame: CGRect(x: 72, y: 0, width: 30, height: 42), letter: "g"),
    ]

    func testTouchesBetweenAndBesideKeysGoToTheNearestKey() {
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 32, y: 20), in: row) { _ in nil }, 0)
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 34, y: 20), in: row) { _ in nil }, 1)
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 110, y: 20), in: row) { _ in nil }, 2)
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 50, y: 50), in: row) { _ in nil }, 1)
    }

    func testANearBoundaryTouchGoesToTheLetterThatContinues() {
        // Apple's example: after GO, a touch on F's edge toward D types D.
        let onlyD: (Character) -> Bool? = { $0 == "d" }
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 38, y: 20), in: row, continues: onlyD), 0)
        // Deep inside F, geometry wins.
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 51, y: 20), in: row, continues: onlyD), 1)
        // Nothing known: geometry wins at the edge too.
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 38, y: 20), in: row) { _ in nil }, 1)
        // Both continue: geometry wins.
        XCTAssertEqual(KeyTargets.resolve(CGPoint(x: 38, y: 20), in: row) { _ in true }, 1)
    }

    func testPinyinSpellingFollowsSyllables() {
        let spelling = PinyinSpelling(syllables: ["a", "ni", "hao", "zhong", "guo", "xian", "xi", "an", "e"])
        XCTAssertTrue(spelling.continues("nih"))
        XCTAssertTrue(spelling.continues("zhongg"))
        XCTAssertTrue(spelling.continues("xi'an"))
        XCTAssertFalse(spelling.continues("nq"))
        XCTAssertFalse(spelling.continues("zho'g"))
        XCTAssertFalse(spelling.continues("i"))
    }
}
