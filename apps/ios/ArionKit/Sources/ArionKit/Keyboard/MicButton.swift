
import UIKit

/// The keyboard's one primary control, drawn as the app icon: Sunbeam's raised
/// yellow face with the voice mark. Listening it turns to the substrate ink
/// and its bars follow the voice; finishing, the bars rest and dim.
@MainActor
final class MicButton: UIControl {
    enum Look: Equatable { case idle, listening, working }

    static let diameter: CGFloat = 72

    private let face = FaceButton(kind: .primary, material: .raised)
    private let dark = UIView()
    private let mark = VoiceMark()
    private(set) var look: Look = .idle

    override init(frame: CGRect) {
        super.init(frame: frame)
        translatesAutoresizingMaskIntoConstraints = false
        isAccessibilityElement = true
        accessibilityTraits = .button
        accessibilityLabel = L10n.keyboardDictate

        face.isUserInteractionEnabled = false
        // Ink on a light keyboard; on a dark one the raised dark face, which
        // stays visible where the substrate would disappear.
        let substrate = SunbeamTokens.Color.substrate.light.uiColor
        let raised = SunbeamTokens.Color.raisedFaceFrom.dark.uiColor
        dark.backgroundColor = UIColor { $0.userInterfaceStyle == .dark ? raised : substrate }
        dark.isUserInteractionEnabled = false
        dark.alpha = 0
        mark.color = Palette.primaryForeground
        for view in [face, dark, mark] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        NSLayoutConstraint.activate([
            widthAnchor.constraint(equalToConstant: Self.diameter),
            heightAnchor.constraint(equalToConstant: Self.diameter),
            face.topAnchor.constraint(equalTo: topAnchor),
            face.bottomAnchor.constraint(equalTo: bottomAnchor),
            face.leadingAnchor.constraint(equalTo: leadingAnchor),
            face.trailingAnchor.constraint(equalTo: trailingAnchor),
            dark.topAnchor.constraint(equalTo: topAnchor),
            dark.bottomAnchor.constraint(equalTo: bottomAnchor),
            dark.leadingAnchor.constraint(equalTo: leadingAnchor),
            dark.trailingAnchor.constraint(equalTo: trailingAnchor),
            mark.centerXAnchor.constraint(equalTo: centerXAnchor),
            mark.centerYAnchor.constraint(equalTo: centerYAnchor),
            mark.widthAnchor.constraint(equalToConstant: 36),
            mark.heightAnchor.constraint(equalToConstant: 30),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func layoutSubviews() {
        super.layoutSubviews()
        dark.layer.cornerRadius = bounds.width / 2
        dark.layer.cornerCurve = .continuous
    }

    override var isHighlighted: Bool {
        didSet { if look == .idle { face.isHighlighted = isHighlighted } }
    }

    func show(_ look: Look) {
        guard look != self.look else { return }
        self.look = look
        accessibilityLabel = look == .idle ? L10n.keyboardDictate : L10n.keyboardDone
        mark.setLevel(0)
        let animations = {
            self.face.alpha = look == .idle ? 1 : 0
            self.dark.alpha = look == .idle ? 0 : 1
            self.mark.color = look == .idle ? Palette.primaryForeground : Palette.primary
            self.mark.alpha = look == .working ? 0.45 : 1
        }
        if UIAccessibility.isReduceMotionEnabled {
            animations()
        } else {
            UIView.animate(withDuration: 0.2, delay: 0, options: [.beginFromCurrentState], animations: animations)
        }
        if look == .working { pulse() } else { mark.layer.removeAnimation(forKey: "pulse") }
    }

    /// Follows the voice level, 0...1.
    func setLevel(_ level: Float) {
        guard look == .listening else { return }
        mark.setLevel(max(level, 0.08))
    }

    private func pulse() {
        guard !UIAccessibility.isReduceMotionEnabled else { return }
        let animation = CABasicAnimation(keyPath: "opacity")
        animation.fromValue = 0.35
        animation.toValue = 0.8
        animation.duration = 0.7
        animation.autoreverses = true
        animation.repeatCount = .infinity
        mark.layer.add(animation, forKey: "pulse")
    }
}
