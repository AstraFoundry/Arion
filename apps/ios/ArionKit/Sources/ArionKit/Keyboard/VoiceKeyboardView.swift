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

    /// The mark of the current language, shown when there is more than one.
    public var languageMark: String? { didSet { render(phase) } }
    public var languageName = "" { didSet { render(phase) } }
    public var translationName = "" { didSet { render(phase) } }
    public var returnTitle = L10n.returnDefault { didSet { returnKey.title = returnTitle } }

    public static let height: CGFloat = 236

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
        returnKey.title = returnTitle
        returnKey.cornerRadius = SunbeamTokens.Radius.lg
        returnKey.addTarget(self, action: #selector(returnTapped), for: .touchUpInside)

        let center = UIStackView(arrangedSubviews: [leading, mic, trailing])
        center.axis = .horizontal
        center.alignment = .center
        center.spacing = 40

        let keys = UIStackView(arrangedSubviews: [globe, space, returnKey])
        keys.axis = .horizontal
        keys.spacing = 6
        globe.widthAnchor.constraint(equalToConstant: 48).isActive = true
        returnKey.widthAnchor.constraint(equalToConstant: 92).isActive = true

        for view in [draftLabel, center, keys] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        // On wide screens the keys keep a phone's reach.
        let keysWidth = keys.widthAnchor.constraint(lessThanOrEqualToConstant: 640)
        let keysFill = keys.widthAnchor.constraint(equalTo: widthAnchor, constant: -12)
        keysFill.priority = .defaultHigh
        let height = heightAnchor.constraint(equalToConstant: Self.height)
        height.priority = .defaultHigh
        NSLayoutConstraint.activate([
            height,
            draftLabel.topAnchor.constraint(equalTo: topAnchor, constant: 10),
            draftLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 20),
            draftLabel.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -20),
            draftLabel.heightAnchor.constraint(equalToConstant: 64),
            center.centerXAnchor.constraint(equalTo: centerXAnchor),
            center.topAnchor.constraint(equalTo: draftLabel.bottomAnchor, constant: 6),
            keys.topAnchor.constraint(equalTo: center.bottomAnchor, constant: 14),
            keys.centerXAnchor.constraint(equalTo: centerXAnchor),
            keysWidth, keysFill,
            keys.heightAnchor.constraint(equalToConstant: 44),
            keys.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -6),
        ])
    }

    // MARK: Rendering

    public func render(_ phase: DictationPhase) {
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
                if noticeTask == nil { draftLabel.text = nil }
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
        case .finishing(let draft), .polishing(let draft):
            mic.show(.working)
            configure(leading, icon: .x, title: nil, label: L10n.keyboardCancel)
            trailing.isHidden = true
            showDraft(draft)
        }
        if holding { highlightHoldTarget() }
    }

    private func setCancelAndTranslate() {
        configure(leading, icon: .x, title: nil, label: L10n.keyboardCancel)
        configure(trailing, icon: .languages, title: nil, label: L10n.keyboardTranslate(language: translationName))
    }

    private func configure(_ button: FaceButton, icon: Icon?, title: String?, label: String) {
        let hidden = icon == nil && title == nil
        if button.isHidden != hidden { button.isHidden = hidden }
        button.icon = icon
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

    /// Shows a short line where the draft goes; a notice clears itself.
    public func showNotice(_ message: String, autoClear: Bool) {
        noticeTask?.cancel()
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
