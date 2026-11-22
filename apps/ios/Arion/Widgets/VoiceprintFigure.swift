import ArionKit
import UIKit

/// The microphone step's figure: a phone lying on its back, its screen
/// showing the microphone button and a voiceprint flowing from left to right
/// as someone speaks. It turns half a revolution as it grows into place,
/// then keeps listening. Drawn as an isometric solid with
/// the Anatomy kit's camera and shading (`.agents/skills/anatomy`, as the
/// desktop's `ui/shared/iso`): a turn by θ is the same phone drawn at
/// azimuth − θ. With Reduce Motion it shows a still voiceprint.
final class VoiceprintFigure: UIView {
    private static let height: CGFloat = 330
    private static let azimuth: CGFloat = 54
    private static let turn: CGFloat = 180
    private static let entrance: CFTimeInterval = 1.3

    // The phone, in world units of about a millimetre: x across, y along,
    // z up; it lies on the floor with its centre at the origin.
    private static let body = Plan(x: -36, y: -74, w: 72, d: 148, r: 12)
    private static let thickness: CGFloat = 7.6
    private static let screen = body.inset(3.4, radius: 9)
    private static let island = Plan(x: -10, y: -66, w: 20, d: 6, r: 3)
    // The voiceprint runs across the screen; the microphone sits below it.
    private static let bars = 15
    private static let barSpan: CGFloat = 52
    private static let barCentre: CGFloat = -12
    private static let barReach: CGFloat = 26
    private static let micCentre = CGPoint(x: 0, y: 42)
    private static let micRadius: CGFloat = 13

    private var link: CADisplayLink?
    private var started: CFTimeInterval = 0
    private var entered = false
    private var clock: CFTimeInterval = 0

