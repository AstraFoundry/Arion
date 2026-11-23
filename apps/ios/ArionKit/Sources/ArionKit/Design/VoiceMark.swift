import UIKit

/// Arion's voice mark, ported from the desktop pill's
/// (`apps/desktop/ui/shared/kit/voicemark.go`): the five bars of the app mark
/// as the input meter, folded into three dots that a pulse travels through
/// while the microphone opens or recognition finishes, the dots turning on a
/// circle while the rewrite organizes the text, and the full mark at rest.
/// Each mode only sets spring targets on the same five shapes, so a change of
/// mode, even mid-way, is one continuous motion.
public final class VoiceMark: UIView {
    public enum Mode: Equatable { case listening, waiting, orbiting, resting }

    /// Bar heights from the icon generator (`apps/desktop/cmd/arion-icon`), as
    /// fractions of the tallest bar.
    static let heights: [CGFloat] = [230, 420, 560, 360, 250].map { $0 / 560 }
    /// The waiting dot each bar folds into: the center bar becomes the center
    /// dot and each outer pair merges into the dot beside it.
    private static let dots = [0, 0, 1, 2, 2]
    /// How much of the silhouette stands in silence.
    private static let rest: CGFloat = 0.4
    /// Delays each bar's fold by its distance from the center.
    private static let stagger: CGFloat = 0.12
    private static let wave: CFTimeInterval = 1.4
    private static let spin: CFTimeInterval = 1.6
    private static let turn: CGFloat = 0.68
    private static let chase: CGFloat = 0.045
    /// Recent levels, newest last: about a second of 20 ms audio frames, as
    /// the desktop keeps.
    private static let history = 48

    public var color: UIColor = Palette.foreground { didSet { paint() } }
    public private(set) var mode: Mode = .resting

    private var shapes: [CALayer] = []
    private var levels: [CGFloat] = []
    // The levels spring as the desktop's do; fold and orbit use its Pop.
    private var bars = (0..<5).map { _ in Spring(value: 1, stiffness: 520, damping: 32) }
    private var fold = Spring.pop()
    private var orbit = Spring.pop()
    private var waveStart: CFTimeInterval = 0
    private var spinStart: CFTimeInterval = 0
    private var link: CADisplayLink?
    private var last: CFTimeInterval = 0

