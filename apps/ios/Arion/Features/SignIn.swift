import ArionKit
import UIKit

/// Arion's site pages the sign-in links to, localized as the desktop's
/// (`ui/shared/config`).
enum SiteLinks {
    private static let site = "https://arion.astrafoundry.si"

    private static var origin: String {
        Locale.preferredLanguages.first?.hasPrefix("zh") == true ? site + "/zh" : site
    }

    static var terms: URL { URL(string: origin + "/terms/")! }
    static var privacy: URL { URL(string: origin + "/privacy/")! }
    static var usageData: URL { URL(string: origin + "/usage-data/")! }
}

/// The account's sign-in methods. Every method opens the same browser
/// sign-in; the issuer offers the choice again.
final class SignInMethods: UIStackView {
    private var buttons: [UIButton] = []

    init(action: @escaping () -> Void) {
        super.init(frame: .zero)
        axis = .vertical
        spacing = 8
        let methods: [(String, UIImage?)] = [
            (L10n.accountContinueGoogle, Icon.google.image(size: 18)),
            (L10n.accountContinueGitHub, Icon.github.image(size: 18)),
            (L10n.accountContinueEmail, Icon.mail.image(size: 18)),
        ]
        for (title, image) in methods {
            let button = OutlineButton(title: title, image: image)
            button.addAction(UIAction { _ in action() }, for: .touchUpInside)
            addArrangedSubview(button)
            buttons.append(button)
        }
    }

    @available(*, unavailable)
    required init(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    var isEnabled: Bool = true {
        didSet { buttons.forEach { $0.isEnabled = isEnabled } }
    }
}

/// A bordered button on the page: the card's face with a hairline edge.
final class OutlineButton: UIButton {
    init(title: String, image: UIImage?) {
        super.init(frame: .zero)
        var config = UIButton.Configuration.plain()
        var text = AttributedString(title)
        text.font = TextStyle.label.font
        config.attributedTitle = text
        config.image = image
        config.imagePadding = 10
        config.baseForegroundColor = Palette.foreground
        config.background.backgroundColor = Palette.card
        config.background.strokeColor = Palette.border
        config.background.strokeWidth = 1
        config.background.cornerRadius = SunbeamTokens.Radius.xl
        config.cornerStyle = .fixed
        config.contentInsets = NSDirectionalEdgeInsets(top: 0, leading: 16, bottom: 0, trailing: 16)
        configuration = config
        configurationUpdateHandler = { button in
            button.configuration?.background.backgroundColor = button.isHighlighted ? Palette.flatFacePressed : Palette.card
            button.alpha = button.isEnabled ? 1 : 0.45
        }
        heightAnchor.constraint(equalToConstant: 48).isActive = true
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

}

/// A checkbox with a one-line label whose named words link to documents.
/// The links keep their own taps; the rest of the row toggles the box.
final class ConsentRow: UIView, UITextViewDelegate {
    private let box = UIImageView()
    private let text = UITextView()
    var onChange: ((Bool) -> Void)?

    var isChecked: Bool {
        didSet { paint() }
    }

