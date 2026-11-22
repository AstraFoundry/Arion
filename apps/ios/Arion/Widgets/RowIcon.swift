import ArionKit
import UIKit

/// The leading icon of a settings row: one size and one reserved column, so
/// every row's text starts at the same edge.
enum RowIcon {
    static let size: CGFloat = 18

    static func apply(_ icon: Icon, to content: inout UIListContentConfiguration, color: UIColor = Palette.foreground) {
        let original = icon.original
        content.image = icon.image(size: size)
        content.imageProperties.maximumSize = CGSize(width: size, height: size)
        content.imageProperties.reservedLayoutSize = CGSize(width: size, height: size)
        if !original { content.imageProperties.tintColor = color }
        content.imageToTextPadding = 14
    }

    /// The mark of a cloud provider; compatible endpoints use their vendor's.
    static func provider(_ provider: CloudProvider) -> Icon {
        switch provider {
        case .openai, .openaiCompatible: .lobeOpenai
        case .anthropic, .anthropicCompatible: .lobeAnthropic
        case .gemini: .lobeGeminiMono
        case .deepseek: .lobeDeepseek
        case .openrouter: .lobeOpenrouter
        }
    }
}
