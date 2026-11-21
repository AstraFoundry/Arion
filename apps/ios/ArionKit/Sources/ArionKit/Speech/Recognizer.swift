@preconcurrency import AVFAudio
import Foundation
import Speech

/// Streams microphone buffers through Apple's on-device dictation model and
/// reports a revisioned draft. Audio never leaves the device.
@MainActor
public final class Recognizer {
    public var onDraft: ((Draft) -> Void)?

    private var analyzer: SpeechAnalyzer?
    private var input: AsyncStream<AnalyzerInput>.Continuation?
    private var results: Task<Void, Error>?
    private var converter: AVAudioConverter?
    private var targetFormat: AVAudioFormat?
    private var draft = Draft()

    public init() {}

    /// Whether the on-device model for a language is installed.
    public static func isInstalled(_ language: String) async -> Bool {
        guard let locale = await DictationTranscriber.supportedLocale(equivalentTo: Locale(identifier: language)) else { return false }
        let installed = await DictationTranscriber.installedLocales
        return installed.contains { $0.identifier(.bcp47) == locale.identifier(.bcp47) }
    }

    /// Downloads the on-device model for a language when it is missing.
    public static func install(_ language: String) async throws {
        guard let locale = await DictationTranscriber.supportedLocale(equivalentTo: Locale(identifier: language)) else {
            throw DictationFailure.languageUnavailable
        }
        let module = DictationTranscriber(locale: locale, preset: .progressiveShortDictation)
        if let request = try await AssetInventory.assetInstallationRequest(supporting: [module]) {
            try await request.downloadAndInstall()
        }
    }

    /// Prepares the model and begins accepting audio.
    public func start(language: String, vocabulary: [String]) async throws {
        cancel()
        guard let locale = await DictationTranscriber.supportedLocale(equivalentTo: Locale(identifier: language)) else {
            throw DictationFailure.languageUnavailable
        }
        guard await Self.isInstalled(language) else {
            // Begin the download so the next dictation can run, and say so now.
            Task.detached { try? await Self.install(language) }
            throw DictationFailure.modelInstalling
        }
        let transcriber = DictationTranscriber(
            locale: locale,
            contentHints: [.shortForm],
            transcriptionOptions: [.punctuation],
            reportingOptions: [.volatileResults, .frequentFinalization],
            attributeOptions: []
        )
        guard let format = await SpeechAnalyzer.bestAvailableAudioFormat(compatibleWith: [transcriber], considering: WebMicrophone.format) else {
            throw DictationFailure.modelInstalling
        }
        targetFormat = format
        converter = format == WebMicrophone.format ? nil : AVAudioConverter(from: WebMicrophone.format, to: format)
        let analyzer = SpeechAnalyzer(modules: [transcriber])
        if !vocabulary.isEmpty {
            let context = AnalysisContext()
            context.contextualStrings[.general] = vocabulary
            try await analyzer.setContext(context)
        }
        try await analyzer.prepareToAnalyze(in: format)
        let (stream, continuation) = AsyncStream.makeStream(of: AnalyzerInput.self)
        self.analyzer = analyzer
        input = continuation
        draft = Draft()
        results = Task { [weak self] in
            for try await result in transcriber.results {
                let text = String(result.text.characters)
                await MainActor.run {
                    guard let self, self.analyzer === analyzer else { return }
                    if result.isFinal {
                        self.draft.finalized += text
                        self.draft.volatile = ""
                    } else {
                        self.draft.volatile = text
                    }
                    self.onDraft?(self.draft)
                }
            }
        }
        try await analyzer.start(inputSequence: stream)
    }

    public func append(_ buffer: AVAudioPCMBuffer) {
        guard let input else { return }
        guard let converter, let targetFormat else {
            input.yield(AnalyzerInput(buffer: buffer))
            return
        }
        let ratio = targetFormat.sampleRate / buffer.format.sampleRate
        let capacity = AVAudioFrameCount(Double(buffer.frameLength) * ratio) + 32
        guard let output = AVAudioPCMBuffer(pcmFormat: targetFormat, frameCapacity: capacity) else { return }
        let supplied = Once()
        var error: NSError?
        converter.convert(to: output, error: &error) { _, status in
            if supplied.done {
                status.pointee = .noDataNow
                return nil
            }
            supplied.done = true
            status.pointee = .haveData
            return buffer
        }
        if error == nil, output.frameLength > 0 {
            input.yield(AnalyzerInput(buffer: output))
        }
    }

    /// Finalizes everything heard and returns the transcript.
    public func finish() async throws -> String {
        guard let analyzer else { return draft.text }
        input?.finish()
        input = nil
        try await analyzer.finalizeAndFinishThroughEndOfInput()
        try await results?.value
        let text = draft.text
        reset()
        return text
    }

    /// Drops the dictation; nothing it recognized is reported afterwards.
    public func cancel() {
        input?.finish()
        input = nil
        results?.cancel()
        if let analyzer {
            Task { await analyzer.cancelAndFinishNow() }
        }
        reset()
    }

    private func reset() {
        analyzer = nil
        results = nil
        converter = nil
        targetFormat = nil
    }
}

/// Marks the converter's single input buffer as supplied. AVAudioConverter
/// calls its input block synchronously inside convert.
private final class Once: @unchecked Sendable {
    var done = false
}
