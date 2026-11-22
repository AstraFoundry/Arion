import UIKit

public extension UIImageView {
    /// An icon on a fixed square frame, so icons of different shapes align
    /// on the same column and baseline.
    convenience init(icon: Icon, size: CGFloat, color: UIColor = Palette.foreground) {
        self.init(image: icon.image)
        contentMode = .scaleAspectFit
        tintColor = color
        isAccessibilityElement = false
        translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            widthAnchor.constraint(equalToConstant: size),
            heightAnchor.constraint(equalToConstant: size),
        ])
        setContentHuggingPriority(.required, for: .horizontal)
        setContentCompressionResistancePriority(.required, for: .horizontal)
    }
}

public extension Icon {
    /// The icon drawn at a size, for places that take an image rather than a view.
    func image(size: CGFloat) -> UIImage {
        let base = image
        let rendered = UIGraphicsImageRenderer(size: CGSize(width: size, height: size)).image { _ in
            base.draw(in: CGRect(x: 0, y: 0, width: size, height: size))
        }
        return rendered.withRenderingMode(base.renderingMode == .alwaysOriginal ? .alwaysOriginal : .alwaysTemplate)
    }
}
