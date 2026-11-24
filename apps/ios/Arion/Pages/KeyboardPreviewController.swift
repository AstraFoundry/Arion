#if DEBUG
import ArionKit
import UIKit

/// Development only: renders the keyboard view in one state over a field, so
/// its states can be reviewed and captured without enabling the extension.
/// Launch with `-keyboardPreview idle|listening|polishing|failed`.
final class KeyboardPreviewController: UIViewController {
    private let state: String

    init(state: String) {
        self.state = state
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = Palette.background
        let field = PracticeField(placeholder: L10n.homePractice)
        let keyboard = VoiceKeyboardView()
        let backdrop = UIVisualEffectView(effect: UIBlurEffect(style: .systemChromeMaterial))
        for item in [field, backdrop, keyboard] as [UIView] {
            item.translatesAutoresizingMaskIntoConstraints = false
            view.addSubview(item)
        }
        NSLayoutConstraint.activate([
            field.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 24),
            field.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
            field.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
            field.heightAnchor.constraint(equalToConstant: 140),
            keyboard.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            keyboard.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            keyboard.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor),
            backdrop.topAnchor.constraint(equalTo: keyboard.topAnchor),
            backdrop.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            backdrop.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            backdrop.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        keyboard.languageMark = DictationLanguage.mark(DictationLanguage.english)
        keyboard.languageName = DictationLanguage.name(DictationLanguage.english)
        keyboard.translationName = TranslationTarget.zhCN.name
        let draft = Draft(finalized: "Let's move the review to Thursday afternoon, ", volatile: "and keep the")
        switch state {
        case "listening": keyboard.render(.listening(draft, level: 0.6))
        case "polishing": keyboard.render(.polishing(Draft(finalized: draft.text)))
        case "failed": keyboard.render(.failed(.fullAccessRequired))
        default: keyboard.render(.idle)
        }
    }
}
#endif