    init(label: String, links: [(String, URL)], checked: Bool) {
        isChecked = checked
        super.init(frame: .zero)
        let attributed = NSMutableAttributedString(string: label, attributes: [
            .font: TextStyle.caption.font,
            .foregroundColor: Palette.foreground,
        ])
        for (word, url) in links {
            let range = (label as NSString).range(of: word)
            precondition(range.location != NSNotFound, "consent link is missing from its localized label")
            attributed.addAttributes([.link: url], range: range)
        }
        text.attributedText = attributed
        text.linkTextAttributes = [.foregroundColor: Palette.foreground, .underlineStyle: NSUnderlineStyle.single.rawValue]
        text.isEditable = false
        text.isScrollEnabled = false
        text.backgroundColor = .clear
        text.textContainerInset = .zero
        text.textContainer.lineFragmentPadding = 0
        text.adjustsFontForContentSizeCategory = true
        box.contentMode = .scaleAspectFit
        box.image = Icon.square.image
        box.tintColor = Palette.foreground
        for view in [box, text] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        NSLayoutConstraint.activate([
            box.leadingAnchor.constraint(equalTo: leadingAnchor),
            box.widthAnchor.constraint(equalToConstant: 20),
            box.heightAnchor.constraint(equalToConstant: 20),
            box.topAnchor.constraint(equalTo: topAnchor),
            text.leadingAnchor.constraint(equalTo: box.trailingAnchor, constant: 10),
            text.trailingAnchor.constraint(equalTo: trailingAnchor),
            text.topAnchor.constraint(equalTo: topAnchor, constant: 1),
            text.bottomAnchor.constraint(equalTo: bottomAnchor),
            box.bottomAnchor.constraint(lessThanOrEqualTo: bottomAnchor),
        ])
        let tap = UITapGestureRecognizer(target: self, action: #selector(toggle(_:)))
        addGestureRecognizer(tap)
        isAccessibilityElement = true
        accessibilityLabel = label
        accessibilityTraits = .button
        paint()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    @objc private func toggle(_ tap: UITapGestureRecognizer) {
        // A tap on a link opens it; anywhere else toggles the box.
        let point = tap.location(in: text)
        if text.bounds.contains(point), let position = text.closestPosition(to: point) {
            let offset = text.offset(from: text.beginningOfDocument, to: position)
            if offset < text.attributedText.length, text.attributedText.attribute(.link, at: offset, effectiveRange: nil) != nil { return }
        }
        isChecked.toggle()
        onChange?(isChecked)
    }

    override func accessibilityActivate() -> Bool {
        isChecked.toggle()
        onChange?(isChecked)
        return true
    }

    private func paint() {
        box.image = (isChecked ? Icon.squareCheck : Icon.square).image
        accessibilityValue = isChecked ? "1" : "0"
        accessibilityTraits = isChecked ? [.button, .selected] : .button
    }
}

/// The account in a settings row or a page header: the profile picture or
/// initials, then the name and email, or the person glyph while signed out.
final class AvatarView: UIView {
    private let initials = UILabel()
    private let image = UIImageView()
    private var loaded: URL?
    private let size: CGFloat

    init(size: CGFloat) {
        self.size = size
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        backgroundColor = Palette.secondary
        layer.cornerRadius = size / 2
        clipsToBounds = true
        initials.font = .systemFont(ofSize: size * 0.38, weight: .semibold)
        initials.textColor = Palette.secondaryForeground
        initials.textAlignment = .center
        image.contentMode = .scaleAspectFill
        image.tintColor = Palette.mutedForeground
        for view in [initials, image] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
            NSLayoutConstraint.activate([
                view.topAnchor.constraint(equalTo: topAnchor),
                view.bottomAnchor.constraint(equalTo: bottomAnchor),
                view.leadingAnchor.constraint(equalTo: leadingAnchor),
                view.trailingAnchor.constraint(equalTo: trailingAnchor),
            ])
        }
        NSLayoutConstraint.activate([widthAnchor.constraint(equalToConstant: size), heightAnchor.constraint(equalToConstant: size)])
        isAccessibilityElement = false
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    func show(_ profile: AccountProfile?) {
        guard let profile else {
            initials.text = nil
            image.contentMode = .center
            image.image = Icon.user.image(size: (size * 0.5).rounded())
            loaded = nil
            return
        }
        initials.text = profile.initials
        image.contentMode = .scaleAspectFill
        if loaded != profile.picture { image.image = nil }
        guard let url = profile.picture, url.scheme == "https", loaded != url else { return }
        loaded = url
        Task { [weak self] in
            // At most 2 MiB, as the desktop; a failure leaves the initials.
            guard let (data, response) = try? await URLSession.shared.data(from: url),
                  (response as? HTTPURLResponse)?.statusCode == 200, data.count <= 2 << 20,
                  let picture = UIImage(data: data), self?.loaded == url else { return }
            self?.image.image = picture
        }
    }
}

/// Copy for each account failure.
enum AccountCopy {
    static func failure(_ failure: AccountFailure) -> String {
        switch failure {
        case .inProgress: L10n.accountErrorInProgress
        case .denied: L10n.accountErrorDenied
        case .authorization: L10n.accountErrorAuthorization
        case .token: L10n.accountErrorToken
        case .discovery: L10n.accountErrorDiscovery
        case .store: L10n.accountErrorStore
        case .sessionEnded: L10n.accountErrorSessionEnded
        case .refresh: L10n.accountErrorRefresh
        case .revokeUnconfirmed: L10n.accountErrorRevokeUnconfirmed
        case .usageConsent: L10n.accountErrorUsageConsent
        case .generic: L10n.accountErrorGeneric
        }
    }
}
