import UIKit

/// What the keyboard view reports; the input view controller acts on it.
@MainActor
public protocol VoiceKeyboardViewDelegate: AnyObject {
    func keyboardMicTapped()
    func keyboardHoldBegan()
    func keyboardHoldEnded(_ target: VoiceKeyboardView.HoldTarget)
    func keyboardLeadingTapped()
    func keyboardTranslateTapped()
    func keyboardDelete()
    func keyboardSpace()
    func keyboardMoveCaret(by offset: Int)
    func keyboardReturn()
    /// The keyboard key: switch from the microphone to typing.
    func keyboardShowTyping()
    /// The hide key, top right on both panels.
    func keyboardDismiss()
}

/// Arion's keyboard: one microphone, the keys a voice keyboard still needs,
/// and the text being recognized. The keyboard is a layer above the app, so
/// its controls are Sunbeam's raised material.
///
/// Tap the microphone to start and again to finish. Or hold it and speak:
/// releasing finishes, releasing over cancel or translate does that instead.
public final class VoiceKeyboardView: UIView {
    public enum HoldTarget { case mic, cancel, translate }

    public weak var delegate: VoiceKeyboardViewDelegate?

    /// The globe key; the controller routes it to the system's keyboard list.
    public let globe = FaceButton(kind: .secondary, material: .raised, icon: .globe)
    private let typing = KeyboardTopKey(icon: .keyboard, label: L10n.typingKeyboard)
    private(set) var draftLabel = UILabel(style: .body)
    private let mic = MicButton()
    private let leading = FaceButton(kind: .plain, material: .raised)
    private let trailing = FaceButton(kind: .plain, material: .raised)
    private let space = FaceButton(kind: .secondary, material: .raised)
    private let returnKey = FaceButton(kind: .secondary, material: .raised)
    private let feedback = UIImpactFeedbackGenerator(style: .light)

    private var phase: DictationPhase = .idle
    private var holding = false
    private var holdTarget: HoldTarget = .mic
    private var deleteTimer: Timer?
    private var noticeTask: Task<Void, Never>?
    private var spaceAnchor: CGFloat = 0
    /// Paces the draft as the desktop pill does; one session per dictation.
    private var typewriter = Typewriter()
    private var typingSession = UUID()
    private var typed: (draft: Draft, final: Bool)?
    private var typingLink: CADisplayLink?

    /// The mark of the current language, shown when there is more than one.
    public var languageMark: String? { didSet { render(phase) } }
    public var languageName = "" { didSet { render(phase) } }
    public var translationName = "" { didSet { render(phase) } }
    /// The field's return word (搜索, 发送…), or nil for a plain new line,
    /// which the key shows as an icon.
    public var returnTitle: String? { didSet { showReturn() } }
    private let dismiss = KeyboardTopKey(icon: .chevronDown, label: L10n.keyboardDismiss)

    /// False when the field enables return only once it has text and it has none.
    public var returnEnabled = true { didSet { showReturn() } }

    /// Like the system key: a named action is the yellow key, a new line a
    /// gray icon, dimmed while the field needs text first.
    private func showReturn() {
        returnKey.kind = returnTitle == nil ? .secondary : .primary
        returnKey.isEnabled = returnEnabled
        returnKey.title = returnTitle
        returnKey.icon = returnTitle == nil ? .cornerDownLeft : nil
        returnKey.accessibilityLabel = returnTitle ?? L10n.returnDefault
    }