    init() {
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        heightAnchor.constraint(equalToConstant: Self.height).isActive = true
        backgroundColor = .clear
        isOpaque = false
        contentMode = .redraw
        isAccessibilityElement = false
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: VoiceprintFigure, _) in self.setNeedsDisplay() }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func didMoveToWindow() {
        super.didMoveToWindow()
        link?.invalidate()
        link = nil
        guard window != nil else { return }
        if UIAccessibility.isReduceMotionEnabled {
            entered = true
            clock = 0.9
            setNeedsDisplay()
            return
        }
        // Moved back into a window (a step rebuilt after the permission
        // prompt), it carries on instead of entering again.
        if !entered { started = CACurrentMediaTime() }
        let link = CADisplayLink(target: self, selector: #selector(step))
        link.preferredFrameRateRange = CAFrameRateRange(minimum: 30, maximum: 60, preferred: 60)
        link.add(to: .main, forMode: .common)
        self.link = link
    }

    @objc private func step(_ link: CADisplayLink) {
        clock = link.timestamp - started
        setNeedsDisplay()
    }

    // MARK: Drawing

    override func draw(_ rect: CGRect) {
        guard let context = UIGraphicsGetCurrentContext() else { return }
        let tones = Tones(traitCollection)
        // The entrance runs once per appearance of the figure.
        let progress = entered ? 1 : min(1, CGFloat(clock / Self.entrance))
        if progress >= 1 { entered = true }
        let eased = 1 - pow(1 - progress, 3)
        let azimuth = Self.azimuth + Self.turn * (1 - eased)
        let zoom = 0.55 + 0.45 * eased
        context.setAlpha(min(1, progress * 3))

        let fit = Camera(azimuth: Self.azimuth, scale: 1, origin: .zero)
        let corners = Self.body.ring(steps: 2).flatMap { [fit.point($0, 0), fit.point($0, Self.thickness)] }
        let spanX = 2 * corners.map { abs($0.x) }.max()!
        let spanY = 2 * corners.map { abs($0.y - fit.point(.zero, Self.thickness / 2).y) }.max()!
        let scale = min((bounds.width - 24) / spanX, (bounds.height - 40) / spanY) * zoom
        var camera = Camera(azimuth: azimuth, scale: scale, origin: .zero)
        let middle = camera.point(.zero, Self.thickness / 2)
        camera.origin = CGPoint(x: bounds.midX - middle.x, y: bounds.midY - middle.y)

        // A soft shadow under the phone.
        let shadow = Slab(Self.body.ring(steps: 8), z: 0, height: 0.1, camera: camera).fill
        context.saveGState()
        context.setShadow(offset: CGSize(width: 0, height: 3 + 2000), blur: 14, color: tones.halo.cgColor)
        context.translateBy(x: 0, y: -2000)
        context.addPath(shadow)
        context.setFillColor(UIColor.black.cgColor)
        context.fillPath()
        context.restoreGState()

        // Buttons on the far sides go behind the body, the near ones in front.
        let buttons = Self.buttons.map { button in (button, button.normal.x * camera.viewer.x + button.normal.y * camera.viewer.y > 0) }
        for (button, near) in buttons where !near { draw(Slab(button.plan.ring(steps: 2), z: 2.2, height: 3.4, camera: camera), tones: tones, edge: tones.mid, in: context) }
        draw(Slab(Self.body.ring(steps: 8), z: 0, height: Self.thickness, camera: camera, bevel: 1), tones: tones, edge: tones.mid, in: context)
        for (button, near) in buttons where near { draw(Slab(button.plan.ring(steps: 2), z: 2.2, height: 3.4, camera: camera), tones: tones, edge: tones.mid, in: context) }

        // The screen, the island and the voiceprint, on the top face.
        let top = Self.thickness
        fill(camera.polygon(Self.screen.ring(steps: 8), top), tones.screen, stroke: tones.lo, in: context)
        fill(camera.polygon(Self.island.ring(steps: 6), top), tones.island, stroke: nil, in: context)
        let levels = levels(at: clock)
        for (index, amplitude) in levels.enumerated() {
            let x = -Self.barSpan / 2 + Self.barSpan * CGFloat(index) / CGFloat(Self.bars - 1)
            let reach = 1.4 + amplitude * Self.barReach
            let bar = Plan(x: x - 1.3, y: Self.barCentre - reach, w: 2.6, d: reach * 2, r: 1.3)
            fill(camera.polygon(bar.ring(steps: 4), top), tones.lit, stroke: nil, in: context)
        }

        // The microphone button, its ring swelling with the newest sound.
        let mic = Self.micCentre
        let ring = Self.micRadius + 2 + levels[0] * 6
        context.saveGState()
        context.setAlpha(min(1, progress * 3) * (0.5 - 0.3 * levels[0]))
        stroke(camera.polygon(circle(mic, ring), top), Palette.primary, width: 1.4, in: context)
        context.restoreGState()
        fill(camera.polygon(circle(mic, Self.micRadius), top), Palette.primary, stroke: nil, in: context)
        // The icon lies in the screen's plane: the camera maps that plane affinely.
        context.saveGState()
        let origin = camera.point(.zero, top), across = camera.point(CGPoint(x: 1, y: 0), top), along = camera.point(CGPoint(x: 0, y: 1), top)
        context.concatenate(CGAffineTransform(a: across.x - origin.x, b: across.y - origin.y, c: along.x - origin.x, d: along.y - origin.y, tx: origin.x, ty: origin.y))
        let glyph = Self.micRadius * 1.1
        // Rasterized at its size on screen, so the projection only shears it.
        Icon.mic.image(size: glyph * camera.k).withTintColor(Palette.primaryForeground, renderingMode: .alwaysOriginal)
            .draw(in: CGRect(x: mic.x - glyph / 2, y: mic.y - glyph / 2, width: glyph, height: glyph))
        context.restoreGState()
    }

    private func circle(_ centre: CGPoint, _ radius: CGFloat) -> [CGPoint] {
        (0..<48).map { step in
            let angle = CGFloat(step) / 48 * 2 * .pi
            return CGPoint(x: centre.x + radius * cos(angle), y: centre.y + radius * sin(angle))
        }
    }

    /// How far each bar of the voiceprint reaches, 0…1, left to right. Each
    /// bar shows the voice a moment later than the one on its left, so the
    /// syllables enter at the left and flow to the right.
    private func levels(at time: CFTimeInterval) -> [CGFloat] {
        func voice(_ t: CGFloat) -> CGFloat {
            let phase = (t * 1.6 + 0.3 * sin(t * 0.7)).truncatingRemainder(dividingBy: 1)
            let syllable = pow(max(0, sin(.pi * phase)), 1.4)
            let grain = 0.6 + 0.4 * sin(t * 11.3) * sin(t * 5.7 + 1)
            return (0.12 + 0.88 * syllable) * grain
        }
        let t = CGFloat(time)
        return (0..<Self.bars).map { index in
            let i = CGFloat(index)
            // Quieter toward the ends, so the print reads as one phrase.
            let window = pow(sin(.pi * (i + 0.5) / CGFloat(Self.bars)), 0.6)
            return min(1, max(0.05, voice(t - i * 0.075) * window))
        }
    }

    private func draw(_ slab: Slab, tones: Tones, edge: UIColor, in context: CGContext) {
        fill(slab.fill, tones.paper, stroke: nil, in: context)
        for (index, path) in slab.shades.enumerated() { fill(path, tones.shades[index], stroke: nil, in: context) }
        fill(slab.top, tones.top, stroke: nil, in: context)
        stroke(slab.crease, tones.faint, in: context)
        stroke(slab.bevel, tones.faint, in: context)
        stroke(slab.outline, edge, in: context)
    }

    private func fill(_ path: CGPath, _ color: UIColor, stroke line: UIColor?, in context: CGContext) {
        context.addPath(path)
        context.setFillColor(color.cgColor)
        context.fillPath()
        if let line { stroke(path, line, in: context) }
    }

    private func stroke(_ path: CGPath, _ color: UIColor, width: CGFloat = 0.6, in context: CGContext) {
        guard !path.isEmpty else { return }
        context.addPath(path)
        context.setStrokeColor(color.cgColor)
        context.setLineWidth(width)
        context.setLineCap(.round)
        context.setLineJoin(.round)
        context.strokePath()
    }

    private struct Button { let plan: Plan; let normal: CGPoint }

    // The action button and volume keys on the left side, the side button
    // on the right.
    private static let buttons: [Button] = [
        Button(plan: Plan(x: -37.4, y: -50, w: 1.8, d: 8, r: 0.8), normal: CGPoint(x: -1, y: 0)),
        Button(plan: Plan(x: -37.4, y: -36, w: 1.8, d: 13, r: 0.8), normal: CGPoint(x: -1, y: 0)),
        Button(plan: Plan(x: -37.4, y: -19, w: 1.8, d: 13, r: 0.8), normal: CGPoint(x: -1, y: 0)),
        Button(plan: Plan(x: 35.6, y: -40, w: 1.8, d: 20, r: 0.8), normal: CGPoint(x: 1, y: 0)),
    ]
}

