import UIKit

/// Filled key icons for pressed and active states, composed from the
/// variants `pnpm mobile:generate` derives from the official SVGs: the fill,
/// and the open strokes, which are knocked out (Delete's ×) or drawn over it
/// (caps lock's bar).
enum KeyGlyph {
    @MainActor private static var cache: [Icon: UIImage] = [:]

    @MainActor
    static func filled(_ icon: Icon, size: CGFloat) -> UIImage? {
        if let image = cache[icon] { return image }
        let parts: (fill: Icon, cut: Icon?, knockout: Bool)
        switch icon {
        case .arrowBigUp: parts = (.arrowBigUpFill, nil, false)
        case .arrowBigUpDash: parts = (.arrowBigUpDashFill, .arrowBigUpDashCut, false)
        case .delete: parts = (.deleteFill, .deleteCut, true)
        default: return nil
        }
        let frame = CGRect(x: 0, y: 0, width: size, height: size)
        let image = UIGraphicsImageRenderer(size: frame.size).image { _ in
            parts.fill.image(size: size).withTintColor(.black).draw(in: frame)
            parts.cut?.image(size: size).withTintColor(.black)
                .draw(in: frame, blendMode: parts.knockout ? .destinationOut : .normal, alpha: 1)
        }.withRenderingMode(.alwaysTemplate)
        cache[icon] = image
        return image
    }
}
