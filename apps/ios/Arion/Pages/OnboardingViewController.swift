import ArionKit
import UIKit

/// First-run setup: what Arion does, the microphone, adding the keyboard, and
/// a first dictation. Each step asks for one thing.
final class OnboardingViewController: UIViewController {
    enum Step: CaseIterable { case welcome, microphone, keyboard, practice }

    private let preferences = Preferences()
    private var step: Step
    private let content = UIStackView()
    private let primary = FaceButton(kind: .primary)
    private let secondary = FaceButton(kind: .plain)
    private var openedSettings = false
    private var bottom: NSLayoutConstraint?
    private weak var practiceField: UITextView?

    init(start: Step = .welcome) {
        step = start
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = Palette.background
        content.axis = .vertical
        content.spacing = 12
        content.translatesAutoresizingMaskIntoConstraints = false
        primary.heightAnchor.constraint(equalToConstant: 52).isActive = true
        primary.addTarget(self, action: #selector(primaryTapped), for: .touchUpInside)
        secondary.addTarget(self, action: #selector(secondaryTapped), for: .touchUpInside)
        let actions = UIStackView(arrangedSubviews: [primary, secondary])
        actions.axis = .vertical
        actions.spacing = 4
        actions.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(content)
        view.addSubview(actions)
        let guide = view.safeAreaLayoutGuide
        let bottom = actions.bottomAnchor.constraint(equalTo: view.keyboardLayoutGuide.topAnchor, constant: -16)
        self.bottom = bottom
        NSLayoutConstraint.activate([
            content.topAnchor.constraint(equalTo: guide.topAnchor, constant: 56),
            content.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 24),
            content.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -24),
            content.bottomAnchor.constraint(lessThanOrEqualTo: actions.topAnchor, constant: -16),
            actions.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 24),
            actions.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -24),
            bottom,
        ])
        NotificationCenter.default.addObserver(self, selector: #selector(becameActive), name: UIApplication.didBecomeActiveNotification, object: nil)
        show(step, animated: false)
    }

    private func show(_ step: Step, animated: Bool) {
        self.step = step
        let build = {
            self.content.arrangedSubviews.forEach { $0.removeFromSuperview() }
            self.build(step)
        }
        let focus = { if step == .practice { self.practiceField?.becomeFirstResponder() } }
        guard animated, !UIAccessibility.isReduceMotionEnabled else {
            build()
            DispatchQueue.main.async(execute: focus)
            return
        }
        UIView.transition(with: content, duration: 0.24, options: [.transitionCrossDissolve, .allowUserInteraction], animations: build) { _ in focus() }
    }

    private func build(_ step: Step) {
        var showsSecondary = false
        defer {
            // A stack view miscounts repeated isHidden writes inside animations.
            if secondary.isHidden == showsSecondary { secondary.isHidden = !showsSecondary }
        }
        switch step {
        case .welcome:
            content.addArrangedSubview(AppMark())
            content.setCustomSpacing(28, after: content.arrangedSubviews.last!)
            addTitle(L10n.onboardingWelcomeTitle, headline: true)
            addBody(L10n.onboardingWelcomeBody)
            primary.title = L10n.onboardingContinue
        case .microphone:
            addTitle(L10n.onboardingMicrophoneTitle)
            addBody(Permissions.microphoneDenied ? L10n.onboardingMicrophoneDenied : L10n.onboardingMicrophoneBody)
            primary.title = Permissions.microphoneDenied ? L10n.onboardingKeyboardOpen : L10n.onboardingMicrophoneAllow
        case .keyboard:
            addTitle(L10n.onboardingKeyboardTitle)
            for (index, text) in [L10n.onboardingKeyboardStep1, L10n.onboardingKeyboardStep2, L10n.onboardingKeyboardStep3].enumerated() {
                content.addArrangedSubview(StepRow(number: index + 1, text: text))
            }
            content.setCustomSpacing(20, after: content.arrangedSubviews.last!)
            addBody(L10n.onboardingKeyboardFullAccessNote, muted: true, style: .bodySmall)
            primary.title = openedSettings ? L10n.onboardingContinue : L10n.onboardingKeyboardOpen
            secondary.title = L10n.onboardingContinue
            showsSecondary = !openedSettings
        case .practice:
            addTitle(L10n.onboardingTryTitle)
            addBody(L10n.onboardingTryBody)
            let field = PracticeField(placeholder: L10n.onboardingTryPlaceholder)
            field.heightAnchor.constraint(equalToConstant: 140).isActive = true
            content.setCustomSpacing(20, after: content.arrangedSubviews.last!)
            content.addArrangedSubview(field)
            practiceField = field.textView
            primary.title = L10n.onboardingFinish
        }
    }

    private func addTitle(_ text: String, headline: Bool = false) {
        let label = UILabel(style: headline ? .headline : .title)
        label.text = text
        label.accessibilityTraits = .header
        content.addArrangedSubview(label)
        content.setCustomSpacing(headline ? 16 : 12, after: label)
    }

    private func addBody(_ text: String, muted: Bool = false, style: TextStyle = .body) {
        let label = UILabel(style: style, color: muted ? Palette.mutedForeground : Palette.secondaryForeground)
        label.text = text
        content.addArrangedSubview(label)
    }

    @objc private func primaryTapped() {
        switch step {
        case .welcome:
            show(.microphone, animated: true)
        case .microphone:
            if Permissions.microphoneDenied {
                openSettings()
                return
            }
            Task {
                _ = await Permissions.request()
                // The speech models download while the user adds the keyboard.
                for language in preferences.languages {
                    Task.detached { try? await Recognizer.install(language) }
                }
                show(Permissions.microphoneGranted ? .keyboard : .microphone, animated: true)
            }
        case .keyboard:
            if openedSettings {
                show(.practice, animated: true)
            } else {
                openedSettings = true
                openSettings()
            }
        case .practice:
            preferences.onboardingCompleted = true
            practiceField?.resignFirstResponder()
            dismiss(animated: true)
        }
    }

    @objc private func secondaryTapped() {
        show(.practice, animated: true)
    }

    @objc private func becameActive() {
        if step == .microphone || step == .keyboard { show(step, animated: false) }
    }

    private func openSettings() {
        guard let url = URL(string: UIApplication.openSettingsURLString) else { return }
        UIApplication.shared.open(url)
    }
}