// MARK: - The kit's camera and slabs

/// The Anatomy kit's tones (`kit/iso.css`), light and dark.
private struct Tones {
    let paper, top, hi, mid, lo, faint, lit, screen, island, halo: UIColor
    let shades: [UIColor]

    init(_ traits: UITraitCollection) {
        let dark = traits.userInterfaceStyle == .dark
        func hex(_ light: UInt32, _ dark: UInt32, alpha: CGFloat = 1) -> UIColor {
            let value = traits.userInterfaceStyle == .dark ? dark : light
            return UIColor(red: CGFloat(value >> 16 & 0xff) / 255, green: CGFloat(value >> 8 & 0xff) / 255, blue: CGFloat(value & 0xff) / 255, alpha: alpha)
        }
        paper = hex(0xfbfbfa, 0x131313)
        top = hex(0xffffff, 0x1c1c1c)
        hi = hex(0x26262a, 0xd4d4d4)
        mid = hex(0x8a8a8f, 0x6a6a6a)
        lo = hex(0xb4b4b8, 0x3e3e3e)
        faint = hex(0xd4d4d6, 0x2a2a2a)
        lit = hex(0x000000, 0xffffff)
        shades = [hex(0xdcdcd9, 0x0f0f0f), hex(0xe5e5e2, 0x121212), hex(0xededea, 0x151515), hex(0xf4f4f1, 0x181818)]
        screen = hex(0xedede9, 0x0b0b0b)
        island = hex(0xb4b4b8, 0x2a2a2a)
        halo = dark ? hex(0, 0x050505, alpha: 0.9) : hex(0x8a8a80, 0, alpha: 0.32)
    }
}

