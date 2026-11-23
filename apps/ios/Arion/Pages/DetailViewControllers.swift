import ArionKit
import UIKit

/// Base for the settings detail lists: Sunbeam surfaces on an inset table.
class SettingsListController: UITableViewController {
    let preferences = Preferences()

    init(title: String) {
        super.init(style: .insetGrouped)
        self.title = title
        navigationItem.largeTitleDisplayMode = .never
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView.backgroundColor = Palette.background
    }

    func cell(_ text: String, secondary: String? = nil, checked: Bool = false, icon: Icon? = nil) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = Palette.card
        var content = UIListContentConfiguration.subtitleCell()
        content.text = text
        content.textProperties.font = TextStyle.body.font
        content.textProperties.color = Palette.foreground
        content.secondaryText = secondary
        if let icon { RowIcon.apply(icon, to: &content) }
        content.secondaryTextProperties.font = TextStyle.caption.font
        content.secondaryTextProperties.color = Palette.mutedForeground
        cell.contentConfiguration = content
        if checked {
            let mark = UIImageView(image: Icon.check.image(size: 18))
            mark.tintColor = Palette.foreground
            cell.accessoryView = mark
        }
        return cell
    }
}

/// The recognition languages, in the order the language key cycles them.
final class LanguagesViewController: SettingsListController {
    init() { super.init(title: L10n.settingsLanguages) }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { DictationLanguage.all.count }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let language = DictationLanguage.all[indexPath.row]
        return cell(DictationLanguage.name(language), checked: preferences.languages.contains(language))
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let language = DictationLanguage.all[indexPath.row]
        var languages = preferences.languages
        if let index = languages.firstIndex(of: language) {
            guard languages.count > 1 else { return }
            languages.remove(at: index)
        } else {
            languages.append(language)
            Task.detached { try? await Recognizer.install(language) }
        }
        preferences.languages = languages
        tableView.reloadRows(at: [indexPath], with: .none)
    }
}

/// The language the translate key produces.
final class TargetViewController: SettingsListController {
    init() { super.init(title: L10n.settingsTranslateInto) }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { TranslationTarget.allCases.count }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let target = TranslationTarget.allCases[indexPath.row]
        return cell(target.name, checked: preferences.translationTarget == target)
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        preferences.translationTarget = TranslationTarget.allCases[indexPath.row]
        tableView.reloadData()
    }
}

/// The Pinyin layout the keyboard's 中 key opens. 中/EN is the only toggle
/// on the keyboard itself, so this choice lives here.
final class ChineseTypingViewController: SettingsListController {
    private static let choices: [TypingLayout] = [.pinyin, .pinyin9]

    static func name(_ layout: TypingLayout) -> String {
        layout == .pinyin9 ? L10n.typingPinyin9 : L10n.typingPinyin
    }

    init() { super.init(title: L10n.typingSetting) }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { Self.choices.count }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let layout = Self.choices[indexPath.row]
        return cell(Self.name(layout), checked: preferences.chineseTypingLayout == layout)
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        preferences.chineseTypingLayout = Self.choices[indexPath.row]
        tableView.reloadData()
    }
}

/// Words to recognize and keep as written.
final class VocabularyViewController: SettingsListController, UITextFieldDelegate {
    private let vocabulary = Vocabulary()
    private var terms: [String] = []
    private let field = UITextField()

    init() { super.init(title: L10n.settingsVocabulary) }

    override func viewDidLoad() {
        super.viewDidLoad()
        field.placeholder = L10n.settingsVocabularyAdd
        field.font = TextStyle.body.font
        field.returnKeyType = .done
        field.autocorrectionType = .no
        field.autocapitalizationType = .none
        field.clearButtonMode = .whileEditing
        field.delegate = self
        terms = (try? vocabulary.terms()) ?? []
        // History is the other thing Arion remembers; it opens from here.
        let history = UIBarButtonItem(image: Icon.history.image(size: 22), primaryAction: UIAction { [weak self] _ in
            self?.navigationController?.pushViewController(HistoryViewController(), animated: true)
        })
        history.accessibilityLabel = L10n.settingsHistory
        navigationItem.rightBarButtonItem = history
    }

