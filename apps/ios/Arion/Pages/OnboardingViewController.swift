import ArionKit
import UIKit

/// First-run setup, one decision per step with its progress shown: what
/// Arion does with the optional account, the microphone, adding the
/// keyboard, polishing, and a first dictation.
final class OnboardingViewController: UIViewController {
    enum Step: Int, CaseIterable { case welcome, microphone, keyboard, polish, practice }

    private let preferences = Preferences()
    private var step: Step
    private let scroll = UIScrollView()
    private let content = UIStackView()
    private let progress = StepProgress(count: Step.allCases.count)
    private let back = UIButton(type: .system)
    private let primary = FaceButton(kind: .primary)
    private let secondary = FaceButton(kind: .plain)
    private var openedSettings = false
    private weak var practiceField: UITextView?
    private var terms = true
    private var shareUsage = false
    private var polish: PolishChoice

    init(start: Step = .welcome) {
        step = start
        polish = PolishChoice.current(Preferences())
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = Palette.background
        back.setImage(Icon.chevronLeft.image(size: 24), for: .normal)
        back.tintColor = Palette.foreground
        back.accessibilityLabel = L10n.onboardingBack
        back.addAction(UIAction { [weak self] _ in self?.goBack() }, for: .touchUpInside)
        let top = UIStackView(arrangedSubviews: [back, progress])
        top.spacing = 12
        top.alignment = .center
        top.translatesAutoresizingMaskIntoConstraints = false
        back.widthAnchor.constraint(equalToConstant: 28).isActive = true
        content.axis = .vertical
        content.spacing = 12
        content.translatesAutoresizingMaskIntoConstraints = false
        scroll.translatesAutoresizingMaskIntoConstraints = false
        scroll.keyboardDismissMode = .interactive
        scroll.addSubview(content)
        primary.heightAnchor.constraint(equalToConstant: 52).isActive = true
        secondary.heightAnchor.constraint(equalToConstant: 44).isActive = true
        primary.addTarget(self, action: #selector(primaryTapped), for: .touchUpInside)
        secondary.addTarget(self, action: #selector(secondaryTapped), for: .touchUpInside)
        let actions = UIStackView(arrangedSubviews: [primary, secondary])
        actions.axis = .vertical
        actions.spacing = 4
        actions.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(top)
        view.addSubview(scroll)
        view.addSubview(actions)
        let guide = view.safeAreaLayoutGuide
        NSLayoutConstraint.activate([
            top.topAnchor.constraint(equalTo: guide.topAnchor, constant: 8),
            top.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 16),
            top.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -24),
            top.heightAnchor.constraint(equalToConstant: 44),
            scroll.topAnchor.constraint(equalTo: top.bottomAnchor),
            scroll.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scroll.bottomAnchor.constraint(equalTo: actions.topAnchor, constant: -12),
            content.topAnchor.constraint(equalTo: scroll.contentLayoutGuide.topAnchor, constant: 20),
            content.bottomAnchor.constraint(equalTo: scroll.contentLayoutGuide.bottomAnchor, constant: -8),
            content.leadingAnchor.constraint(equalTo: scroll.frameLayoutGuide.leadingAnchor, constant: 24),
            content.trailingAnchor.constraint(equalTo: scroll.frameLayoutGuide.trailingAnchor, constant: -24),
            actions.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 24),
            actions.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -24),
            actions.bottomAnchor.constraint(equalTo: view.keyboardLayoutGuide.topAnchor, constant: -16),
        ])
        NotificationCenter.default.addObserver(self, selector: #selector(becameActive), name: UIApplication.didBecomeActiveNotification, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(accountChanged), name: .accountChanged, object: nil)
        show(step, animated: false)
    }

    private func show(_ step: Step, animated: Bool, direction: CGFloat = 1) {
        let moved = step != self.step
        self.step = step
        progress.current = step.rawValue
        back.alpha = step == .welcome ? 0 : 1
        back.isEnabled = step != .welcome
        let build = {
            self.content.arrangedSubviews.forEach { $0.removeFromSuperview() }
            self.build(step)
        }
        let focus = { if step == .practice { self.practiceField?.becomeFirstResponder() } }
        guard animated, moved, !UIAccessibility.isReduceMotionEnabled else {
            build()
            DispatchQueue.main.async(execute: focus)
            return
        }
        // The next step slides in from the side it lies on.
        UIView.animate(withDuration: 0.12, animations: {
            self.content.alpha = 0
            self.content.transform = CGAffineTransform(translationX: -20 * direction, y: 0)
        }, completion: { _ in
            build()
            self.scroll.contentOffset = .zero
            self.content.transform = CGAffineTransform(translationX: 28 * direction, y: 0)
            UIView.animate(withDuration: 0.36, delay: 0, usingSpringWithDamping: 1, initialSpringVelocity: 0, options: [.allowUserInteraction], animations: {
                self.content.alpha = 1
                self.content.transform = .identity
            }, completion: { _ in focus() })
        })
        UIAccessibility.post(notification: .screenChanged, argument: content)
    }

    private func build(_ step: Step) {
        var showsSecondary = false
        var showsPrimary = true
        defer {
            // A stack view miscounts repeated isHidden writes inside animations.
            if secondary.isHidden == showsSecondary { secondary.isHidden = !showsSecondary }
            if primary.isHidden == showsPrimary { primary.isHidden = !showsPrimary }
        }
        switch step {
        case .welcome:
            content.addArrangedSubview(AppMark())
            content.setCustomSpacing(28, after: content.arrangedSubviews.last!)
            addTitle(L10n.onboardingWelcomeTitle, headline: true)
            addBody(L10n.onboardingWelcomeBody)
            content.setCustomSpacing(32, after: content.arrangedSubviews.last!)
            let account = Services.account.state
            if account.signedIn || account.phase == .restoring {
                content.addArrangedSubview(AccountSummary(state: account))
                primary.title = L10n.onboardingContinue
            } else {
                addSignIn(waiting: account.phase == .signingIn, failure: account.failure)
                showsPrimary = false
                secondary.title = L10n.accountContinueLocally
                showsSecondary = true
            }
        case .microphone:
            addTitle(L10n.onboardingMicrophoneTitle)
            if Permissions.microphoneDenied { addBody(L10n.onboardingMicrophoneDenied) }
            primary.title = Permissions.microphoneDenied ? L10n.onboardingKeyboardOpen : L10n.onboardingMicrophoneAllow
        case .keyboard:
            addTitle(L10n.onboardingKeyboardTitle)
            // The steps are on Arion's page in Settings (Settings.bundle),
            // where the user follows them; this step only sends them there.
            addBody(L10n.onboardingKeyboardBody)
            content.setCustomSpacing(28, after: content.arrangedSubviews.last!)
            content.addArrangedSubview(SettingsPreview())
            // Back from Settings, continuing comes first; Settings stays one tap away.
            primary.title = openedSettings ? L10n.onboardingContinue : L10n.onboardingKeyboardOpen
            secondary.title = openedSettings ? L10n.onboardingKeyboardOpen : L10n.onboardingContinue
            showsSecondary = true
        case .polish:
            addTitle(L10n.onboardingPolishTitle)
            content.setCustomSpacing(24, after: content.arrangedSubviews.last!)
            let card = PolishChoiceCard(selected: polish)
            card.onChoose = { [weak self] in self?.polish = $0 }
            content.addArrangedSubview(card)
            primary.title = L10n.onboardingContinue
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

    /// Sign-in methods with the terms (checked) and usage sharing
    /// (unchecked) choices below them, as the desktop's welcome page.
    private func addSignIn(waiting: Bool, failure: AccountFailure?) {
        let title = UILabel(style: .label)
        title.text = L10n.accountSignInTitle
        content.addArrangedSubview(title)
        content.setCustomSpacing(12, after: title)
        if waiting {
            addBody(L10n.accountWaiting, style: .bodySmall)
            let cancel = OutlineButton(title: L10n.accountCancel, image: nil)
            cancel.addAction(UIAction { _ in Services.account.cancel() }, for: .touchUpInside)
            content.addArrangedSubview(cancel)
            return
        }
        let methods = SignInMethods { [weak self] in
            guard let self else { return }
            Services.account.signIn(shareUsage: self.shareUsage)
        }
        methods.isEnabled = terms
        content.addArrangedSubview(methods)
        content.setCustomSpacing(16, after: methods)
        let accept = ConsentRow(label: L10n.accountTermsAccept, links: [(L10n.accountTerms, SiteLinks.terms), (L10n.accountPrivacy, SiteLinks.privacy)], checked: terms)
        accept.onChange = { [weak self, weak methods] checked in
            self?.terms = checked
            methods?.isEnabled = checked
        }
        let share = ConsentRow(label: L10n.accountUsageAgree, links: [(L10n.accountUsageData, SiteLinks.usageData)], checked: shareUsage)
        share.onChange = { [weak self] in self?.shareUsage = $0 }
        content.addArrangedSubview(accept)
        content.setCustomSpacing(10, after: accept)
        content.addArrangedSubview(share)
        if let failure {
            let label = UILabel(style: .caption, color: Palette.danger)
            label.text = AccountCopy.failure(failure)
            content.setCustomSpacing(12, after: share)
            content.addArrangedSubview(label)
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

    private func move(to next: Step) {
        show(next, animated: true, direction: next.rawValue >= step.rawValue ? 1 : -1)
    }

    private func goBack() {
        guard let previous = Step(rawValue: step.rawValue - 1) else { return }
        move(to: previous)
    }

    @objc private func primaryTapped() {
        switch step {
        case .welcome:
            move(to: .microphone)
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
                if Permissions.microphoneGranted { move(to: .keyboard) } else { show(.microphone, animated: false) }
            }
        case .keyboard:
            if openedSettings {
                move(to: .polish)
            } else {
                openedSettings = true
                openSettings()
            }
        case .polish:
            polish.apply(preferences)
            guard polish == .cloud, Services.cloud.active?.model == nil else {
                move(to: .practice)
                return
            }
            // Cloud polishing needs a provider; connecting one moves on.
            let cloud = CloudViewController { [weak self] in
                self?.dismiss(animated: true) { self?.move(to: .practice) }
            }
            let sheet = UINavigationController(rootViewController: cloud)
            cloud.navigationItem.leftBarButtonItem = UIBarButtonItem(title: L10n.onboardingSkipCloud, primaryAction: UIAction { [weak self] _ in
                self?.dismiss(animated: true) { self?.move(to: .practice) }
            })
            present(sheet, animated: true)
        case .practice:
            preferences.onboardingCompleted = true
            practiceField?.resignFirstResponder()
            dismiss(animated: true)
        }
    }

    @objc private func secondaryTapped() {
        switch step {
        case .welcome: move(to: .microphone)
        case .keyboard where openedSettings: openSettings()
        default: move(to: .polish)
        }
    }

    @objc private func becameActive() {
        if step == .microphone || step == .keyboard { show(step, animated: false) }
    }

    @objc private func accountChanged() {
        if step == .welcome { show(.welcome, animated: false) }
    }

    private func openSettings() {
        guard let url = URL(string: UIApplication.openSettingsURLString) else { return }
        UIApplication.shared.open(url)
    }
}

/// Segments for the steps of a flow; the current and earlier ones in ink.
final class StepProgress: UIStackView {
    private let count: Int

    var current = 0 {
        didSet { paint() }
    }

    init(count: Int) {
        self.count = count
        super.init(frame: .zero)
        spacing = 6
        distribution = .fillEqually
        alignment = .center
        for _ in 0..<count {
            let segment = UIView()
            segment.layer.cornerRadius = 2
            segment.heightAnchor.constraint(equalToConstant: 4).isActive = true
            addArrangedSubview(segment)
        }
        isAccessibilityElement = true
        accessibilityTraits = .updatesFrequently
        paint()
    }

    @available(*, unavailable)
    required init(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    private func paint() {
        for (index, segment) in arrangedSubviews.enumerated() {
            segment.backgroundColor = index <= current ? Palette.foreground : Palette.border
        }
        accessibilityLabel = L10n.onboardingProgress(current: String(current + 1), total: String(count))
    }
}

/// The signed-in account on the welcome step.
final class AccountSummary: Card {
    init(state: AccountState) {
        super.init(padding: 14, spacing: 0)
        let avatar = AvatarView(size: 40)
        avatar.show(state.profile)
        let name = UILabel(style: .body)
        name.text = state.profile.map { $0.name.isEmpty ? $0.email : $0.name }
        let detail = UILabel(style: .caption, color: Palette.mutedForeground)
        detail.text = state.phase == .restoring ? L10n.accountRestoring : L10n.accountSignedIn
        let labels = UIStackView(arrangedSubviews: [name, detail])
        labels.axis = .vertical
        labels.spacing = 2
        let row = UIStackView(arrangedSubviews: [avatar, labels])
        row.spacing = 12
        row.alignment = .center
        stack.addArrangedSubview(row)
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