/// A rounded rectangle on the floor plan.
private struct Plan {
    let x, y, w, d, r: CGFloat

    func inset(_ by: CGFloat, radius: CGFloat) -> Plan {
        Plan(x: x + by, y: y + by, w: w - by * 2, d: d - by * 2, r: radius)
    }

    /// The outline, clockwise from the top right corner, as the kit's `roundedPlan`.
    func ring(steps: Int) -> [CGPoint] {
        let radius = max(0, min(r, w / 2, d / 2))
        let corners: [(CGFloat, CGFloat, CGFloat)] = [
            (x + w - radius, y + radius, -90), (x + w - radius, y + d - radius, 0),
            (x + radius, y + d - radius, 90), (x + radius, y + radius, 180),
        ]
        return corners.flatMap { cx, cy, start in
            (0...steps).map { step in
                let angle = (start + 90 * CGFloat(step) / CGFloat(steps)) * .pi / 180
                return CGPoint(x: cx + radius * cos(angle), y: cy + radius * sin(angle))
            }
        }
    }
}

/// The kit's camera: azimuth around z, 30° above the floor.
private struct Camera {
    let sinA, cosA, k: CGFloat
    var origin: CGPoint
    private let sinE = sin(30 * CGFloat.pi / 180)
    private let cosE = cos(30 * CGFloat.pi / 180)

    init(azimuth: CGFloat, scale: CGFloat, origin: CGPoint) {
        sinA = sin(azimuth * .pi / 180)
        cosA = cos(azimuth * .pi / 180)
        k = scale * sqrt(1.6)
        self.origin = origin
    }

    var viewer: CGPoint { CGPoint(x: cosA, y: sinA) }
    var light: CGPoint { CGPoint(x: -sinA, y: cosA) }

    func point(_ p: CGPoint, _ z: CGFloat) -> CGPoint {
        CGPoint(x: origin.x + (p.x * sinA - p.y * cosA) * k, y: origin.y + ((p.x * cosA + p.y * sinA) * sinE - z * cosE) * k)
    }

    func polygon(_ ring: [CGPoint], _ z: CGFloat) -> CGPath {
        let path = CGMutablePath()
        path.addLines(between: ring.map { point($0, z) })
        path.closeSubpath()
        return path
    }
}

/// A plan extruded upward, shaded as the kit's `extrude`: side faces in
/// four tones by how they face the light, a top, the outline where faces
/// turn away, and a faint crease where the top meets a near side.
private struct Slab {
    let fill, top: CGPath
    let shades: [CGPath]
    let outline, crease, bevel: CGPath

