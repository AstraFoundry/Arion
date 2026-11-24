import ArionKit
import UIKit

/// A flat Sunbeam card: white on the page, a hairline edge, continuous corners.
class Card: UIView {
    let stack = UIStackView()

    init(padding: CGFloat = 16, spacing: CGFloat = 12) {
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        backgroundColor = Palette.card
        layer.cornerRadius = SunbeamTokens.Radius.xl2
        layer.cornerCurve = .continuous
        stack.axis = .vertical
        stack.spacing = spacing
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor, constant: padding),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -padding),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: padding),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -padding),
        ])
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: Card, _) in self.paintEdge() }
        paintEdge()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func layoutSubviews() {
        super.layoutSubviews()
        layer.borderWidth = 1 / max(traitCollection.displayScale, 1)
    }

    private func paintEdge() {
        layer.borderColor = Palette.border.resolvedColor(with: traitCollection).cgColor
    }

    /// A header row: the title, and an optional control at the trailing edge.
    func header(_ title: String, trailing: UIView? = nil) {
        let label = UILabel(style: .label)
        label.text = title
        label.accessibilityTraits = .header
        label.setContentHuggingPriority(.defaultLow, for: .horizontal)
        let row = UIStackView(arrangedSubviews: [label] + (trailing.map { [$0] } ?? []))
        row.alignment = .center
        row.spacing = 8
        stack.addArrangedSubview(row)
    }
}

/// Cards rise into place one after another when a page is entered; with
/// Reduce Motion they only fade in.
enum Entrance {
    static func play(_ views: [UIView]) {
        let reduce = UIAccessibility.isReduceMotionEnabled
        for (index, view) in views.enumerated() {
            view.alpha = 0
            view.transform = reduce ? .identity : CGAffineTransform(translationX: 0, y: 8)
            UIView.animate(withDuration: 0.52, delay: reduce ? 0 : Double(index) * 0.045, usingSpringWithDamping: 1, initialSpringVelocity: 0, options: [.allowUserInteraction]) {
                view.alpha = 1
                view.transform = .identity
            }
        }
    }
}

/// Numbers and durations as the desktop's Overview shows them.
enum Figures {
    static func count(_ value: Int) -> String {
        value.formatted(.number)
    }

    /// Seconds under a minute, minutes under an hour, then hours to one decimal.
    static func duration(milliseconds: Int64) -> (value: String, unit: String) {
        let seconds = Double(milliseconds) / 1000
        if seconds < 60 { return (Int(seconds.rounded()).formatted(.number), L10n.unitSeconds) }
        if seconds < 3600 { return (Int((seconds / 60).rounded()).formatted(.number), L10n.unitMinutes) }
        return ((seconds / 3600).formatted(.number.precision(.fractionLength(0...1))), L10n.unitHours)
    }
}
