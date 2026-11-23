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

/// Gain on the voice before recognition, for a quiet microphone or voice.
final class GainViewController: UIViewController {
    static func name(_ decibels: Int) -> String {
        L10n.settingsGainValue(decibels: decibels > 0 ? "+\(decibels)" : String(decibels))
    }

    static func stop(_ stop: MicrophoneGain.Stop) -> String {
        L10n.settingsGainStop(gain: name(stop.decibels), distance: L10n.settingsGainCentimeters(value: String(stop.centimeters)))
    }

    private let preferences = Preferences()
    private let value = UILabel()
    private let slider = UISlider()
    private let ticks = GainTicks()
    private let feedback = UISelectionFeedbackGenerator()

    init() {
        super.init(nibName: nil, bundle: nil)
        title = L10n.settingsGain
        navigationItem.largeTitleDisplayMode = .never
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = Palette.background
        let stops = MicrophoneGain.stops
        slider.minimumValue = Float(stops[0].decibels)
        slider.maximumValue = Float(stops[stops.count - 1].decibels)
        slider.value = Float(preferences.microphoneGain)
        slider.minimumTrackTintColor = Palette.foreground
        slider.accessibilityLabel = L10n.settingsGain
        slider.addTarget(self, action: #selector(slid), for: .valueChanged)
        slider.addTarget(self, action: #selector(settle), for: [.touchUpInside, .touchUpOutside, .touchCancel])
        value.font = TextStyle.body.font
        value.textColor = Palette.foreground
        value.numberOfLines = 0

        let card = UIStackView(arrangedSubviews: [value, slider, ticks])
        card.axis = .vertical
        card.spacing = 12
        card.setCustomSpacing(4, after: slider)
        card.isLayoutMarginsRelativeArrangement = true
        card.directionalLayoutMargins = NSDirectionalEdgeInsets(top: 20, leading: 20, bottom: 16, trailing: 20)
        card.backgroundColor = Palette.card
        card.layer.cornerRadius = 24
        card.layer.cornerCurve = .continuous
        card.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(card)
        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 24),
            card.leadingAnchor.constraint(equalTo: view.layoutMarginsGuide.leadingAnchor),
            card.trailingAnchor.constraint(equalTo: view.layoutMarginsGuide.trailingAnchor),
        ])
        ticks.slider = slider
        show(MicrophoneGain.nearest(slider.value))
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        ticks.setNeedsLayout()
    }

    /// Steps like the desktop's slider: the value jumps from stop to stop.
    @objc private func slid() {
        let stop = MicrophoneGain.nearest(slider.value)
        slider.value = Float(stop.decibels)
        guard stop.decibels != preferences.microphoneGain else { return }
        preferences.microphoneGain = stop.decibels
        feedback.selectionChanged()
        show(stop)
    }

    @objc private func settle() { slider.setValue(Float(preferences.microphoneGain), animated: true) }

    private func show(_ stop: MicrophoneGain.Stop) {
        value.text = Self.stop(stop)
        slider.accessibilityValue = value.text
        ticks.selected = stop.decibels
    }
}

/// The speaking distance under each stop, in centimeters as the headline
/// names them, centered where the slider's thumb sits for that stop.
private final class GainTicks: UIView {
    weak var slider: UISlider?
    private let labels: [UILabel] = MicrophoneGain.stops.map { stop in
        let label = UILabel()
        label.text = String(stop.centimeters)
        label.font = TextStyle.caption.font
        label.isAccessibilityElement = false
        return label
    }

    var selected = 0 { didSet { restyle() } }

    override init(frame: CGRect) {
        super.init(frame: frame)
        labels.forEach(addSubview)
        restyle()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override var intrinsicContentSize: CGSize { CGSize(width: UIView.noIntrinsicMetric, height: TextStyle.caption.font.lineHeight) }

    private func restyle() {
        for (label, stop) in zip(labels, MicrophoneGain.stops) {
            label.textColor = stop.decibels == selected ? Palette.foreground : Palette.mutedForeground
        }
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        guard let slider else { return }
        let track = slider.trackRect(forBounds: slider.bounds)
        for (label, stop) in zip(labels, MicrophoneGain.stops) {
            let thumb = slider.thumbRect(forBounds: slider.bounds, trackRect: track, value: Float(stop.decibels))
            label.sizeToFit()
            label.center = CGPoint(x: convert(CGPoint(x: thumb.midX, y: 0), from: slider).x, y: bounds.midY)
        }
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