    init(_ ring: [CGPoint], z: CGFloat, height: CGFloat, camera: Camera, bevel inset: CGFloat = 0) {
        let count = ring.count
        // Plans run clockwise in a y-down plan, so outward normals turn right.
        let normals = (0..<count).map { index -> CGPoint in
            let a = ring[index], b = ring[(index + 1) % count]
            let length = max(hypot(b.x - a.x, b.y - a.y), 1e-9)
            return CGPoint(x: (b.y - a.y) / length, y: -(b.x - a.x) / length)
        }
        let facing = normals.map { $0.x * camera.viewer.x + $0.y * camera.viewer.y > 1e-9 }
        let upper = ring.map { camera.point($0, z + height) }
        let lower = ring.map { camera.point($0, z) }

        let outline = CGMutablePath(), crease = CGMutablePath()
        for run in Self.runs(facing, false) { outline.addLines(between: run.map { upper[$0] }) }
        for run in Self.runs(facing, true) {
            crease.addLines(between: run.map { upper[$0] })
            outline.addLines(between: run.map { lower[$0] })
        }
        for index in 0..<count where facing[(index + count - 1) % count] != facing[index] {
            outline.addLines(between: [upper[index], lower[index]])
        }
        self.outline = outline
        self.crease = crease

        let hull = Self.hull(upper + lower)
        let fill = CGMutablePath()
        fill.addLines(between: hull)
        fill.closeSubpath()
        self.fill = fill
        let top = CGMutablePath()
        top.addLines(between: upper)
        top.closeSubpath()
        self.top = top

        var shades = (0..<4).map { _ in CGMutablePath() }
        for index in 0..<count where facing[index] {
            let lit = (normals[index].x * camera.light.x + normals[index].y * camera.light.y + 1) / 2
            let shade = min(3, max(0, Int(lit * 4)))
            let next = (index + 1) % count
            shades[shade].addLines(between: [upper[index], upper[next], lower[next], lower[index]])
            shades[shade].closeSubpath()
        }
        self.shades = shades

        let bevel = CGMutablePath()
        if inset > 0 {
            let inner = (0..<count).map { index -> CGPoint in
                let a = normals[(index + count - 1) % count], b = normals[index]
                let m = CGPoint(x: a.x + b.x, y: a.y + b.y)
                let length = max(hypot(m.x, m.y), 1e-9)
                return camera.point(CGPoint(x: ring[index].x - m.x / length * inset, y: ring[index].y - m.y / length * inset), z + height)
            }
            for run in Self.runs(facing, true) where run.count > 2 {
                bevel.addLines(between: [upper[run[0]]] + run.dropFirst().dropLast().map { inner[$0] } + [upper[run[run.count - 1]]])
            }
        }
        self.bevel = bevel
    }

    /// Runs of consecutive edges whose facing is `want`, as vertex indices.
    private static func runs(_ flags: [Bool], _ want: Bool) -> [[Int]] {
        let count = flags.count
        guard let start = (0..<count).first(where: { flags[$0] != flags[($0 + count - 1) % count] }) else {
            return flags.first == want ? [Array(0...count).map { $0 % count }] : []
        }
        var runs: [[Int]] = [], run: [Int]? = nil
        for step in 0..<count {
            let index = (start + step) % count
            if flags[index] == want {
                if run == nil { run = [index] }
                run!.append((index + 1) % count)
            } else if let done = run {
                runs.append(done)
                run = nil
            }
        }
        if let run { runs.append(run) }
        return runs
    }

    private static func hull(_ points: [CGPoint]) -> [CGPoint] {
        let sorted = points.sorted { $0.x == $1.x ? $0.y < $1.y : $0.x < $1.x }
        guard sorted.count > 2 else { return sorted }
        func cross(_ o: CGPoint, _ a: CGPoint, _ b: CGPoint) -> CGFloat { (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x) }
        var lower: [CGPoint] = [], upper: [CGPoint] = []
        for point in sorted {
            while lower.count >= 2, cross(lower[lower.count - 2], lower[lower.count - 1], point) <= 0 { lower.removeLast() }
            lower.append(point)
        }
        for point in sorted.reversed() {
            while upper.count >= 2, cross(upper[upper.count - 2], upper[upper.count - 1], point) <= 0 { upper.removeLast() }
            upper.append(point)
        }
        return lower.dropLast() + upper.dropLast()
    }
}
