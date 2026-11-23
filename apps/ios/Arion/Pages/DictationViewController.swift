import ArionKit
import UIKit

/// The Dictation tab: recognition languages, the translation target, the
/// speech model and the keyboard.
final class DictationViewController: UITableViewController {
    private enum Row {
        case languages, target, model, keyboard, chineseTyping
    }

    private struct Section {
        let title: String?
        let rows: [Row]
        let footer: String?
    }

    private let preferences = Preferences()
    private var sections: [Section] = []
    private var modelState: [String: ModelState] = [:]

    private enum ModelState: Equatable { case ready, missing, downloading, failed }

    init() {
        super.init(style: .insetGrouped)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = L10n.tabDictation
        tableView.backgroundColor = Palette.background
        tableView.keyboardDismissMode = .interactive
        NotificationCenter.default.addObserver(self, selector: #selector(refresh), name: UIApplication.didBecomeActiveNotification, object: nil)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        refresh()
    }

    @objc private func refresh() {
        sections = [
            Section(title: nil, rows: [.languages, .target, .model], footer: nil),
            Section(title: L10n.settingsSectionKeyboard, rows: [.keyboard, .chineseTyping], footer: nil),
        ]
        tableView.reloadData()
        Task { await refreshModels() }
    }


    private func refreshModels() async {
        for language in preferences.languages where modelState[language] != .downloading {
            modelState[language] = await Recognizer.isInstalled(language) ? .ready : (modelState[language] == .failed ? .failed : .missing)
        }
        reload(.model)
    }

    private func reload(_ row: Row) {
        for (s, section) in sections.enumerated() {
            if let r = section.rows.firstIndex(of: row) {
                tableView.reloadRows(at: [IndexPath(row: r, section: s)], with: .none)
            }
        }
    }

    // MARK: Table

    override func numberOfSections(in tableView: UITableView) -> Int { sections.count }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { sections[section].rows.count }
    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? { sections[section].title }
    override func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? { sections[section].footer }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let row = sections[indexPath.section].rows[indexPath.row]
        let cell = UITableViewCell(style: .value1, reuseIdentifier: nil)
        cell.backgroundColor = Palette.card
        var content = UIListContentConfiguration.valueCell()
        content.textProperties.font = TextStyle.body.font
        content.textProperties.color = Palette.foreground
        content.secondaryTextProperties.font = TextStyle.body.font
        content.secondaryTextProperties.color = Palette.mutedForeground
        switch row {
        case .languages:
            content.text = L10n.settingsLanguages
            content.secondaryText = preferences.languages.map(DictationLanguage.mark).joined(separator: " · ")
            cell.accessoryType = .disclosureIndicator
        case .target:
            content.text = L10n.settingsTranslateInto
            content.secondaryText = preferences.translationTarget.name
            cell.accessoryType = .disclosureIndicator
        case .model:
            content.text = L10n.settingsModel
            let states = preferences.languages.compactMap { modelState[$0] }
            if states.contains(.downloading) {
                content.secondaryText = L10n.settingsModelDownloading
                let spinner = UIActivityIndicatorView(style: .medium)
                spinner.startAnimating()
                cell.accessoryView = spinner
            } else if states.contains(.failed) {
                content.secondaryText = L10n.settingsModelFailed
            } else if states.contains(.missing) {
                content.secondaryText = L10n.settingsModelDownload
                content.secondaryTextProperties.color = Palette.foreground
            } else if !states.isEmpty {
                content.secondaryText = L10n.settingsModelReady
            }
            cell.selectionStyle = states.contains(.missing) || states.contains(.failed) ? .default : .none
        case .keyboard:
            content.text = L10n.settingsKeyboardSetup
            let ready = preferences.keyboardLastSeen != nil
            content.secondaryText = ready ? L10n.settingsKeyboardReady : L10n.settingsKeyboardSetUp
            if !ready { content.secondaryTextProperties.color = Palette.foreground }
            cell.accessoryType = .disclosureIndicator
        case .chineseTyping:
            content.text = L10n.typingSetting
            content.secondaryText = ChineseTypingViewController.name(preferences.chineseTypingLayout)
            cell.accessoryType = .disclosureIndicator
        }
        if let icon = Self.icon(row) { RowIcon.apply(icon, to: &content) }
        // A value stays on the label's line; a long one truncates.
        content.prefersSideBySideTextAndSecondaryText = true
        content.secondaryTextProperties.numberOfLines = 1
        cell.contentConfiguration = content
        return cell
    }

    private static func icon(_ row: Row) -> Icon? {
        switch row {
        case .languages: .globe
        case .target: .languages
        case .model: .cpu
        case .keyboard: .keyboard
        case .chineseTyping: .type
        }
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch sections[indexPath.section].rows[indexPath.row] {
        case .languages: navigationController?.pushViewController(LanguagesViewController(), animated: true)
        case .target: navigationController?.pushViewController(TargetViewController(), animated: true)
        case .keyboard:
            let onboarding = OnboardingViewController(start: .keyboard)
            onboarding.modalPresentationStyle = .pageSheet
            present(onboarding, animated: true)
        case .model: downloadModels()
        case .chineseTyping: navigationController?.pushViewController(ChineseTypingViewController(), animated: true)
        }
    }

    private func downloadModels() {
        for language in preferences.languages where modelState[language] == .missing || modelState[language] == .failed {
            modelState[language] = .downloading
            Task {
                do {
                    try await Recognizer.install(language)
                    modelState[language] = .ready
                } catch {
                    modelState[language] = .failed
                }
                reload(.model)
            }
        }
        reload(.model)
    }
}
