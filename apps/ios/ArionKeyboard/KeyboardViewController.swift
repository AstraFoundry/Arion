import ArionKit
import UIKit

/// Arion's keyboard: one microphone, the keys a voice keyboard still needs,
/// and the text being recognized. Recognition drafts stay in the keyboard;
/// the editor receives one insertion when the dictation ends.
final class KeyboardViewController: UIInputViewController {
    private let preferences = Preferences()
    private lazy var microphone = WebMicrophone()
    private lazy var session = DictationSession(microphone: microphone, preferences: preferences)

    private let draftLabel = UILabel(style: .body)
    private let mic = MicButton()
    private let leading = FaceButton(kind: .plain, material: .raised)
    private let trailing = FaceButton(kind: .plain, material: .raised)
    private let globe = FaceButton(kind: .secondary, material: .raised, symbol: "globe")
    private let space = FaceButton(kind: .secondary, material: .raised)
    private let returnKey = FaceButton(kind: .secondary, material: .raised)
    private let feedback = UIImpactFeedbackGenerator(style: .light)

    private var holding = false
    private var holdTarget: HoldTarget = .mic
    private var deleteTimer: Timer?
    private var noticeTask: Task<Void, Never>?
    private var spaceAnchor: CGFloat = 0

    private enum HoldTarget { case mic, cancel, translate }

    override func loadView() {
        view = KeyboardInputView(frame: .zero, inputViewStyle: .keyboard)
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        buildLayout()
        session.onPhase = { [weak self] phase in self?.render(phase) }
        render(.idle)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        if hasFullAccess {
            microphone.attach(to: view)
            preferences.keyboardLastSeen = Date()
        }
        refreshKeys()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        session.cancel()
        microphone.detach()
    }

    override func textDidChange(_ textInput: (any UITextInput)?) {
        super.textDidChange(textInput)
        refreshKeys()
    }

    // MARK: Layout

