import ArionKit
import UIKit

/// Connects Arion's keyboard view to the editor. Recognition drafts stay in
/// the keyboard; the editor receives one insertion when the dictation ends.
final class KeyboardViewController: UIInputViewController {
    private let preferences = Preferences()
    private lazy var microphone = WebMicrophone()
    private lazy var session = DictationSession(microphone: microphone, preferences: preferences)
    private let keyboard = VoiceKeyboardView()
    private let usage = UsageLedger()
    /// When the microphone started listening, for the day's listening time.
    private var listeningSince: Date?

    override func loadView() {
        view = KeyboardInputView(frame: .zero, inputViewStyle: .keyboard)
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        keyboard.translatesAutoresizingMaskIntoConstraints = false
        keyboard.delegate = self
        keyboard.globe.addTarget(self, action: #selector(handleInputModeList(from:with:)), for: .allTouchEvents)
        view.addSubview(keyboard)
        NSLayoutConstraint.activate([
            keyboard.topAnchor.constraint(equalTo: view.topAnchor),
            keyboard.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            keyboard.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            keyboard.trailingAnchor.constraint(equalTo: view.trailingAnchor),
        ])
        session.onPhase = { [weak self] phase in
            guard let self else { return }
            if case .listening = phase, self.listeningSince == nil { self.listeningSince = .now }
            if phase == .starting { self.listeningSince = nil }
            self.keyboard.render(phase)
        }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        if hasFullAccess {
            microphone.attach(to: view)
            preferences.keyboardLastSeen = Date()
        }
        refresh()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        session.cancel()
        microphone.detach()
    }

    override func textDidChange(_ textInput: (any UITextInput)?) {
        super.textDidChange(textInput)
        refresh()
    }

    private func refresh() {
        keyboard.globe.isHidden = !needsInputModeSwitchKey
        keyboard.returnTitle = Self.returnTitle(textDocumentProxy.returnKeyType)
        let languages = preferences.languages
        keyboard.languageMark = languages.count > 1 ? DictationLanguage.mark(preferences.language) : nil
        keyboard.languageName = DictationLanguage.name(preferences.language)
        keyboard.translationName = preferences.translationTarget.name
    }

    private static func returnTitle(_ type: UIReturnKeyType?) -> String {
        switch type ?? .default {
        case .go: L10n.returnGo
        case .google, .yahoo, .search: L10n.returnSearch
        case .send: L10n.returnSend
        case .next: L10n.returnNext
        case .done: L10n.returnDone
        case .join: L10n.returnJoin
        case .route: L10n.returnRoute
        case .continue: L10n.returnContinue
        default: L10n.returnDefault
        }
    }

    /// The scene follows the field: a search field gets search finishing,
    /// everything else chat. The host app's identity is not available here.
    private var scene: Scene {
        let proxy = textDocumentProxy
        if proxy.keyboardType == .webSearch { return .search }
        switch proxy.returnKeyType {
        case .search, .google, .yahoo: return .search
        default: return .chat
        }
    }

    private func begin() {
        guard hasFullAccess else {
            keyboard.render(.failed(.fullAccessRequired))
            return
        }
        session.start(language: preferences.language, vocabulary: (try? Vocabulary().terms()) ?? [])
    }

    private func end(_ intent: DictationIntent) {
        let scene = scene
        let listened = listeningSince.map { Int64(Date.now.timeIntervalSince($0) * 1000) } ?? 0
        listeningSince = nil
        Task {
            guard let delivery = await session.finish(intent: intent, scene: scene) else { return }
            insert(delivery)
            // Numbers and the scene only; the text is counted, never kept (005).
            let polished = delivery.outcome == .polished || delivery.outcome == .translated
            try? usage.record(speechMs: listened, text: delivery.text, scene: polished ? scene : nil)
        }
    }

    private func insert(_ delivery: Delivery) {
        textDocumentProxy.insertText(TextJoining.joining(delivery.text, after: textDocumentProxy.documentContextBeforeInput))
        if preferences.historyEnabled {
            try? History().record(HistoryEntry(text: delivery.text, original: delivery.source))
        }
        switch delivery.outcome {
        case .polishUnavailable where preferences.polishingEnabled && (preferences.polishEngine == .cloud || OnDeviceModel.availability == .available):
            keyboard.showNotice(L10n.noticeUnpolished, autoClear: true)
        case .translationUnavailable:
            keyboard.showNotice(L10n.noticeUntranslated, autoClear: true)
        default:
            break
        }
    }
}

extension KeyboardViewController: VoiceKeyboardViewDelegate {
    func keyboardMicTapped() {
        switch session.phase {
        case .idle, .failed: begin()
        case .starting, .listening: end(.dictate)
        case .finishing, .polishing: break
        }
    }

    func keyboardHoldBegan() { begin() }

    func keyboardHoldEnded(_ target: VoiceKeyboardView.HoldTarget) {
        switch target {
        case .cancel: session.cancel()
        case .translate: end(.translate(preferences.translationTarget))
        case .mic: end(.dictate)
        }
    }

    func keyboardLeadingTapped() {
        if session.phase.isActive {
            session.cancel()
            return
        }
        let languages = preferences.languages
        guard languages.count > 1, let index = languages.firstIndex(of: preferences.language) else { return }
        preferences.language = languages[(index + 1) % languages.count]
        refresh()
    }

    func keyboardTranslateTapped() { end(.translate(preferences.translationTarget)) }

    func keyboardDelete() {
        session.dismissFailure()
        guard textDocumentProxy.hasText else { return }
        UIDevice.current.playInputClick()
        textDocumentProxy.deleteBackward()
    }

    func keyboardSpace() {
        UIDevice.current.playInputClick()
        textDocumentProxy.insertText(" ")
    }

    func keyboardMoveCaret(by offset: Int) {
        textDocumentProxy.adjustTextPosition(byCharacterOffset: offset)
    }

    func keyboardReturn() {
        UIDevice.current.playInputClick()
        textDocumentProxy.insertText("\n")
    }
}

/// The keyboard's root view; it opts in to the system input clicks.
final class KeyboardInputView: UIInputView, UIInputViewAudioFeedback {
    var enableInputClicksWhenVisible: Bool { true }
}
