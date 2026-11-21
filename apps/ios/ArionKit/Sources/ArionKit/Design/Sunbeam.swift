import UIKit

// Sunbeam on UIKit. Tokens come from SunbeamTokens.swift, generated from the
// Sunbeam package the desktop pins. Text uses the system font scaled with
// Dynamic Type at Sunbeam's sizes and weights; headlines use the system serif
// in place of Fraunces.

public extension SunbeamRGBA {
    var uiColor: UIColor {
        UIColor(red: CGFloat(r) / 255, green: CGFloat(g) / 255, blue: CGFloat(b) / 255, alpha: CGFloat(a) / 255)
    }
}

public extension SunbeamColorToken {
    /// A color that follows the trait collection's appearance.
    var color: UIColor {
        let light = light.uiColor
        let dark = dark.uiColor
        return UIColor { $0.userInterfaceStyle == .dark ? dark : light }
    }
}

/// Sunbeam's semantic colors.
public enum Palette {
    public static let background = SunbeamTokens.Color.background.color
    public static let foreground = SunbeamTokens.Color.foreground.color
    public static let card = SunbeamTokens.Color.card.color
    public static let primary = SunbeamTokens.Color.primary.color
    public static let primaryForeground = SunbeamTokens.Color.primaryForeground.color
    public static let secondary = SunbeamTokens.Color.secondary.color
    public static let secondaryForeground = SunbeamTokens.Color.secondaryForeground.color
    public static let muted = SunbeamTokens.Color.muted.color
    public static let mutedForeground = SunbeamTokens.Color.mutedForeground.color
    public static let accent = SunbeamTokens.Color.accent.color
    public static let accentForeground = SunbeamTokens.Color.accentForeground.color
    public static let border = SunbeamTokens.Color.border.color
    public static let faint = SunbeamTokens.Color.faint.color
    public static let success = SunbeamTokens.Color.success.color
    public static let danger = SunbeamTokens.Color.danger.color
    public static let flatFace = SunbeamTokens.Color.flatFace.color
    public static let flatFacePressed = SunbeamTokens.Color.flatFacePressed.color
    public static let raisedFaceFrom = SunbeamTokens.Color.raisedFaceFrom.color
    public static let raisedFaceTo = SunbeamTokens.Color.raisedFaceTo.color
    public static let raisedFacePressed = SunbeamTokens.Color.raisedFacePressed.color
    public static let raisedEdge = SunbeamTokens.Color.raisedEdge.color
    public static let raisedHighlight = SunbeamTokens.Color.raisedHighlight.color
    public static let raisedPrimaryFrom = SunbeamTokens.Color.raisedPrimaryFrom.color
    public static let raisedPrimaryTo = SunbeamTokens.Color.raisedPrimaryTo.color
    public static let raisedPrimaryEdge = SunbeamTokens.Color.raisedPrimaryEdge.color
    public static let wellFace = SunbeamTokens.Color.wellFace.color
    public static let substrate = SunbeamTokens.Color.substrate.color
    public static let substrateSurface = SunbeamTokens.Color.substrateSurface.color
    public static let substrateRaised = SunbeamTokens.Color.substrateRaised.color
    public static let substrateHi = SunbeamTokens.Color.substrateHi.color
    public static let substrateMuted = SunbeamTokens.Color.substrateMuted.color
}

/// Sunbeam's text styles, scaled with Dynamic Type.
public enum TextStyle {
    case headline, title, body, bodySmall, label, caption

    private var spec: (size: CGFloat, weight: UIFont.Weight, textStyle: UIFont.TextStyle, serif: Bool) {
        switch self {
        case .headline: (34, .regular, .largeTitle, true)
        case .title: (24, .medium, .title2, false)
        case .body: (16, .regular, .body, false)
        case .bodySmall: (14, .regular, .subheadline, false)
        case .label: (14, .medium, .subheadline, false)
        case .caption: (13, .regular, .footnote, false)
        }
    }

    public var font: UIFont {
        let s = spec
        var font = UIFont.systemFont(ofSize: s.size, weight: s.weight)
        if s.serif, let descriptor = font.fontDescriptor.withDesign(.serif) {
            font = UIFont(descriptor: descriptor, size: s.size)
        }
        return UIFontMetrics(forTextStyle: s.textStyle).scaledFont(for: font)
    }
}

public extension UILabel {
    /// A label in a Sunbeam text style that wraps and scales with Dynamic Type.
    convenience init(style: TextStyle, color: UIColor = Palette.foreground) {
        self.init()
        font = style.font
        textColor = color
        numberOfLines = 0
        adjustsFontForContentSizeCategory = true
        translatesAutoresizingMaskIntoConstraints = false
    }
}

/// The surface material of a layer: flat on the page, raised on a layer that
/// floats above it, such as the keyboard.
public enum Material {
    case flat, raised
}

/// A button drawn with a Sunbeam face. Primary is the one yellow action.
public final class FaceButton: UIControl {
    public enum Kind { case primary, secondary, plain }

    public let kind: Kind
    public let material: Material
    private let gradient = CAGradientLayer()
    private let highlight = CALayer()
    private let stack = UIStackView()
    public let titleLabel = UILabel(style: .label)
    public let iconView = UIImageView()

