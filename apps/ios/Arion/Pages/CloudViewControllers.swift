import ArionKit
import UIKit

/// Settings › Polish: off, this device or cloud, and the cloud model in use.
final class PolishViewController: SettingsListController {
    init() { super.init(title: L10n.polishTitle) }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        tableView.reloadData()
    }

    private var showsCloud: Bool { PolishChoice.current(preferences) == .cloud }

    override func numberOfSections(in tableView: UITableView) -> Int { showsCloud ? 2 : 1 }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { section == 0 ? PolishChoice.allCases.count : 1 }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if indexPath.section == 1 {
            let cell = cell(L10n.cloudTitle, secondary: PolishChoice.cloudSummary ?? L10n.cloudNotConnected, icon: Services.cloud.active.map { RowIcon.provider($0.provider) } ?? .server)
            cell.accessoryType = .disclosureIndicator
            return cell
        }
        let choice = PolishChoice.allCases[indexPath.row]
        let cell = cell(choice.title, secondary: choice == .cloud ? nil : choice.detail, checked: PolishChoice.current(preferences) == choice)
        if !choice.available {
            cell.selectionStyle = .none
            cell.contentView.alpha = 0.45
        }
        return cell
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        if indexPath.section == 1 {
            navigationController?.pushViewController(CloudViewController(), animated: true)
            return
        }
        let choice = PolishChoice.allCases[indexPath.row]
        guard choice.available else { return }
        choice.apply(preferences)
        tableView.reloadData()
        if choice == .cloud, Services.cloud.active == nil {
            navigationController?.pushViewController(CloudViewController(), animated: true)
        }
    }
}

/// The cloud providers: the one in use with its model, then every provider
/// that takes an API key. Opening a provider never switches to it.
final class CloudViewController: SettingsListController {
    private enum Row { case active, model, disconnect, provider(CloudProvider) }

    private var sections: [(title: String?, rows: [Row], footer: String?)] = []
    private var cloud: CloudServing { Services.cloud }
    private var failure: CloudFailure?
    /// Set by onboarding: a new connection ends the setup instead of
    /// returning to this list.
    private let onConnected: (() -> Void)?

    init(onConnected: (() -> Void)? = nil) {
        self.onConnected = onConnected
        super.init(title: L10n.cloudTitle)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        render()
    }

    private func render() {
        sections = []
        if cloud.active != nil {
            sections.append((L10n.cloudInUse, [.active, .model, .disconnect], failure.map(CloudCopy.failure)))
        }
        let others = CloudProvider.allCases.filter { $0 != cloud.active?.provider }
        sections.append((L10n.cloudApiKeys, others.map(Row.provider), nil))
        tableView.reloadData()
    }