    override func numberOfSections(in tableView: UITableView) -> Int { 2 }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { section == 0 ? 1 : terms.count }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        guard indexPath.section == 0 else { return cell(terms[indexPath.row]) }
        let cell = UITableViewCell()
        cell.backgroundColor = Palette.card
        cell.selectionStyle = .none
        field.translatesAutoresizingMaskIntoConstraints = false
        cell.contentView.addSubview(field)
        NSLayoutConstraint.activate([
            field.leadingAnchor.constraint(equalTo: cell.contentView.layoutMarginsGuide.leadingAnchor),
            field.trailingAnchor.constraint(equalTo: cell.contentView.layoutMarginsGuide.trailingAnchor),
            field.topAnchor.constraint(equalTo: cell.contentView.topAnchor),
            field.bottomAnchor.constraint(equalTo: cell.contentView.bottomAnchor),
            field.heightAnchor.constraint(greaterThanOrEqualToConstant: 48),
        ])
        return cell
    }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        if let text = textField.text, (try? vocabulary.add(text)) == true {
            terms = (try? vocabulary.terms()) ?? terms
            tableView.insertRows(at: [IndexPath(row: 0, section: 1)], with: .automatic)
        }
        textField.text = nil
        return false
    }

    override func tableView(_ tableView: UITableView, canEditRowAt indexPath: IndexPath) -> Bool { indexPath.section == 1 }

    override func tableView(_ tableView: UITableView, commit editingStyle: UITableViewCell.EditingStyle, forRowAt indexPath: IndexPath) {
        guard editingStyle == .delete else { return }
        do {
            try vocabulary.remove(terms[indexPath.row])
            terms.remove(at: indexPath.row)
            tableView.deleteRows(at: [indexPath], with: .automatic)
        } catch {
            tableView.reloadData()
        }
    }
}

/// Recent dictations, when the user keeps them.
final class HistoryViewController: SettingsListController {
    private let history = History()
    private var entries: [HistoryEntry] = []
    private let formatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        formatter.doesRelativeDateFormatting = true
        return formatter
    }()

    init() { super.init(title: L10n.settingsHistory) }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        entries = (try? history.entries()) ?? []
        tableView.reloadData()
    }

    override func numberOfSections(in tableView: UITableView) -> Int { preferences.historyEnabled ? 2 : 1 }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        section == 0 ? 1 : max(entries.count, 1)
    }

    override func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? {
        section == 0 ? L10n.settingsHistoryFooter : nil
    }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if indexPath.section == 0 {
            let cell = cell(L10n.settingsHistoryKeep)
            let toggle = UISwitch()
            toggle.isOn = preferences.historyEnabled
            toggle.addTarget(self, action: #selector(toggled(_:)), for: .valueChanged)
            cell.accessoryView = toggle
            cell.selectionStyle = .none
            return cell
        }
        guard !entries.isEmpty else {
            let cell = cell(L10n.settingsHistoryEmpty)
            if var content = cell.contentConfiguration as? UIListContentConfiguration {
                content.textProperties.color = Palette.mutedForeground
                cell.contentConfiguration = content
            }
            cell.selectionStyle = .none
            return cell
        }
        let entry = entries[indexPath.row]
        return cell(entry.text, secondary: formatter.string(from: entry.date))
    }

    @objc private func toggled(_ toggle: UISwitch) {
        preferences.historyEnabled = toggle.isOn
        if !toggle.isOn {
            try? history.erase()
            entries = []
        }
        tableView.reloadData()
    }

    override func tableView(_ tableView: UITableView, contextMenuConfigurationForRowAt indexPath: IndexPath, point: CGPoint) -> UIContextMenuConfiguration? {
        guard indexPath.section == 1, indexPath.row < entries.count else { return nil }
        let entry = entries[indexPath.row]
        return UIContextMenuConfiguration(actionProvider: { _ in
            var actions = [UIAction(title: L10n.settingsHistoryCopy, image: Icon.copy.image(size: 20)) { _ in
                UIPasteboard.general.string = entry.text
            }]
            if let original = entry.original {
                actions.append(UIAction(title: L10n.settingsHistoryOriginal, image: Icon.history.image(size: 20)) { _ in
                    UIPasteboard.general.string = original
                })
            }
            actions.append(UIAction(title: L10n.settingsHistoryDelete, image: Icon.trash.image(size: 20), attributes: .destructive) { [weak self] _ in
                try? self?.history.remove(entry.id)
                self?.entries.removeAll { $0.id == entry.id }
                self?.tableView.reloadData()
            })
            return UIMenu(children: actions)
        })
    }
}