    public override init(frame: CGRect) {
        super.init(frame: frame)
        build()
        render(.idle)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    private func build() {
        draftLabel.textAlignment = .center
        draftLabel.numberOfLines = 3
        draftLabel.lineBreakMode = .byTruncatingHead
        draftLabel.adjustsFontForContentSizeCategory = false
        draftLabel.accessibilityTraits = .updatesFrequently

        for side in [leading, trailing] {
            side.cornerRadius = 22
            side.iconSize = 20
            side.widthAnchor.constraint(equalToConstant: 56).isActive = true
            side.heightAnchor.constraint(equalToConstant: 44).isActive = true
        }
        leading.addTarget(self, action: #selector(leadingTapped), for: .touchUpInside)
        trailing.addTarget(self, action: #selector(trailingDown), for: .touchDown)
        trailing.addTarget(self, action: #selector(trailingUp), for: [.touchUpInside, .touchUpOutside, .touchCancel])

        let tap = UITapGestureRecognizer(target: self, action: #selector(micTapped))
        let hold = UILongPressGestureRecognizer(target: self, action: #selector(micHeld(_:)))
        hold.minimumPressDuration = 0.3
        tap.require(toFail: hold)
        mic.addGestureRecognizer(tap)
        mic.addGestureRecognizer(hold)

        globe.accessibilityLabel = L10n.keyboardNextKeyboard
        globe.cornerRadius = SunbeamTokens.Radius.lg
        space.title = L10n.keyboardSpace
        space.cornerRadius = SunbeamTokens.Radius.lg
        space.addTarget(self, action: #selector(spaceTapped), for: .touchUpInside)
        space.addGestureRecognizer(UIPanGestureRecognizer(target: self, action: #selector(spaceDragged(_:))))
        showReturn()
        dismiss.addAction(UIAction { [weak self] _ in self?.delegate?.keyboardDismiss() }, for: .touchUpInside)
        returnKey.cornerRadius = SunbeamTokens.Radius.lg
        returnKey.addTarget(self, action: #selector(returnTapped), for: .touchUpInside)

        let center = UIStackView(arrangedSubviews: [leading, mic, trailing])
        center.axis = .horizontal
        center.alignment = .center
        center.spacing = 40

        typing.addAction(UIAction { [weak self] _ in self?.delegate?.keyboardShowTyping() }, for: .touchUpInside)

        // The switch and hide keys sit in the top corners on both panels,
        // which share their height, so neither moves between them.
        let keys = UIStackView(arrangedSubviews: [globe, space, returnKey])
        keys.axis = .horizontal
        keys.spacing = 6
        // Below required: the stack collapses the globe when the device needs none.
        let globeWidth = KeyboardMetrics.width(globe, letters: KeyboardMetrics.functionKey, of: keys)
        globeWidth.priority = .required - 1
        globeWidth.isActive = true
        KeyboardMetrics.width(returnKey, letters: KeyboardMetrics.returnKey, of: keys).isActive = true

        for view in [draftLabel, center, keys, dismiss, typing] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        // The row spans the panel like the typing rows, so key widths derived
        // from it match the typing panel's.
        let keysFill = keys.widthAnchor.constraint(equalTo: widthAnchor, constant: -2 * KeyboardMetrics.sideInset)
        let height = heightAnchor.constraint(equalToConstant: KeyboardMetrics.height)
        height.priority = .defaultHigh
        // The microphone row sits in the middle of the space above the keys;
        // the text being recognized fills the band above it.
        let stage = UILayoutGuide()
        addLayoutGuide(stage)
        NSLayoutConstraint.activate([
            height,
            stage.topAnchor.constraint(equalTo: topAnchor),
            stage.bottomAnchor.constraint(equalTo: keys.topAnchor),
            center.centerXAnchor.constraint(equalTo: centerXAnchor),
            center.centerYAnchor.constraint(equalTo: stage.centerYAnchor),
            draftLabel.topAnchor.constraint(equalTo: topAnchor, constant: 8),
            // The hide key sits where the typing panel has it.
            draftLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 56),
            draftLabel.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -56),
            draftLabel.bottomAnchor.constraint(equalTo: center.topAnchor, constant: -4),
            keys.centerXAnchor.constraint(equalTo: centerXAnchor),
            keysFill,
            keys.heightAnchor.constraint(equalToConstant: KeyboardMetrics.rowHeight),
            keys.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -KeyboardMetrics.bottomInset),
        ])
        dismiss.pin(in: self, trailing: false)
        typing.pin(in: self, trailing: true)
    }

    // MARK: Rendering

    public func render(_ phase: DictationPhase) {
        if phase.isActive, !self.phase.isActive { typingSession = UUID() }
        self.phase = phase
        switch phase {
        case .idle, .failed, .downloading, .downloaded:
            mic.show(.idle)
            configure(leading, icon: nil, title: languageMark, label: L10n.keyboardLanguage(language: languageName))
            configure(trailing, icon: .delete, title: nil, label: L10n.keyboardDelete)
            switch phase {
            case .failed(let failure):
                showNotice(Self.message(failure), autoClear: false)
            case .downloading(let fraction):
                showProgress(L10n.keyboardDownloading(language: languageName, percent: fraction.formatted(.percent.precision(.fractionLength(0)))))
            case .downloaded:
                showNotice(L10n.keyboardDownloaded, autoClear: true)
            default:
                stopTyping()
                if noticeTask == nil { draftLabel.text = nil }
            }
        case .starting:
            mic.show(.waiting)
            setCancelAndTranslate()
            stopTyping()
            draftLabel.text = nil
        case .listening(let draft, let level):
            mic.show(.listening)
            mic.setLevel(level)
            setCancelAndTranslate()
            showDraft(draft, final: false)
        case .finishing(let draft), .polishing(let draft):
            if case .polishing = phase { mic.show(.polishing) } else { mic.show(.waiting) }
            configure(leading, icon: .x, title: nil, label: L10n.keyboardCancel)
            configure(trailing, icon: nil, title: nil, label: "")
            showDraft(draft, final: true)
        }
        if holding { highlightHoldTarget() }
    }

    private func setCancelAndTranslate() {
        configure(leading, icon: .x, title: nil, label: L10n.keyboardCancel)
        configure(trailing, icon: .languages, title: nil, label: L10n.keyboardTranslate(language: translationName))
    }

    /// An empty side button keeps its place in the row, so the microphone
    /// stays centered through every phase.
    private func configure(_ button: FaceButton, icon: Icon?, title: String?, label: String) {
        let hidden = icon == nil && title == nil
        button.alpha = hidden ? 0 : 1
        button.isUserInteractionEnabled = !hidden
        button.accessibilityElementsHidden = hidden
        button.isAccessibilityElement = !hidden
        button.icon = icon
        button.title = title
        button.accessibilityLabel = label
    }

    /// Types the draft out through the typewriter; `final` drains the rest
    /// at full speed once recognition has finished.
    private func showDraft(_ draft: Draft, final: Bool) {
        noticeTask?.cancel()
        noticeTask = nil
        typed = (draft, final)
        draftLabel.font = TextStyle.body.font
        if UIAccessibility.isReduceMotionEnabled {
            paint(Typewriter.Frame(text: draft.text, opacity: [], animating: false), finalized: draft.finalized.count)
            return
        }
        typeFrame()
        guard typingLink == nil, typed != nil else { return }
        let link = CADisplayLink(target: self, selector: #selector(typeFrame))
        link.add(to: .main, forMode: .common)
        typingLink = link
    }

    @objc private func typeFrame() {
        guard let typed else { return stopTyping() }
        let frame = typewriter.update(session: typingSession, text: typed.draft.text, final: typed.final, now: CACurrentMediaTime())
        paint(frame, finalized: typed.draft.finalized.count)
        if !frame.animating { stopTyping(keep: true) }
    }

    /// Settled text in ink and the volatile tail muted; new characters fade in.
    private func paint(_ frame: Typewriter.Frame, finalized: Int) {
        let characters = Array(frame.text)
        let fading = characters.count - frame.opacity.count
        let text = NSMutableAttributedString()
        func add(_ range: Range<Int>, alpha: Double = 1) {
            let split = min(max(finalized, range.lowerBound), range.upperBound)
            for (part, color) in [(range.lowerBound..<split, Palette.foreground), (split..<range.upperBound, Palette.mutedForeground)] where !part.isEmpty {
                text.append(NSAttributedString(string: String(characters[part]),
                                               attributes: [.foregroundColor: color.withAlphaComponent(alpha)]))
            }
        }
        add(0..<fading)
        for (offset, alpha) in frame.opacity.enumerated() {
            add((fading + offset)..<(fading + offset + 1), alpha: alpha)
        }
        draftLabel.attributedText = text
    }

    /// Stops the typing frames; unless kept, the next draft starts over.
    private func stopTyping(keep: Bool = false) {
        typingLink?.invalidate()
        typingLink = nil
        if !keep { typed = nil }
    }

    /// Shows a short line where the draft goes; a notice clears itself.
    public func showNotice(_ message: String, autoClear: Bool) {
        noticeTask?.cancel()
        stopTyping()
        draftLabel.attributedText = nil
        draftLabel.text = message
        draftLabel.font = TextStyle.bodySmall.font
        draftLabel.textColor = Palette.mutedForeground
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

    /// Updates a line that changes often, such as a percentage, without
    /// announcing every change to VoiceOver.
    private func showProgress(_ message: String) {
        let first = draftLabel.text?.isEmpty ?? true
        if first || noticeTask != nil {
            showNotice(message, autoClear: false)
        } else {
            draftLabel.text = message
        }
    }

    static func message(_ failure: DictationFailure) -> String {
        switch failure {
        case .fullAccessRequired: L10n.failureFullAccess
        case .microphoneDenied: L10n.failureMicrophone
        case .languageUnavailable: L10n.failureLanguage
        case .modelMissing, .modelInstalling: L10n.failureInstalling
        case .modelDownloadFailed: L10n.failureDownload
        case .captureEnded: L10n.failureCapture
        case .recognitionFailed: L10n.failureRecognition
        case .nothingHeard: L10n.failureNothingHeard
        }
    }

    // MARK: Gestures

    @objc private func micTapped() {
        feedback.impactOccurred()
        delegate?.keyboardMicTapped()
    }

    @objc private func micHeld(_ gesture: UILongPressGestureRecognizer) {
        switch gesture.state {
        case .began:
            guard !phase.isActive else { return }
            holding = true
            holdTarget = .mic
            feedback.impactOccurred()
            delegate?.keyboardHoldBegan()
        case .changed:
            guard holding else { return }
            let point = gesture.location(in: self)
            let next: HoldTarget = leading.convert(leading.bounds, to: self).insetBy(dx: -16, dy: -24).contains(point) ? .cancel
                : trailing.convert(trailing.bounds, to: self).insetBy(dx: -16, dy: -24).contains(point) ? .translate : .mic
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
            feedback.impactOccurred()
            delegate?.keyboardHoldEnded(holdTarget)
        default:
            break
        }
    }

    private func highlightHoldTarget() {
        leading.isHighlighted = holdTarget == .cancel
        trailing.isHighlighted = holdTarget == .translate
    }

    @objc private func leadingTapped() {
        UIDevice.current.playInputClick()
        delegate?.keyboardLeadingTapped()
    }

    @objc private func trailingDown() {
        switch phase {
        case .starting, .listening:
            feedback.impactOccurred()
            delegate?.keyboardTranslateTapped()
        case .idle, .failed:
            delegate?.keyboardDelete()
            deleteTimer?.invalidate()
            deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.45, repeats: false) { [weak self] _ in
                MainActor.assumeIsolated {
                    self?.deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.08, repeats: true) { _ in
                        MainActor.assumeIsolated { self?.delegate?.keyboardDelete() }
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

    @objc private func spaceTapped() {
        delegate?.keyboardSpace()
    }

    /// Dragging along the space bar moves the caret, as on the system keyboard.
    @objc private func spaceDragged(_ gesture: UIPanGestureRecognizer) {
        switch gesture.state {
        case .began:
            spaceAnchor = 0
        case .changed:
            let step: CGFloat = 9
            let moves = Int((gesture.translation(in: space).x - spaceAnchor) / step)
            if moves != 0 {
                delegate?.keyboardMoveCaret(by: moves)
                spaceAnchor += CGFloat(moves) * step
            }
        default:
            break
        }
    }

    @objc private func returnTapped() {
        delegate?.keyboardReturn()
    }
}