/// The app icon in miniature: the voice mark on Sunbeam yellow.
final class AppMark: UIView {
    private let mark = VoiceMark()

    init() {
        super.init(frame: .zero)
        let tile = UIView()
        tile.backgroundColor = Palette.primary
        tile.layer.cornerRadius = 16
        tile.layer.cornerCurve = .continuous
        mark.color = Palette.primaryForeground
        for view in [tile, mark] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        NSLayoutConstraint.activate([
            heightAnchor.constraint(equalToConstant: 64),
            tile.leadingAnchor.constraint(equalTo: leadingAnchor),
            tile.topAnchor.constraint(equalTo: topAnchor),
            tile.widthAnchor.constraint(equalToConstant: 64),
            tile.heightAnchor.constraint(equalToConstant: 64),
            mark.centerXAnchor.constraint(equalTo: tile.centerXAnchor),
            mark.centerYAnchor.constraint(equalTo: tile.centerYAnchor),
            mark.widthAnchor.constraint(equalToConstant: 38),
            mark.heightAnchor.constraint(equalToConstant: 34),
        ])
        isAccessibilityElement = false
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}

/// A numbered instruction.
final class StepRow: UIView {
    init(number: Int, text: String) {
        super.init(frame: .zero)
        let badge = UILabel(style: .label, color: Palette.foreground)
        badge.text = String(number)
        badge.textAlignment = .center
        badge.backgroundColor = Palette.secondary
        badge.layer.cornerRadius = 14
        badge.layer.cornerCurve = .continuous
        badge.clipsToBounds = true
        let label = UILabel(style: .body)
        label.text = text
        addSubview(badge)
        addSubview(label)
        NSLayoutConstraint.activate([
            badge.widthAnchor.constraint(equalToConstant: 28),
            badge.heightAnchor.constraint(equalToConstant: 28),
            badge.leadingAnchor.constraint(equalTo: leadingAnchor),
            badge.centerYAnchor.constraint(equalTo: label.centerYAnchor),
            label.leadingAnchor.constraint(equalTo: badge.trailingAnchor, constant: 14),
            label.trailingAnchor.constraint(equalTo: trailingAnchor),
            label.topAnchor.constraint(equalTo: topAnchor, constant: 6),
            label.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -6),
        ])
        isAccessibilityElement = true
        accessibilityLabel = "\(number). \(text)"
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}

/// A flat card holding a text view to dictate into.
final class PracticeField: UIView, UITextViewDelegate {
    let textView = UITextView()
    private let placeholder = UILabel(style: .body, color: Palette.faint)

    private let bordered: Bool

    init(placeholder text: String, bordered: Bool = true) {
        self.bordered = bordered
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        backgroundColor = Palette.card
        layer.cornerRadius = SunbeamTokens.Radius.xl2
        layer.cornerCurve = .continuous
        layer.borderColor = Palette.border.cgColor
        textView.font = TextStyle.body.font
        textView.adjustsFontForContentSizeCategory = true
        textView.textColor = Palette.foreground
        textView.backgroundColor = .clear
        textView.textContainerInset = UIEdgeInsets(top: 14, left: 12, bottom: 14, right: 12)
        textView.delegate = self
        textView.translatesAutoresizingMaskIntoConstraints = false
        textView.accessibilityLabel = text
        placeholder.text = text
        placeholder.isAccessibilityElement = false
        addSubview(textView)
        addSubview(placeholder)
        NSLayoutConstraint.activate([
            textView.topAnchor.constraint(equalTo: topAnchor),
            textView.bottomAnchor.constraint(equalTo: bottomAnchor),
            textView.leadingAnchor.constraint(equalTo: leadingAnchor),
            textView.trailingAnchor.constraint(equalTo: trailingAnchor),
            placeholder.topAnchor.constraint(equalTo: topAnchor, constant: 14),
            placeholder.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 17),
            placeholder.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor, constant: -16),
        ])
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: PracticeField, _) in
            self.layer.borderColor = Palette.border.resolvedColor(with: self.traitCollection).cgColor
        }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func layoutSubviews() {
        super.layoutSubviews()
        layer.borderWidth = bordered ? 1 / max(traitCollection.displayScale, 1) : 0
    }

    func textViewDidChange(_ textView: UITextView) {
        placeholder.isHidden = !textView.text.isEmpty
    }
}
