import ArionKit
import UIKit

/// The app's one screen: a field to try the keyboard in, then the settings.
final class HomeViewController: UITableViewController {
    private enum Row {
        case practice, languages, target, model, polish, vocabulary, history, keyboard, version
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
        title = L10n.appName
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
            Section(title: nil, rows: [.practice], footer: nil),
            Section(title: L10n.settingsSectionDictation, rows: [.languages, .target, .model], footer: L10n.settingsTranslateFooter),
            Section(title: nil, rows: [.polish], footer: polishFooter),
            Section(title: L10n.settingsSectionMemory, rows: [.vocabulary, .history], footer: nil),
            Section(title: L10n.settingsSectionKeyboard, rows: [.keyboard, .version], footer: nil),
        ]
        tableView.reloadData()
        Task { await refreshModels() }
    }

    private var polishFooter: String {
        switch OnDeviceModel.availability {
        case .available: L10n.settingsPolishFooter
        case .turnedOff: L10n.settingsPolishTurnedOff
        case .unsupportedDevice: L10n.settingsPolishUnsupported
        case .preparing: L10n.settingsPolishPreparing
        }
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
        case .practice:
            let field = PracticeField(placeholder: L10n.homePractice, bordered: false)
            field.backgroundColor = .clear
            cell.contentView.addSubview(field)
            NSLayoutConstraint.activate([
                field.topAnchor.constraint(equalTo: cell.contentView.topAnchor),
                field.bottomAnchor.constraint(equalTo: cell.contentView.bottomAnchor),
                field.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor),
                field.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor),
                field.heightAnchor.constraint(equalToConstant: 120),
            ])
            cell.selectionStyle = .none
            return cell
        case .languages:
            content.text = L10n.settingsLanguages
            content.secondaryText = preferences.languages.map(DictationLanguage.name).joined(separator: ", ")
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
            let toggle = UISwitch()
            toggle.isOn = preferences.polishingEnabled
            toggle.isEnabled = OnDeviceModel.availability != .unsupportedDevice
            toggle.addTarget(self, action: #selector(polishChanged(_:)), for: .valueChanged)
            cell.accessoryView = toggle
            cell.selectionStyle = .none
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
        cell.contentConfiguration = content
        return cell
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch sections[indexPath.section].rows[indexPath.row] {
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

    @objc private func polishChanged(_ toggle: UISwitch) {
        preferences.polishingEnabled = toggle.isOn
    }
}
