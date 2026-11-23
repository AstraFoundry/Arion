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
    /// Typing starts the Rime engine the first time it is shown.
    private var typing: TypingKeyboardView?
    /// The mode and layout the keyboard reopens in; keyboard-only state, so
    /// it lives in the extension's own defaults and needs no Full Access.
    private let memory = UserDefaults.standard
    private enum Memory {
        static let typing = "keyboard.typing"
        /// Whether typing was last in English rather than Chinese.
        static let english = "typing.english"
    }
    /// When the microphone started listening, for the day's listening time.
    private var listeningSince: Date?

    override func loadView() {
        view = KeyboardInputView(frame: .zero, inputViewStyle: .keyboard)
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        // Declare the height on the root input view before it first appears,
        // as Apple documents, so hosts that lay out from the first keyboard
        // frame (Telegram) see the height the panels draw. 999: below the
        // system's own required constraints.
        let height = view.heightAnchor.constraint(equalToConstant: KeyboardMetrics.height)
        height.priority = UILayoutPriority(999)
        height.isActive = true
        keyboard.translatesAutoresizingMaskIntoConstraints = false
        keyboard.delegate = self
        keyboard.globe.addTarget(self, action: #selector(handleInputModeList(from:with:)), for: .allTouchEvents)
        place(keyboard)
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
        if memory.bool(forKey: Memory.typing), typing == nil { showTyping() }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        session.cancel()
        microphone.detach()
        typing?.reset()
    }

    override func textDidChange(_ textInput: (any UITextInput)?) {
        super.textDidChange(textInput)
        refresh()
    }

    override func selectionDidChange(_ textInput: (any UITextInput)?) {
        super.selectionDidChange(textInput)
        // The caret moved elsewhere: a pending composition no longer fits.
        typing?.reset()
    }

    /// Shows one panel; the other stays built but off screen.
    private func place(_ panel: UIView) {
        view.subviews.forEach { $0.removeFromSuperview() }
        panel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(panel)
        NSLayoutConstraint.activate([
            panel.topAnchor.constraint(equalTo: view.topAnchor),
            panel.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            panel.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            panel.trailingAnchor.constraint(equalTo: view.trailingAnchor),
        ])
    }

    private func showTyping() {
        if typing == nil {
            do {
                guard let resources = Bundle.main.resourceURL,
                      let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
                else { throw TypingFailure.dataMissing }
                try TypingEngine.start(data: resources.appendingPathComponent("Rime/data"), support: support)
            } catch let failure as TypingFailure {
                memory.set(false, forKey: Memory.typing)
                keyboard.showNotice(Self.message(failure), autoClear: false)
                return
            } catch {
                return
            }
            let chinese = preferences.chineseTypingLayout
            let panel = TypingKeyboardView(engine: TypingEngine(layout: memory.bool(forKey: Memory.english) ? .english : chinese))
            panel.chineseLayout = chinese
            panel.delegate = self
            panel.globe.addTarget(self, action: #selector(handleInputModeList(from:with:)), for: .allTouchEvents)
            typing = panel
        }
        session.cancel()
        refreshTyping()
        place(typing!)
        memory.set(true, forKey: Memory.typing)
    }

    private func showVoice() {
        typing?.reset()
        place(keyboard)
        memory.set(false, forKey: Memory.typing)
    }

    private static func message(_ failure: TypingFailure) -> String {
        switch failure {
        case .dataMissing: L10n.typingFailureData
        case .learningKey: L10n.typingFailureLearning
        case .engine: L10n.typingFailureEngine
        }
    }

    /// A field that enables return automatically wants text before it.
    private var returnEnabled: Bool {
        textDocumentProxy.enablesReturnKeyAutomatically != true || textDocumentProxy.hasText
    }

    private func refreshTyping() {
        guard let typing else { return }
        typing.returnTitle = Self.returnTitle(textDocumentProxy.returnKeyType)
        typing.returnEnabled = returnEnabled
        typing.showsGlobe = needsInputModeSwitchKey
        // The app may have changed the setting since the keyboard opened.
        typing.chineseLayout = preferences.chineseTypingLayout
    }

    private func refresh() {
        refreshTyping()
        keyboard.globe.isHidden = !needsInputModeSwitchKey
        keyboard.returnTitle = Self.returnTitle(textDocumentProxy.returnKeyType)
        keyboard.returnEnabled = returnEnabled
        let languages = preferences.languages
        keyboard.languageMark = languages.count > 1 ? DictationLanguage.mark(preferences.language) : nil
        keyboard.languageName = DictationLanguage.name(preferences.language)
        keyboard.translationName = preferences.translationTarget.name
    }

    /// The return key's word for the field, or nil for a plain new line,
    /// which both panels draw as an icon.
    private static func returnTitle(_ type: UIReturnKeyType?) -> String? {
        switch type ?? .default {
        case .go: L10n.returnGo
        case .google, .yahoo, .search: L10n.returnSearch
        case .send: L10n.returnSend
        case .next: L10n.returnNext
        case .done: L10n.returnDone
        case .join: L10n.returnJoin
        case .route: L10n.returnRoute
        case .continue: L10n.returnContinue
        default: nil
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
        case .idle, .failed, .downloaded: begin()
        case .starting, .listening: end(.dictate)
        case .finishing, .polishing, .downloading: break
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
        // The download belongs to the shown language; switching would mislabel it.
        if case .downloading = session.phase { return }
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

    func keyboardShowTyping() { showTyping() }

    func keyboardDismiss() { dismissKeyboard() }
}

extension KeyboardViewController: TypingKeyboardViewDelegate {
    func typingInsert(_ text: String) {
        textDocumentProxy.insertText(text)
        refreshTyping()
    }

    func typingDeleteBackward() {
        guard textDocumentProxy.hasText else { return }
        textDocumentProxy.deleteBackward()
        refreshTyping()
    }

    func typingReturn() { textDocumentProxy.insertText("\n") }

    func typingLayoutChanged(_ layout: TypingLayout) { memory.set(layout == .english, forKey: Memory.english) }

    func typingShowVoice() { showVoice() }

    func typingDismiss() { dismissKeyboard() }

    func typingMoveCaret(by offset: Int) { textDocumentProxy.adjustTextPosition(byCharacterOffset: offset) }

    func typingTextBeforeCaret() -> String? { textDocumentProxy.documentContextBeforeInput }
}

/// The keyboard's root view; it opts in to the system input clicks.
final class KeyboardInputView: UIInputView, UIInputViewAudioFeedback {
    var enableInputClicksWhenVisible: Bool { true }
}