    public override init(frame: CGRect) {
        super.init(frame: frame)
        isUserInteractionEnabled = false
        isAccessibilityElement = false
        for _ in Self.heights {
            let shape = CALayer()
            shape.cornerCurve = .circular
            layer.addSublayer(shape)
            shapes.append(shape)
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
        shapes.forEach { $0.backgroundColor = cg }
    }

    public func setMode(_ mode: Mode) {
        guard mode != self.mode else { return }
        let now = CACurrentMediaTime()
        if mode == .listening { levels.removeAll() }
        let folded = { (m: Mode) in m == .waiting || m == .orbiting }
        if folded(mode), !folded(self.mode) { waveStart = now }
        if mode == .orbiting, self.mode != .orbiting { spinStart = now }
        self.mode = mode
        fold.target = folded(mode) ? 1 : 0
        orbit.target = mode == .orbiting ? 1 : 0
        retarget()
        animate()
    }

    /// Adds the newest voice level, 0...1, mapped as the desktop maps it.
    public func setLevel(_ value: Float) {
        guard mode == .listening else { return }
        levels.append(CGFloat(max(0, min(1, value))))
        if levels.count > Self.history { levels.removeFirst(levels.count - Self.history) }
        retarget()
        animate()
    }

    /// The center bar shows the newest level and each step outward an older
    /// one, so the voice travels out from the middle (the desktop's levelAt).
    private func retarget() {
        for index in bars.indices {
            if mode == .resting {
                bars[index].target = 1
                continue
            }
            let n = levels.count
            let position = n - 1 - abs(index - 2) * max(n / bars.count, 1)
            bars[index].target = n == 0 || position < 0 ? 0 : levels[position]
        }
    }

    private func animate() {
        if UIAccessibility.isReduceMotionEnabled {
            for index in bars.indices { bars[index].settle() }
            fold.settle()
            orbit.settle()
            draw(at: CACurrentMediaTime())
            return
        }
        guard link == nil, window != nil else { return }
        last = CACurrentMediaTime()
        let link = CADisplayLink(target: self, selector: #selector(tick(_:)))
        link.add(to: .main, forMode: .common)
        self.link = link
    }

    @objc private func tick(_ link: CADisplayLink) {
        let now = link.timestamp
        let dt = min(max(now - last, 0), 0.05)
        last = now
        var moving = false
        for index in bars.indices { moving = bars[index].step(dt) || moving }
        moving = fold.step(dt) || moving
        moving = orbit.step(dt) || moving
        draw(at: now)
        // The waiting and orbiting loops run for as long as their mode does.
        if !moving, fold.value < 0.004 {
            link.invalidate()
            self.link = nil
        }
    }

    public override func didMoveToWindow() {
        super.didMoveToWindow()
        if window == nil {
            link?.invalidate()
            link = nil
        } else {
            animate()
        }
    }

    public override func layoutSubviews() {
        super.layoutSubviews()
        draw(at: CACurrentMediaTime())
    }

    /// The desktop's markLayout without its live dot: the keyboard's button
    /// face already says it is listening.
    private func draw(at now: CFTimeInterval) {
        let h = min(bounds.height, bounds.width / 1.35)
        let bar = 0.15 * h
        let dot = bar * 5 / 3
        let orbit = Self.clamp(self.orbit.value)
        let wave = fold.value > 0.004 && orbit < 0.996 && !UIAccessibility.isReduceMotionEnabled
            ? CGFloat((now - waveStart).truncatingRemainder(dividingBy: Self.wave) / Self.wave) : 1.0 / 3
        let u = orbit > 0.004 && !UIAccessibility.isReduceMotionEnabled
            ? CGFloat((now - spinStart).truncatingRemainder(dividingBy: Self.spin) / Self.spin) : 0
        let left = (bounds.width - 9 * bar) / 2
        let top = (bounds.height - h) / 2
        let center = left + 4.5 * bar
        let ring = 0.3 * h
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        for (index, shape) in shapes.enumerated() {
            let distance = CGFloat(abs(index - 2))
            let p = Self.clamp((fold.value - Self.stagger * distance) / (1 - 2 * Self.stagger))
            let j = Self.dots[index]
            var d = abs(wave - CGFloat(j) / 3)
            d = min(d, 1 - d)
            // On the circle the dots stay whole: the turn is the motion.
            let pulse = Self.lerp(exp(-(d * d) / 0.045), 1, orbit)
            let size = dot * (0.84 + 0.22 * pulse)
            let spin = Self.ease(Self.clamp((u - Self.chase * CGFloat(j)) / Self.turn))
            let angle = 2 * .pi * (CGFloat(j - 1) / 3 - 0.25 + spin)
            let dx = Self.lerp(center + CGFloat(j - 1) * 3 * bar, center + ring * cos(angle), orbit)
            let dy = Self.lerp(top + h / 2 - 0.2 * h * pulse, top + h / 2 + ring * sin(angle), orbit)
            let standing = bar + (h * Self.heights[index] - bar) * (Self.rest + (1 - Self.rest) * Self.clamp(bars[index].value))
            let w = Self.lerp(bar, size, p)
            let ht = Self.lerp(standing, size, p)
            let x = Self.lerp(left + bar / 2 + CGFloat(index) * 2 * bar, dx, p)
            let y = Self.lerp(top + h / 2, dy, p)
            var alpha = Self.lerp(1, 0.5 + 0.5 * pulse, p)
            // Odd bars merge into a neighbor's dot and leave their alpha behind.
            if index % 2 == 1 { alpha *= 1 - p }
            shape.frame = CGRect(x: x - w / 2, y: y - ht / 2, width: w, height: ht)
            shape.cornerRadius = min(w, ht) / 2
            shape.opacity = Float(alpha)
        }
        CATransaction.commit()
    }

    private static func lerp(_ a: CGFloat, _ b: CGFloat, _ t: CGFloat) -> CGFloat { a * (1 - t) + b * t }
    private static func clamp(_ v: CGFloat) -> CGFloat { min(1, max(0, v)) }

    /// The orbit's turn, cubic-bezier(0.7, 0, 0.25, 1): a slow start, a fast
    /// middle and a long settle.
    static func ease(_ x: CGFloat) -> CGFloat {
        let (x1, y1, x2, y2): (CGFloat, CGFloat, CGFloat, CGFloat) = (0.7, 0, 0.25, 1)
        func bezier(_ t: CGFloat, _ a: CGFloat, _ b: CGFloat) -> CGFloat {
            3 * (1 - t) * (1 - t) * t * a + 3 * (1 - t) * t * t * b + t * t * t
        }
        var low: CGFloat = 0, high: CGFloat = 1, t = x
        for _ in 0..<24 {
            t = (low + high) / 2
            if bezier(t, x1, x2) < x { low = t } else { high = t }
        }
        return bezier(t, y1, y2)
    }
}

/// A damped spring with unit mass, as the desktop's motion springs.
struct Spring {
    var value: CGFloat
    var velocity: CGFloat = 0
    var target: CGFloat
    let stiffness: CGFloat
    let damping: CGFloat

    init(value: CGFloat, stiffness: CGFloat, damping: CGFloat) {
        self.value = value
        target = value
        self.stiffness = stiffness
        self.damping = damping
    }

    /// The desktop theme's Pop: 380 ms with a 0.18 bounce, converted as
    /// SwiftUI and the desktop's mygo describe a duration and bounce.
    static func pop() -> Spring {
        let duration: CGFloat = 0.38, bounce: CGFloat = 0.18
        return Spring(value: 0, stiffness: pow(2 * .pi / duration, 2), damping: 4 * .pi * (1 - bounce) / duration)
    }

    mutating func settle() {
        value = target
        velocity = 0
    }

    /// Advances by dt; false once it has come to rest.
    mutating func step(_ dt: CFTimeInterval) -> Bool {
        var left = CGFloat(dt)
        while left > 0 {
            let h = min(left, 1.0 / 240)
            velocity += (-stiffness * (value - target) - damping * velocity) * h
            value += velocity * h
            left -= h
        }
        if abs(value - target) < 0.001, abs(velocity) < 0.001 {
            settle()
            return false
        }
        return true
    }
}