    private func buildLayout() {
        guard let root = view else { return }
        root.translatesAutoresizingMaskIntoConstraints = false

        draftLabel.textAlignment = .center
        draftLabel.numberOfLines = 3
        draftLabel.lineBreakMode = .byTruncatingHead
        draftLabel.adjustsFontForContentSizeCategory = false
        draftLabel.accessibilityTraits = .updatesFrequently

        for side in [leading, trailing] {
            side.cornerRadius = 22
            side.widthAnchor.constraint(equalToConstant: 56).isActive = true
            side.heightAnchor.constraint(equalToConstant: 44).isActive = true
        }
        leading.titleLabel.font = TextStyle.label.font
        trailing.addTarget(self, action: #selector(trailingDown), for: .touchDown)
        trailing.addTarget(self, action: #selector(trailingUp), for: [.touchUpInside, .touchUpOutside, .touchCancel])
        leading.addTarget(self, action: #selector(leadingTapped), for: .touchUpInside)

        let tap = UITapGestureRecognizer(target: self, action: #selector(micTapped))
        let hold = UILongPressGestureRecognizer(target: self, action: #selector(micHeld(_:)))
        hold.minimumPressDuration = 0.3
        tap.require(toFail: hold)
        mic.addGestureRecognizer(tap)
        mic.addGestureRecognizer(hold)

        globe.accessibilityLabel = L10n.keyboardNextKeyboard
        globe.cornerRadius = SunbeamTokens.Radius.lg
        globe.addTarget(self, action: #selector(handleInputModeList(from:with:)), for: .allTouchEvents)

        space.title = L10n.keyboardSpace
        space.cornerRadius = SunbeamTokens.Radius.lg
        space.addTarget(self, action: #selector(spaceTapped), for: .touchUpInside)
        space.addGestureRecognizer(UIPanGestureRecognizer(target: self, action: #selector(spaceDragged(_:))))

        returnKey.cornerRadius = SunbeamTokens.Radius.lg
        returnKey.addTarget(self, action: #selector(returnTapped), for: .touchUpInside)

        let center = UIStackView(arrangedSubviews: [leading, mic, trailing])
        center.axis = .horizontal
        center.alignment = .center
        center.spacing = 40

        let keys = UIStackView(arrangedSubviews: [globe, space, returnKey])
        keys.axis = .horizontal
        keys.spacing = 6
        keys.distribution = .fill
        globe.widthAnchor.constraint(equalToConstant: 48).isActive = true
        returnKey.widthAnchor.constraint(equalToConstant: 92).isActive = true

        for view in [draftLabel, center, keys] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            root.addSubview(view)
        }
        let height = root.heightAnchor.constraint(equalToConstant: 236)
        height.priority = .defaultHigh
        NSLayoutConstraint.activate([
            height,
            draftLabel.topAnchor.constraint(equalTo: root.topAnchor, constant: 10),
            draftLabel.leadingAnchor.constraint(equalTo: root.leadingAnchor, constant: 20),
            draftLabel.trailingAnchor.constraint(equalTo: root.trailingAnchor, constant: -20),
            draftLabel.heightAnchor.constraint(equalToConstant: 64),
            center.centerXAnchor.constraint(equalTo: root.centerXAnchor),
            center.topAnchor.constraint(equalTo: draftLabel.bottomAnchor, constant: 6),
            keys.topAnchor.constraint(equalTo: center.bottomAnchor, constant: 14),
            keys.leadingAnchor.constraint(equalTo: root.leadingAnchor, constant: 6),
            keys.trailingAnchor.constraint(equalTo: root.trailingAnchor, constant: -6),
            keys.heightAnchor.constraint(equalToConstant: 44),
            keys.bottomAnchor.constraint(equalTo: root.bottomAnchor, constant: -6),
        ])
    }

    private func refreshKeys() {
        globe.isHidden = !needsInputModeSwitchKey
        returnKey.title = Self.returnTitle(textDocumentProxy.returnKeyType)
        if !session.phase.isActive { render(session.phase) }
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

    // MARK: Rendering

    private func render(_ phase: DictationPhase) {
        let languages = preferences.languages
        switch phase {
        case .idle, .failed:
            mic.show(.idle)
            configure(leading, symbol: nil, title: languages.count > 1 ? DictationLanguage.mark(preferences.language) : nil,
                      label: L10n.keyboardLanguage(language: DictationLanguage.name(preferences.language)))
            configure(trailing, symbol: "delete.left", title: nil, label: L10n.keyboardDelete)
            if case .failed(let failure) = phase {
                showNotice(Self.message(failure), muted: true, autoClear: false)
            } else if noticeTask == nil {
                draftLabel.text = nil
            }
        case .starting:
            mic.show(.listening)
            setCancelAndTranslate()
            draftLabel.text = nil
        case .listening(let draft, let level):
            mic.show(.listening)
            mic.setLevel(level)
            setCancelAndTranslate()
            showDraft(draft)
        case .finishing(let draft):
            mic.show(.working)
            configure(leading, symbol: "xmark", title: nil, label: L10n.keyboardCancel)
            trailing.isHidden = true
            showDraft(draft)
        case .polishing(let draft):
            mic.show(.working)
            configure(leading, symbol: "xmark", title: nil, label: L10n.keyboardCancel)
            trailing.isHidden = true
            showDraft(draft)
        }
        if holding { highlightHoldTarget() }
    }

    private func setCancelAndTranslate() {
        configure(leading, symbol: "xmark", title: nil, label: L10n.keyboardCancel)
        configure(trailing, symbol: "translate", title: nil, label: L10n.keyboardTranslate(language: preferences.translationTarget.name))
    }

    private func configure(_ button: FaceButton, symbol: String?, title: String?, label: String) {
        button.isHidden = symbol == nil && title == nil
        button.symbolSize = 19
        button.symbol = symbol
        button.title = title
        button.accessibilityLabel = label
    }

    private func showDraft(_ draft: Draft) {
        noticeTask?.cancel()
        noticeTask = nil
        let text = NSMutableAttributedString(string: draft.finalized, attributes: [.foregroundColor: Palette.foreground])
        text.append(NSAttributedString(string: draft.volatile, attributes: [.foregroundColor: Palette.mutedForeground]))
        draftLabel.attributedText = text
        draftLabel.font = TextStyle.body.font
    }

    private func showNotice(_ message: String, muted: Bool, autoClear: Bool) {
        noticeTask?.cancel()
        draftLabel.attributedText = nil
        draftLabel.text = message
        draftLabel.font = TextStyle.bodySmall.font
        draftLabel.textColor = muted ? Palette.mutedForeground : Palette.foreground
        UIAccessibility.post(notification: .announcement, argument: message)
        guard autoClear else {
            noticeTask = nil
            return
        }
        noticeTask = Task { [weak self] in
            try? await Task.sleep(for: .seconds(2.5))
            guard !Task.isCancelled, let self else { return }
            self.draftLabel.text = nil
            self.noticeTask = nil
        }
    }

    private static func message(_ failure: DictationFailure) -> String {
        switch failure {
        case .fullAccessRequired: L10n.failureFullAccess
        case .microphoneDenied: L10n.failureMicrophone
        case .languageUnavailable: L10n.failureLanguage
        case .modelInstalling: L10n.failureInstalling
        case .captureEnded: L10n.failureCapture
        case .recognitionFailed: L10n.failureRecognition
        case .nothingHeard: L10n.failureNothingHeard
        }
    }

    // MARK: Dictation

    private func begin() {
        guard hasFullAccess else {
            render(.failed(.fullAccessRequired))
            return
        }
        feedback.impactOccurred()
        let vocabulary = (try? Vocabulary().terms()) ?? []
        session.start(language: preferences.language, vocabulary: vocabulary)
    }

    private func end(_ intent: DictationIntent) {
        feedback.impactOccurred()
        let scene = scene
        Task {
            guard let delivery = await session.finish(intent: intent, scene: scene) else { return }
            insert(delivery)
        }
    }

    private func insert(_ delivery: Delivery) {
        textDocumentProxy.insertText(Self.joining(delivery.text, after: textDocumentProxy.documentContextBeforeInput))
        if preferences.historyEnabled {
            try? History().record(HistoryEntry(text: delivery.text, original: delivery.source))
        }
        switch delivery.outcome {
        case .polishUnavailable where preferences.polishingEnabled && OnDeviceModel.availability == .available:
            showNotice(L10n.noticeUnpolished, muted: true, autoClear: true)
        case .translationUnavailable:
            showNotice(L10n.noticeUntranslated, muted: true, autoClear: true)
        default:
            break
        }
    }

    /// Adds a space between Latin words, as the system keyboard would.
    static func joining(_ text: String, after context: String?) -> String {
        guard let last = context?.last, let first = text.first else { return text }
        let wordLike = { (c: Character) in c.isLetter && c.unicodeScalars.allSatisfy { $0.value < 0x2E80 } || c.isNumber }
        if (wordLike(last) || ".,!?;:".contains(last)) && wordLike(first) { return " " + text }
        return text
    }

    // MARK: Actions

    @objc private func micTapped() {
        switch session.phase {
        case .idle, .failed: begin()
        case .starting, .listening: end(.dictate)
        case .finishing, .polishing: break
        }
    }

    @objc private func micHeld(_ gesture: UILongPressGestureRecognizer) {
        switch gesture.state {
        case .began:
            guard !session.phase.isActive else { return }
            holding = true
            holdTarget = .mic
            begin()
        case .changed:
            guard holding else { return }
            let point = gesture.location(in: view)
            let next: HoldTarget = leading.frame(in: view).insetBy(dx: -16, dy: -24).contains(point) ? .cancel
                : trailing.frame(in: view).insetBy(dx: -16, dy: -24).contains(point) ? .translate : .mic
            if next != holdTarget {
                holdTarget = next
                feedback.impactOccurred(intensity: 0.6)
                highlightHoldTarget()
            }
        case .ended, .cancelled, .failed:
            guard holding else { return }
            holding = false
            leading.isHighlighted = false
            trailing.isHighlighted = false
            switch holdTarget {
            case .cancel: session.cancel()
            case .translate: end(.translate(preferences.translationTarget))
            case .mic: end(.dictate)
            }
        default:
            break
        }
    }

    private func highlightHoldTarget() {
        leading.isHighlighted = holdTarget == .cancel
        trailing.isHighlighted = holdTarget == .translate
    }

    @objc private func leadingTapped() {
        if session.phase.isActive {
            session.cancel()
            return
        }
        let languages = preferences.languages
        guard languages.count > 1, let index = languages.firstIndex(of: preferences.language) else { return }
        preferences.language = languages[(index + 1) % languages.count]
        UIDevice.current.playInputClick()
        render(session.phase)
    }

    @objc private func trailingDown() {
        switch session.phase {
        case .starting, .listening:
            end(.translate(preferences.translationTarget))
        case .idle, .failed:
            session.dismissFailure()
            deleteBackward()
            deleteTimer?.invalidate()
            deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.45, repeats: false) { [weak self] _ in
                MainActor.assumeIsolated {
                    self?.deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.08, repeats: true) { _ in
                        MainActor.assumeIsolated { self?.deleteBackward() }
                    }
                }
            }
        default:
            break
        }
    }

    @objc private func trailingUp() {
        deleteTimer?.invalidate()
        deleteTimer = nil
    }

    private func deleteBackward() {
        guard textDocumentProxy.hasText else { return }
        UIDevice.current.playInputClick()
        textDocumentProxy.deleteBackward()
    }

    @objc private func spaceTapped() {
        UIDevice.current.playInputClick()
        textDocumentProxy.insertText(" ")
    }

    /// Dragging along the space bar moves the caret, as on the system keyboard.
    @objc private func spaceDragged(_ gesture: UIPanGestureRecognizer) {
        switch gesture.state {
        case .began:
            spaceAnchor = 0
        case .changed:
            let x = gesture.translation(in: space).x
            let step: CGFloat = 9
            let moves = Int((x - spaceAnchor) / step)
            if moves != 0 {
                textDocumentProxy.adjustTextPosition(byCharacterOffset: moves)
                spaceAnchor += CGFloat(moves) * step
            }
        default:
            break
        }
    }

    @objc private func returnTapped() {
        UIDevice.current.playInputClick()
        textDocumentProxy.insertText("\n")
    }
}

/// The keyboard's root view; it opts in to the system input clicks.
final class KeyboardInputView: UIInputView, UIInputViewAudioFeedback {
    var enableInputClicksWhenVisible: Bool { true }
}

private extension UIView {
    func frame(in other: UIView?) -> CGRect {
        convert(bounds, to: other)
    }
}