    override func numberOfSections(in tableView: UITableView) -> Int { sections.count }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { sections[section].rows.count }
    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? { sections[section].title }
    override func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? { sections[section].footer }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        switch sections[indexPath.section].rows[indexPath.row] {
        case .active:
            let active = cloud.active!
            let cell = cell(active.provider.name, secondary: active.baseURL?.host(), checked: true, icon: RowIcon.provider(active.provider))
            cell.selectionStyle = .none
            return cell
        case .model:
            let cell = UITableViewCell(style: .value1, reuseIdentifier: nil)
            cell.backgroundColor = Palette.card
            var content = UIListContentConfiguration.valueCell()
            content.text = L10n.cloudModel
            content.textProperties.font = TextStyle.body.font
            RowIcon.apply(.cpu, to: &content)
            content.secondaryText = cloud.active?.model ?? L10n.cloudModelPlaceholder
            content.secondaryTextProperties.font = TextStyle.body.font
            content.secondaryTextProperties.color = Palette.mutedForeground
            cell.contentConfiguration = content
            cell.accessoryType = .disclosureIndicator
            return cell
        case .disconnect:
            let cell = cell(L10n.cloudDisconnect)
            var content = cell.contentConfiguration as? UIListContentConfiguration
            content?.textProperties.color = Palette.danger
            // A destructive action is centered, as Sign out; it has no icon column.
            content?.textProperties.alignment = .center
            cell.contentConfiguration = content
            return cell
        case .provider(let provider):
            let saved = cloud.saved.contains(provider)
            let cell = cell(provider.name, secondary: saved ? L10n.cloudSaved : nil, icon: RowIcon.provider(provider))
            if saved {
                var config = UIButton.Configuration.gray()
                var title = AttributedString(L10n.cloudUse)
                title.font = TextStyle.label.font
                config.attributedTitle = title
                config.baseForegroundColor = Palette.foreground
                config.baseBackgroundColor = Palette.secondary
                config.cornerStyle = .capsule
                config.contentInsets = NSDirectionalEdgeInsets(top: 5, leading: 12, bottom: 5, trailing: 12)
                let use = UIButton(configuration: config, primaryAction: UIAction { [weak self] _ in self?.use(provider) })
                use.sizeToFit()
                cell.accessoryView = use
            } else {
                cell.accessoryType = .disclosureIndicator
            }
            return cell
        }
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch sections[indexPath.section].rows[indexPath.row] {
        case .model:
            navigationController?.pushViewController(CloudModelsViewController(), animated: true)
        case .disconnect:
            do {
                try cloud.disconnect()
                failure = nil
            } catch {
                failure = error
            }
            render()
        case .provider(let provider):
            navigationController?.pushViewController(CloudProviderViewController(provider: provider, onConnected: onConnected), animated: true)
        case .active:
            break
        }
    }

    private func use(_ provider: CloudProvider) {
        do {
            try cloud.activate(provider)
            failure = nil
        } catch {
            failure = error
        }
        render()
        if failure == nil, cloud.active?.model != nil { onConnected?() }
    }
}

/// One provider: its key (and base URL for a compatible endpoint) and Connect.
final class CloudProviderViewController: UIViewController {
    private let provider: CloudProvider
    private let key = FormField(label: L10n.cloudKey, placeholder: L10n.cloudKeyPlaceholder, secure: true)
    private let address = FormField(label: L10n.cloudBaseURL, placeholder: L10n.cloudBaseURLPlaceholder, secure: false)
    private let connect = FaceButton(kind: .primary)
    private let message = UILabel(style: .caption, color: Palette.danger)
    private let spinner = UIActivityIndicatorView(style: .medium)
    private var task: Task<Void, Never>?
    private let onConnected: (() -> Void)?

    init(provider: CloudProvider, onConnected: (() -> Void)? = nil) {
        self.provider = provider
        self.onConnected = onConnected
        super.init(nibName: nil, bundle: nil)
        title = provider.name
        navigationItem.largeTitleDisplayMode = .never
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = Palette.background
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        if provider.needsBaseURL { stack.addArrangedSubview(address) }
        stack.addArrangedSubview(key)
        connect.title = L10n.cloudConnect
        connect.heightAnchor.constraint(equalToConstant: 50).isActive = true
        connect.addTarget(self, action: #selector(submit), for: .touchUpInside)
        stack.setCustomSpacing(24, after: key)
        stack.addArrangedSubview(connect)
        message.isHidden = true
        stack.addArrangedSubview(message)
        spinner.hidesWhenStopped = true
        stack.addArrangedSubview(spinner)
        let footer = UILabel(style: .caption, color: Palette.mutedForeground)
        footer.text = L10n.cloudFooter
        stack.addArrangedSubview(footer)
        view.addSubview(stack)
        let guide = view.safeAreaLayoutGuide
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: guide.topAnchor, constant: 24),
            stack.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: 20),
            stack.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -20),
        ])
        key.onEdit = { [weak self] in self?.message.isHidden = true }
        address.onEdit = { [weak self] in self?.message.isHidden = true }
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        (provider.needsBaseURL ? address : key).field.becomeFirstResponder()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        task?.cancel()
    }

    @objc private func submit() {
        let secret = key.field.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !secret.isEmpty, task == nil else { return }
        view.endEditing(true)
        connect.isEnabled = false
        message.isHidden = true
        spinner.startAnimating()
        let base = provider.needsBaseURL ? address.field.text : nil
        task = Task { [weak self, provider] in
            do throws(CloudFailure) {
                _ = try await Services.cloud.connect(provider, key: secret, baseURL: base)
                guard let self, !Task.isCancelled else { return }
                if let onConnected = self.onConnected { onConnected() } else { self.navigationController?.popViewController(animated: true) }
            } catch {
                guard let self, !Task.isCancelled else { return }
                self.message.text = CloudCopy.failure(error)
                self.message.isHidden = false
                UIAccessibility.post(notification: .announcement, argument: self.message.text)
            }
            self?.spinner.stopAnimating()
            self?.connect.isEnabled = true
            self?.task = nil
        }
    }
}

