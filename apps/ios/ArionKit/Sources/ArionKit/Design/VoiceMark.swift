import UIKit

/// Arion's voice mark: the five rounded bars of the app icon. At rest it
/// draws the icon's proportions; given a level it follows the voice.
public final class VoiceMark: UIView {
    /// Bar heights from the icon generator (`apps/desktop/cmd/arion-icon`), as
    /// fractions of the tallest bar.
    static let heights: [CGFloat] = [230, 420, 560, 360, 250].map { $0 / 560 }
    private static let barRatio: CGFloat = 64 / 560
    private static let gapRatio: CGFloat = 46 / 560

    private var bars: [CALayer] = []
    private var level: CGFloat = 0
    private var phase: CGFloat = 0
    public var color: UIColor = Palette.foreground { didSet { paint() } }

    public override init(frame: CGRect) {
        super.init(frame: frame)
        isUserInteractionEnabled = false
        isAccessibilityElement = false
        for _ in Self.heights {
            let bar = CALayer()
            bar.cornerCurve = .circular
            layer.addSublayer(bar)
            bars.append(bar)
        }
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: VoiceMark, _) in self.paint() }
        paint()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    public override var intrinsicContentSize: CGSize {
        CGSize(width: UIView.noIntrinsicMetric, height: UIView.noIntrinsicMetric)
    }

    private func paint() {
        let cg = color.resolvedColor(with: traitCollection).cgColor
        bars.forEach { $0.backgroundColor = cg }
    }

    /// Follows a voice level in 0...1; zero returns to the mark's shape.
    public func setLevel(_ value: Float) {
        level = CGFloat(max(0, min(1, value)))
        phase += 0.9
        CATransaction.begin()
        CATransaction.setAnimationDuration(UIAccessibility.isReduceMotionEnabled ? 0 : 0.1)
        layoutBars()
        CATransaction.commit()
    }

    public override func layoutSubviews() {
        super.layoutSubviews()
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        layoutBars()
        CATransaction.commit()
    }

    private func layoutBars() {
        let tallest = bounds.height
        let width = tallest * Self.barRatio
        let gap = tallest * Self.gapRatio
        let total = width * CGFloat(bars.count) + gap * CGFloat(bars.count - 1)
        var x = (bounds.width - total) / 2
        for (index, bar) in bars.enumerated() {
            var height = tallest * Self.heights[index]
            if level > 0 {
                // Each bar moves on its own phase so speech reads as motion.
                let wobble = (sin(phase + CGFloat(index) * 1.7) + 1) / 2
                height = max(width, tallest * (0.25 + 0.75 * level * (0.55 + 0.45 * wobble)))
            }
            bar.frame = CGRect(x: x, y: (bounds.height - height) / 2, width: width, height: height)
            bar.cornerRadius = width / 2
            x += width + gap
        }
    }
}
