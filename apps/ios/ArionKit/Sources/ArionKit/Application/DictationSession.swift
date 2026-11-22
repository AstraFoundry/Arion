@preconcurrency import AVFAudio
import Foundation

/// What a finished dictation delivers and how it was produced.
public struct Delivery: Equatable, Sendable {
    public enum Outcome: Equatable, Sendable {
        /// Polishing was off; the rules ran.
        case cleaned
        case polished
        case translated
        /// Polishing was on but could not run; the rule-corrected text is delivered.
        case polishUnavailable
        /// Translation could not run; the source-language text is delivered.
        case translationUnavailable
    }

    public let text: String
    /// The rule-corrected transcript the text was produced from.
    public let source: String
    public let outcome: Outcome
}

/// The request of one dictation: what to do with the speech once it ends.
public enum DictationIntent: Equatable, Sendable {
    case dictate
    case translate(TranslationTarget)
}

/// Runs one keyboard dictation at a time: microphone, on-device recognition,
/// rules, then polishing or translation. Every callback carries the
/// generation it belongs to, and the session ignores work from older ones.
@MainActor
public final class DictationSession {
    public var onPhase: ((DictationPhase) -> Void)?

    private let microphone: WebMicrophone
    private let recognizer = Recognizer()
    private let model = OnDeviceModel()
    private let cloud = CloudModel()
    private let recipe: RewriteRecipe?
    private let preferences: Preferences
    private var generation = DictationGeneration(0)
    private var draft = Draft()
    private var level: Float = 0
    private var finishing = false
    public private(set) var phase: DictationPhase = .idle {
        didSet { onPhase?(phase) }
    }

    public init(microphone: WebMicrophone, preferences: Preferences = Preferences()) {
        self.microphone = microphone
        self.preferences = preferences
        recipe = try? RewriteRecipe.bundled()
        microphone.onEvent = { [weak self] event in self?.receive(event) }
        recognizer.onDraft = { [weak self] draft in
            guard let self, self.phase.isActive, !self.finishing else { return }
            self.draft = draft
            self.phase = .listening(draft, level: self.level)
        }
    }

    /// Starts listening. The caller has checked Full Access.
    public func start(language: String, vocabulary: [String]) {
        guard !phase.isActive else { return }
        // The extension may not see the app's grant; only a denial is certain,
        // and WebKit reports any other refusal when capture starts.
        guard AVAudioApplication.shared.recordPermission != .denied else {
            phase = .failed(.microphoneDenied)
            return
        }
        generation = generation.next()
        let current = generation
        draft = Draft()
        finishing = false
        phase = .starting
        Task {
            do {
                try await recognizer.start(language: language, vocabulary: vocabulary)
                guard current == generation, phase == .starting else { return }
                microphone.start()
            } catch let failure as DictationFailure {
                guard current == generation else { return }
                phase = .failed(failure)
            } catch {
                guard current == generation else { return }
                phase = .failed(.recognitionFailed)
            }
        }
    }

    /// Stops listening and produces the delivery, or nil when cancelled or
    /// superseded before it finished.
    public func finish(intent: DictationIntent, scene: Scene) async -> Delivery? {
        guard phase.isActive, !finishing else { return nil }
        let current = generation
        finishing = true
        microphone.stop()
        phase = .finishing(draft)
        let raw: String
        do {
            raw = try await recognizer.finish()
        } catch {
            guard current == generation else { return nil }
            phase = .failed(.recognitionFailed)
            return nil
        }
        guard current == generation else { return nil }
        let source = CorrectionRules.apply(raw)
        guard !source.isEmpty else {
            phase = .failed(.nothingHeard)
            return nil
        }
        let delivery = await produce(source, intent: intent, scene: scene, generation: current)
        guard current == generation else { return nil }
        phase = .idle
        return delivery
    }

    /// Drops the current dictation. Nothing it produced is delivered.
    public func cancel() {
        generation = generation.next()
        finishing = false
        microphone.stop()
        recognizer.cancel()
        phase = .idle
    }

    /// Clears a shown failure.
    public func dismissFailure() {
        if case .failed = phase { phase = .idle }
    }

    private func produce(_ source: String, intent: DictationIntent, scene: Scene, generation current: DictationGeneration) async -> Delivery {
        switch intent {
        case .translate(let target):
            guard let request = recipe?.translate(source, scene: scene, target: target) else {
                return Delivery(text: source, source: source, outcome: .translationUnavailable)
            }
            phase = .polishing(Draft(finalized: source))
            if let text = try? await run(request) {
                return Delivery(text: text, source: source, outcome: .translated)
            }
            return Delivery(text: source, source: source, outcome: .translationUnavailable)
        case .dictate:
            guard preferences.polishingEnabled else {
                return Delivery(text: source, source: source, outcome: .cleaned)
            }
            guard let recipe, engineReady else {
                return Delivery(text: source, source: source, outcome: .polishUnavailable)
            }
            phase = .polishing(Draft(finalized: source))
            if let text = try? await run(recipe.polish(source, scene: scene)) {
                return Delivery(text: text, source: source, outcome: .polished)
            }
            return Delivery(text: source, source: source, outcome: .polishUnavailable)
        }
    }

    /// Whether the chosen engine can run: a connected cloud model, or Apple
    /// Intelligence on this device.
    private var engineReady: Bool {
        preferences.polishEngine == .cloud ? CloudModel.ready : OnDeviceModel.availability == .available
    }

    /// Runs a request on the chosen engine. The cloud receives only the
    /// rule-corrected transcript and the recipe.
    private func run(_ request: RewriteRequest) async throws -> String {
        preferences.polishEngine == .cloud ? try await cloud.run(request) : try await model.run(request)
    }

    private func receive(_ event: WebMicrophone.Event) {
        switch event {
        case .started:
            if phase == .starting { phase = .listening(draft, level: 0) }
        case .audio(let buffer, let level):
            self.level = level
            recognizer.append(buffer)
            if case .listening = phase, !finishing { phase = .listening(draft, level: level) }
        case .ended(let reason):
            // A stop we asked for is normal; any other end loses the microphone.
            guard reason != "stopped", phase.isActive, !finishing else { return }
            generation = generation.next()
            recognizer.cancel()
            // getUserMedia rejects a refused microphone with NotAllowedError.
            phase = .failed(reason.contains("NotAllowedError") ? .microphoneDenied : .captureEnded)
        }
    }
}