/// The provider's models, the chosen one checked.
final class CloudModelsViewController: SettingsListController {
    private var models: [String] = []
    private var failure: CloudFailure?
    private var loading = true

    init() { super.init(title: L10n.cloudModel) }

    override func viewDidLoad() {
        super.viewDidLoad()
        load()
    }

    private func load() {
        loading = true
        failure = nil
        tableView.reloadData()
        Task {
            do throws(CloudFailure) {
                models = try await Services.cloud.models()
            } catch {
                failure = error
            }
            loading = false
            tableView.reloadData()
        }
    }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        loading ? 1 : (failure != nil ? 1 : models.count)
    }

    override func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? {
        failure.map(CloudCopy.failure)
    }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if loading {
            let cell = cell("")
            let spinner = UIActivityIndicatorView(style: .medium)
            spinner.startAnimating()
            cell.accessoryView = spinner
            cell.selectionStyle = .none
            return cell
        }
        if failure != nil { return cell(L10n.cloudRetry) }
        let model = models[indexPath.row]
        return cell(model, checked: model == Services.cloud.active?.model)
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        guard !loading else { return }
        if failure != nil {
            load()
            return
        }
        Services.cloud.choose(model: models[indexPath.row])
        tableView.reloadData()
        navigationController?.popViewController(animated: true)
    }
}

/// A labelled text field on a flat card.
final class FormField: UIView, UITextFieldDelegate {
    let field = UITextField()
    var onEdit: (() -> Void)?

    init(label: String, placeholder: String, secure: Bool) {
        super.init(frame: .zero)
        let title = UILabel(style: .label, color: Palette.secondaryForeground)
        title.text = label
        let box = UIView()
        box.backgroundColor = Palette.card
        box.layer.cornerRadius = SunbeamTokens.Radius.xl
        box.layer.cornerCurve = .continuous
        box.layer.borderWidth = 1
        box.layer.borderColor = Palette.border.cgColor
        field.placeholder = placeholder
        field.font = TextStyle.body.font
        field.textColor = Palette.foreground
        field.isSecureTextEntry = secure
        field.autocorrectionType = .no
        field.autocapitalizationType = .none
        field.spellCheckingType = .no
        field.keyboardType = secure ? .asciiCapable : .URL
        field.textContentType = secure ? .password : .URL
        field.accessibilityLabel = label
        field.delegate = self
        field.addAction(UIAction { [weak self] _ in self?.onEdit?() }, for: .editingChanged)
        field.translatesAutoresizingMaskIntoConstraints = false
        box.addSubview(field)
        NSLayoutConstraint.activate([
            box.heightAnchor.constraint(equalToConstant: 48),
            field.leadingAnchor.constraint(equalTo: box.leadingAnchor, constant: 14),
            field.trailingAnchor.constraint(equalTo: box.trailingAnchor, constant: -14),
            field.centerYAnchor.constraint(equalTo: box.centerYAnchor),
        ])
        let stack = UIStackView(arrangedSubviews: [title, box])
        stack.axis = .vertical
        stack.spacing = 8
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor),
        ])
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: FormField, _) in
            box.layer.borderColor = Palette.border.resolvedColor(with: self.traitCollection).cgColor
        }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}

enum CloudCopy {
    static func failure(_ failure: CloudFailure) -> String {
        switch failure {
        case .rejected: L10n.cloudFailRejected
        case .network: L10n.cloudFailNetwork
        case .models: L10n.cloudFailModels
        case .store: L10n.cloudFailStore
        case .address: L10n.cloudFailAddress
        }
    }
}
