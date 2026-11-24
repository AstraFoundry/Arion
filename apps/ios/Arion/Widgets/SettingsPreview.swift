import ArionKit
import UIKit

/// What the user will do in the Settings app, drawn as the Settings app: its
/// grouped lists, system colors and switches, Arion's page with the setup
/// note, then the Keyboards page. A touch marker taps Keyboards, the Arion
/// switch and Allow Full Access in turn. Deliberately in the system's look,
/// not Sunbeam's, so it reads as Settings and not as Arion's own screen.
/// With Reduce Motion it shows the finished Keyboards page.
final class SettingsPreview: UIView {
    private static let height: CGFloat = 250
    private let screen = UIView()
    private let appPage = UIView()
    private let keyboardsPage = UIView()
    private let keyboardsRow = UIView()
    private let arionSwitch = UISwitch()
    private let fullAccessSwitch = UISwitch()
    private let touch = UIView()
    private var timer: Timer?
    private var steps: [DispatchWorkItem] = []
    /// The loop starts on the first layout with a width, so the Keyboards
    /// page can wait off screen to the right.
    private var waitingForLayout = false

    init() {
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        heightAnchor.constraint(equalToConstant: Self.height).isActive = true
        screen.backgroundColor = .systemGroupedBackground
        screen.layer.cornerRadius = 28
        screen.layer.cornerCurve = .continuous
        screen.layer.borderWidth = 5
        screen.clipsToBounds = true
        screen.translatesAutoresizingMaskIntoConstraints = false
        addSubview(screen)
        NSLayoutConstraint.activate([
            screen.topAnchor.constraint(equalTo: topAnchor),
            screen.bottomAnchor.constraint(equalTo: bottomAnchor),
            screen.leadingAnchor.constraint(equalTo: leadingAnchor),
            screen.trailingAnchor.constraint(equalTo: trailingAnchor),
        ])
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: SettingsPreview, _) in self.paintFrame() }
        paintFrame()

        build(appPage, title: L10n.appName, back: L10n.onboardingKeyboardPreviewApps, rows: [
            tileRow(.keyboard, .systemBlue, L10n.onboardingKeyboardPreviewKeyboards, trailing: chevron(), into: keyboardsRow),
            tileRow(.mic, .systemOrange, L10n.onboardingKeyboardPreviewMicrophone, trailing: staticSwitch(on: true)),
        ], note: L10n.iosSettingsSetup)
        build(keyboardsPage, title: L10n.onboardingKeyboardPreviewKeyboards, back: L10n.appName, rows: [
            plainRow(L10n.appName, arionSwitch),
            plainRow(L10n.onboardingKeyboardPreviewFullAccess, fullAccessSwitch),
        ], note: nil)
        for page in [appPage, keyboardsPage] {
            page.translatesAutoresizingMaskIntoConstraints = false
            screen.addSubview(page)
            NSLayoutConstraint.activate([
                page.topAnchor.constraint(equalTo: screen.topAnchor),
                page.bottomAnchor.constraint(equalTo: screen.bottomAnchor),
                page.widthAnchor.constraint(equalTo: screen.widthAnchor),
                page.leadingAnchor.constraint(equalTo: screen.leadingAnchor),
            ])
        }
        for toggle in [arionSwitch, fullAccessSwitch] {
            toggle.isUserInteractionEnabled = false
            toggle.onTintColor = .systemGreen
        }
        // A touch: a soft disc where a finger lands, as screen recordings show it.
        touch.bounds = CGRect(x: 0, y: 0, width: 34, height: 34)
        touch.layer.cornerRadius = 17
        touch.backgroundColor = UIColor.label.withAlphaComponent(0.18)
        touch.layer.borderWidth = 2
        touch.layer.borderColor = UIColor.white.withAlphaComponent(0.8).cgColor
        touch.alpha = 0
        screen.addSubview(touch)

        isAccessibilityElement = true
        accessibilityTraits = .image
        accessibilityLabel = [L10n.appName, L10n.onboardingKeyboardPreviewKeyboards, L10n.onboardingKeyboardPreviewFullAccess].joined(separator: ", ")
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    private func paintFrame() {
        screen.layer.borderColor = UIColor.label.withAlphaComponent(0.85).resolvedColor(with: traitCollection).cgColor
    }

    // MARK: Pages

    /// A Settings page: the back button and centered title, the rows on a
    /// grouped card, and an optional footer note.
    private func build(_ page: UIView, title: String, back: String, rows: [UIView], note: String?) {
        page.backgroundColor = .systemGroupedBackground
        let backLabel = UIButton(configuration: {
            var config = UIButton.Configuration.plain()
            config.image = Icon.chevronLeft.image(size: 16)
            config.imagePadding = 2
            config.contentInsets = .zero
            var text = AttributedString(back)
            text.font = .systemFont(ofSize: 14)
            config.attributedTitle = text
            config.baseForegroundColor = .systemBlue
            return config
        }())
        backLabel.isUserInteractionEnabled = false
        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        titleLabel.textColor = .label
        titleLabel.textAlignment = .center
        let card = UIStackView(arrangedSubviews: [])
        card.axis = .vertical
        card.backgroundColor = .secondarySystemGroupedBackground
        card.layer.cornerRadius = 12
        card.layer.cornerCurve = .continuous
        card.clipsToBounds = true
        for (index, row) in rows.enumerated() {
            if index > 0 {
                let line = UIView()
                line.backgroundColor = .separator
                line.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
                let inset = UIStackView(arrangedSubviews: [line])
                inset.isLayoutMarginsRelativeArrangement = true
                inset.layoutMargins = UIEdgeInsets(top: 0, left: index > 0 && row.tag == 1 ? 52 : 14, bottom: 0, right: 0)
                card.addArrangedSubview(inset)
            }
            card.addArrangedSubview(row)
        }
        let column = UIStackView(arrangedSubviews: [card])
        column.axis = .vertical
        column.spacing = 8
        if let note {
            let footer = UILabel()
            footer.text = note
            footer.font = .systemFont(ofSize: 11)
            footer.textColor = .secondaryLabel
            footer.numberOfLines = 0
            let wrap = UIStackView(arrangedSubviews: [footer])
            wrap.isLayoutMarginsRelativeArrangement = true
            wrap.layoutMargins = UIEdgeInsets(top: 0, left: 14, bottom: 0, right: 14)
            column.addArrangedSubview(wrap)
        }
        for view in [backLabel, titleLabel, column] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            page.addSubview(view)
        }
        NSLayoutConstraint.activate([
            backLabel.leadingAnchor.constraint(equalTo: page.leadingAnchor, constant: 14),
            backLabel.topAnchor.constraint(equalTo: page.topAnchor, constant: 22),
            titleLabel.centerXAnchor.constraint(equalTo: page.centerXAnchor),
            titleLabel.centerYAnchor.constraint(equalTo: backLabel.centerYAnchor),
            column.topAnchor.constraint(equalTo: backLabel.bottomAnchor, constant: 22),
            column.leadingAnchor.constraint(equalTo: page.leadingAnchor, constant: 14),
            column.trailingAnchor.constraint(equalTo: page.trailingAnchor, constant: -14),
        ])
    }

    /// A row with a colored icon tile, as Settings shows an app's permissions.
    private func tileRow(_ icon: Icon, _ color: UIColor, _ title: String, trailing: UIView, into row: UIView = UIView()) -> UIView {
        let tile = UIView()
        tile.backgroundColor = color
        tile.layer.cornerRadius = 7
        tile.layer.cornerCurve = .continuous
        let glyph = UIImageView(icon: icon, size: 17, color: .white)
        tile.addSubview(glyph)
        tile.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            tile.widthAnchor.constraint(equalToConstant: 28),
            tile.heightAnchor.constraint(equalToConstant: 28),
            glyph.centerXAnchor.constraint(equalTo: tile.centerXAnchor),
            glyph.centerYAnchor.constraint(equalTo: tile.centerYAnchor),
        ])
        fill(row, [tile, label(title), trailing])
        row.tag = 1
        return row
    }

    private func plainRow(_ title: String, _ toggle: UISwitch) -> UIView {
        let row = UIView()
        fill(row, [label(title), toggle])
        return row
    }

    private func fill(_ row: UIView, _ views: [UIView]) {
        let stack = UIStackView(arrangedSubviews: views)
        stack.alignment = .center
        stack.spacing = 10
        stack.translatesAutoresizingMaskIntoConstraints = false
        row.addSubview(stack)
        NSLayoutConstraint.activate([
            row.heightAnchor.constraint(equalToConstant: 46),
            stack.leadingAnchor.constraint(equalTo: row.leadingAnchor, constant: 14),
            stack.trailingAnchor.constraint(equalTo: row.trailingAnchor, constant: -14),
            stack.centerYAnchor.constraint(equalTo: row.centerYAnchor),
        ])
    }

    private func label(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 15)
        label.textColor = .label
        label.setContentHuggingPriority(.defaultLow, for: .horizontal)
        return label
    }

    private func chevron() -> UIView {
        let view = UIImageView(icon: .chevronRight, size: 14, color: .tertiaryLabel)
        return view
    }

    private func staticSwitch(on: Bool) -> UISwitch {
        let toggle = UISwitch()
        toggle.isOn = on
        toggle.isUserInteractionEnabled = false
        toggle.onTintColor = .systemGreen
        return toggle
    }

    // MARK: Animation

    override func didMoveToWindow() {
        super.didMoveToWindow()
        stop()
        guard window != nil else { return }
        guard !UIAccessibility.isReduceMotionEnabled else {
            showKeyboardsPage(animated: false)
            arionSwitch.setOn(true, animated: false)
            fullAccessSwitch.setOn(true, animated: false)
            return
        }
        waitingForLayout = true
        setNeedsLayout()
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        guard waitingForLayout, bounds.width > 0 else { return }
        waitingForLayout = false
        play()
        timer = Timer.scheduledTimer(withTimeInterval: 6, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { self?.play() }
        }
    }

    private func stop() {
        waitingForLayout = false
        timer?.invalidate()
        timer = nil
        steps.forEach { $0.cancel() }
        steps = []
    }

    /// Arion's page, tap Keyboards, the Keyboards page, tap the Arion switch,
    /// tap Allow Full Access, hold.
    private func play() {
        steps.forEach { $0.cancel() }
        steps = []
        appPage.transform = .identity
        keyboardsPage.transform = CGAffineTransform(translationX: bounds.width, y: 0)
        arionSwitch.setOn(false, animated: false)
        fullAccessSwitch.setOn(false, animated: false)
        at(0.8) { self.tap(self.keyboardsRow, x: 0.3) }
        at(1.3) { self.showKeyboardsPage(animated: true) }
        at(2.2) { self.tap(self.arionSwitch.superview!.superview!, x: 0.88) }
        at(2.4) { self.arionSwitch.setOn(true, animated: true) }
        at(3.1) { self.tap(self.fullAccessSwitch.superview!.superview!, x: 0.88) }
        at(3.3) { self.fullAccessSwitch.setOn(true, animated: true) }
    }

    private func at(_ seconds: Double, _ work: @escaping @MainActor () -> Void) {
        let item = DispatchWorkItem { MainActor.assumeIsolated { work() } }
        steps.append(item)
        DispatchQueue.main.asyncAfter(deadline: .now() + seconds, execute: item)
    }

    private func showKeyboardsPage(animated: Bool) {
        let width = max(bounds.width, 1)
        let change = {
            self.appPage.transform = CGAffineTransform(translationX: -width * 0.3, y: 0)
            self.keyboardsPage.transform = .identity
        }
        if animated {
            UIView.animate(withDuration: 0.45, delay: 0, usingSpringWithDamping: 1, initialSpringVelocity: 0, animations: change)
        } else {
            change()
        }
    }

    /// The touch marker lands on a row at a fraction of its width and lifts.
    private func tap(_ target: UIView, x fraction: CGFloat) {
        let frame = target.convert(target.bounds, to: screen)
        touch.center = CGPoint(x: frame.minX + frame.width * fraction, y: frame.midY)
        touch.transform = CGAffineTransform(scaleX: 1.3, y: 1.3)
        UIView.animate(withDuration: 0.18, animations: {
            self.touch.alpha = 1
            self.touch.transform = .identity
        }, completion: { _ in
            UIView.animate(withDuration: 0.25, delay: 0.15, animations: { self.touch.alpha = 0 })
        })
    }
}