    public init(kind: Kind, material: Material = .flat, title: String? = nil, symbol: String? = nil) {
        self.kind = kind
        self.material = material
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        layer.cornerCurve = .continuous
        layer.insertSublayer(gradient, at: 0)
        gradient.cornerCurve = .continuous
        if material == .raised { layer.addSublayer(highlight) }
        titleLabel.numberOfLines = 1
        titleLabel.textAlignment = .center
        iconView.contentMode = .center
        stack.axis = .horizontal
        stack.spacing = 6
        stack.alignment = .center
        stack.isUserInteractionEnabled = false
        stack.translatesAutoresizingMaskIntoConstraints = false
        stack.addArrangedSubview(iconView)
        stack.addArrangedSubview(titleLabel)
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: leadingAnchor, constant: 16),
            heightAnchor.constraint(greaterThanOrEqualToConstant: 44),
        ])
        // Observers do not run inside the initializer, so draw explicitly.
        defer {
            self.title = title
            self.symbol = symbol
            drawSymbol()
        }
        isAccessibilityElement = true
        accessibilityTraits = .button
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: FaceButton, _) in self.paint() }
        paint()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    public var cornerRadius: CGFloat? { didSet { setNeedsLayout() } }

    /// The button's text; nil hides it.
    public var title: String? {
        get { titleLabel.text }
        set {
            titleLabel.text = newValue
            titleLabel.isHidden = newValue == nil
            accessibilityLabel = newValue ?? accessibilityLabel
        }
    }

    /// The button's SF Symbol; nil hides it.
    public var symbol: String? { didSet { drawSymbol() } }

    public var symbolSize: CGFloat = 17 { didSet { drawSymbol() } }

    private func drawSymbol() {
        iconView.image = symbol.map { UIImage(systemName: $0, withConfiguration: UIImage.SymbolConfiguration(pointSize: symbolSize, weight: .medium)) } ?? nil
        iconView.isHidden = symbol == nil
    }

    public override var isHighlighted: Bool {
        didSet {
            guard oldValue != isHighlighted else { return }
            UIView.animate(withDuration: 0.12, delay: 0, options: [.beginFromCurrentState, .allowUserInteraction]) {
                self.transform = self.isHighlighted && !UIAccessibility.isReduceMotionEnabled ? CGAffineTransform(scaleX: 0.97, y: 0.97) : .identity
                self.paint()
            }
        }
    }

    public override var isEnabled: Bool {
        didSet { alpha = isEnabled ? 1 : 0.4 }
    }

    public override func layoutSubviews() {
        super.layoutSubviews()
        let radius = cornerRadius ?? bounds.height / 2
        layer.cornerRadius = radius
        gradient.frame = bounds
        gradient.cornerRadius = radius
        highlight.frame = CGRect(x: radius / 2, y: 0, width: max(0, bounds.width - radius), height: 1)
        if material == .raised {
            layer.shadowPath = UIBezierPath(roundedRect: bounds, cornerRadius: radius).cgPath
        }
    }

    private func paint() {
        let traits = traitCollection
        let resolve = { (c: UIColor) in c.resolvedColor(with: traits).cgColor }
        let ink: UIColor
        switch (kind, material) {
        case (.primary, .flat):
            gradient.colors = [resolve(Palette.primary), resolve(Palette.primary)]
            ink = Palette.primaryForeground
        case (.primary, .raised):
            gradient.colors = [resolve(Palette.raisedPrimaryFrom), resolve(Palette.raisedPrimaryTo)]
            ink = Palette.primaryForeground
        case (.secondary, .flat):
            let face = isHighlighted ? Palette.flatFacePressed : Palette.flatFace
            gradient.colors = [resolve(face), resolve(face)]
            ink = Palette.foreground
        case (.secondary, .raised):
            if isHighlighted {
                gradient.colors = [resolve(Palette.raisedFacePressed), resolve(Palette.raisedFacePressed)]
            } else {
                gradient.colors = [resolve(Palette.raisedFaceFrom), resolve(Palette.raisedFaceTo)]
            }
            ink = Palette.foreground
        case (.plain, _):
            let face = isHighlighted ? Palette.flatFace : UIColor.clear
            gradient.colors = [resolve(face), resolve(face)]
            ink = Palette.foreground
        }
        if kind == .primary && isHighlighted {
            gradient.opacity = 0.88
        } else {
            gradient.opacity = 1
        }
        titleLabel.textColor = ink
        iconView.tintColor = ink
        if material == .raised && kind != .plain {
            layer.borderWidth = 1 / max(traits.displayScale, 1)
            layer.borderColor = resolve(kind == .primary ? Palette.raisedPrimaryEdge : Palette.raisedEdge)
            highlight.backgroundColor = resolve(Palette.raisedHighlight)
            highlight.isHidden = isHighlighted
            let shadow = (traits.userInterfaceStyle == .dark ? SunbeamTokens.Shadow.raisedShadow.dark : SunbeamTokens.Shadow.raisedShadow.light).last
            if let shadow, !isHighlighted {
                layer.shadowColor = shadow.color.uiColor.withAlphaComponent(1).cgColor
                layer.shadowOpacity = Float(shadow.color.a) / 255
                layer.shadowRadius = shadow.blur / 2
                layer.shadowOffset = CGSize(width: shadow.x, height: shadow.y)
            } else {
                layer.shadowOpacity = 0
            }
        }
    }
}
