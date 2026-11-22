import ArionKit
import UIKit

/// Settings: the account first, as in the system's Settings, then
/// dictation, polishing, memory and the keyboard.
final class SettingsViewController: UITableViewController {
    private enum Row {
        case account, languages, target, model, polish, vocabulary, history, keyboard, version
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
        title = L10n.tabSettings
        tableView.backgroundColor = Palette.background
        tableView.keyboardDismissMode = .interactive
        NotificationCenter.default.addObserver(self, selector: #selector(refresh), name: UIApplication.didBecomeActiveNotification, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(accountChanged), name: .accountChanged, object: nil)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        refresh()
    }

    @objc private func refresh() {
        sections = [
            Section(title: nil, rows: [.account], footer: nil),
            Section(title: L10n.settingsSectionDictation, rows: [.languages, .target, .model], footer: L10n.settingsTranslateFooter),
            Section(title: nil, rows: [.polish], footer: nil),
            Section(title: L10n.settingsSectionMemory, rows: [.vocabulary, .history], footer: nil),
            Section(title: L10n.settingsSectionKeyboard, rows: [.keyboard, .version], footer: nil),
        ]
        tableView.reloadData()
        Task { await refreshModels() }
    }

    @objc private func accountChanged() { reload(.account) }

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
        case .account:
            return AccountCell(state: Services.account.state)
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
        case .polish:
            content.text = L10n.settingsPolish
            let choice = PolishChoice.current(preferences)
            content.secondaryText = choice == .cloud ? (PolishChoice.cloudSummary ?? L10n.cloudNotConnected) : choice.title
            cell.accessoryType = .disclosureIndicator
        case .vocabulary:
            content.text = L10n.settingsVocabulary
            content.secondaryText = ((try? Vocabulary().terms().count) ?? 0).description
            cell.accessoryType = .disclosureIndicator
        case .history:
            content.text = L10n.settingsHistory
            cell.accessoryType = .disclosureIndicator
        case .keyboard:
            content.text = L10n.settingsKeyboardSetup
            let ready = preferences.keyboardLastSeen != nil
            content.secondaryText = ready ? L10n.settingsKeyboardReady : L10n.settingsKeyboardSetUp
            if !ready { content.secondaryTextProperties.color = Palette.foreground }
            cell.accessoryType = .disclosureIndicator
        case .version:
            content.text = L10n.settingsVersion
            content.secondaryText = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String
            cell.selectionStyle = .none
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
        case .account: nil
        case .languages: .globe
        case .target: .languages
        case .model: .cpu
        case .polish: .sparkles
        case .vocabulary: .bookOpen
        case .history: .history
        case .keyboard: .keyboard
        case .version: .info
        }
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch sections[indexPath.section].rows[indexPath.row] {
        case .account: navigationController?.pushViewController(AccountViewController(), animated: true)
        case .polish: navigationController?.pushViewController(PolishViewController(), animated: true)
        case .languages: navigationController?.pushViewController(LanguagesViewController(), animated: true)
        case .target: navigationController?.pushViewController(TargetViewController(), animated: true)
        case .vocabulary: navigationController?.pushViewController(VocabularyViewController(), animated: true)
        case .history: navigationController?.pushViewController(HistoryViewController(), animated: true)
        case .keyboard:
            let onboarding = OnboardingViewController(start: .keyboard)
            onboarding.modalPresentationStyle = .pageSheet
            present(onboarding, animated: true)
        case .model: downloadModels()
        default: break
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

/// The account row: the picture or initials, then the name and email, or
/// the person glyph and "Sign in" while signed out.
private final class AccountCell: UITableViewCell {
    init(state: AccountState) {
        super.init(style: .default, reuseIdentifier: nil)
        backgroundColor = Palette.card
        accessoryType = .disclosureIndicator
        let avatar = AvatarView(size: 44)
        let profile = state.phase == .signedOut || state.phase == .signingIn ? nil : state.profile
        avatar.show(profile)
        let name = UILabel(style: .body)
        name.font = .systemFont(ofSize: 17, weight: .semibold)
        let detail = UILabel(style: .caption, color: Palette.mutedForeground)
        if let profile {
            name.text = profile.name.isEmpty ? profile.email : profile.name
            detail.text = state.phase == .restoring ? L10n.accountRestoring : (profile.name.isEmpty ? L10n.accountSignedIn : profile.email)
        } else {
            name.text = L10n.accountSignInTitle
            detail.isHidden = true
        }
        let labels = UIStackView(arrangedSubviews: [name, detail])
        labels.axis = .vertical
        labels.spacing = 2
        let row = UIStackView(arrangedSubviews: [avatar, labels])
        row.spacing = 14
        row.alignment = .center
        row.translatesAutoresizingMaskIntoConstraints = false
        contentView.addSubview(row)
        NSLayoutConstraint.activate([
            row.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 10),
            row.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -10),
            row.leadingAnchor.constraint(equalTo: contentView.layoutMarginsGuide.leadingAnchor),
            row.trailingAnchor.constraint(equalTo: contentView.layoutMarginsGuide.trailingAnchor),
        ])
        isAccessibilityElement = true
        accessibilityLabel = [name.text, detail.text].compactMap(\.self).joined(separator: ", ")
        accessibilityTraits = .button
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}
