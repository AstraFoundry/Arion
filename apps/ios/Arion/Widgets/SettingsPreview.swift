import ArionKit
import UIKit

/// The two switches the user turns on in Settings › Arion › Keyboards, shown
/// as they look there: real switches with the system's labels, turning on
/// one after the other. With Reduce Motion they are simply on.
final class SettingsPreview: UIView {
    private let keyboard = UISwitch()
    private let fullAccess = UISwitch()
    private var timer: Timer?

    init() {
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        let header = UILabel(style: .caption, color: Palette.mutedForeground)
        header.text = L10n.onboardingKeyboardPreviewKeyboards
        let card = Card(padding: 0, spacing: 0)
        card.stack.addArrangedSubview(row(L10n.appName, keyboard))
        let line = UIView()
        line.backgroundColor = Palette.border
        line.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
        let inset = UIStackView(arrangedSubviews: [line])
        inset.isLayoutMarginsRelativeArrangement = true
        inset.layoutMargins = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 0)
        card.stack.addArrangedSubview(inset)
        card.stack.addArrangedSubview(row(L10n.onboardingKeyboardPreviewFullAccess, fullAccess))
        let stack = UIStackView(arrangedSubviews: [header, card])
        stack.axis = .vertical
        stack.spacing = 8
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor),
        ])
        // A picture of Settings, not a control: VoiceOver reads it as one image.
        isAccessibilityElement = true
        accessibilityTraits = .image
        accessibilityLabel = "\(L10n.onboardingKeyboardPreviewKeyboards), \(L10n.appName), \(L10n.onboardingKeyboardPreviewFullAccess)"
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    private func row(_ title: String, _ toggle: UISwitch) -> UIView {
        let label = UILabel(style: .body)
        label.text = title
        toggle.isUserInteractionEnabled = false
        // The system's green, as the user will see it in Settings.
        toggle.onTintColor = .systemGreen
        let row = UIStackView(arrangedSubviews: [label, toggle])
        row.alignment = .center
        row.isLayoutMarginsRelativeArrangement = true
        row.layoutMargins = UIEdgeInsets(top: 10, left: 16, bottom: 10, right: 16)
        return row
    }

    override func didMoveToWindow() {
        super.didMoveToWindow()
        timer?.invalidate()
        timer = nil
        guard window != nil else { return }
        guard !UIAccessibility.isReduceMotionEnabled else {
            keyboard.setOn(true, animated: false)
            fullAccess.setOn(true, animated: false)
            return
        }
        play()
        timer = Timer.scheduledTimer(withTimeInterval: 3.6, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { self?.play() }
        }
    }

    /// Off, then the keyboard, then Full Access, then a pause before repeating.
    private func play() {
        keyboard.setOn(false, animated: false)
        fullAccess.setOn(false, animated: false)
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.7) { [weak self] in self?.keyboard.setOn(true, animated: true) }
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { [weak self] in self?.fullAccess.setOn(true, animated: true) }
    }
}
