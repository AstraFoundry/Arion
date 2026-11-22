import ArionKit
import UIKit

/// The AstraFoundry account: sign in, or the profile with usage sharing, the
/// account portal and Sign out.
final class AccountViewController: UITableViewController {
    private enum Row { case share, portal(String, String), signOut, version }

    private var account: AccountServing { Services.account }
    private var sections: [(title: String?, rows: [Row], footer: String?)] = []

    init() {
        super.init(style: .insetGrouped)
        title = L10n.accountTitle
        navigationItem.largeTitleDisplayMode = .never
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView.backgroundColor = Palette.background
        NotificationCenter.default.addObserver(self, selector: #selector(render), name: .accountChanged, object: nil)
        render()
    }

    @objc private func render() {
        let state = account.state
        let failure = state.failure.map(AccountCopy.failure)
        switch state.phase {
        case .signedOut, .signingIn:
            sections = []
            tableView.tableHeaderView = signInHeader(waiting: state.phase == .signingIn, failure: failure)
        case .signedIn, .restoring, .signingOut:
            tableView.tableHeaderView = profileHeader(state)
            sections = []
            if state.signedIn {
                sections.append((nil, [.share], nil))
                sections.append((L10n.accountPortal, [
                    .portal(L10n.accountPortalProfile, "profile"),
                    .portal(L10n.accountPortalSecurity, "security"),
                    .portal(L10n.accountPortalConnections, "connections"),
                    .portal(L10n.accountPortalProducts, "products"),
                    .portal(L10n.accountPortalData, "data"),
                ], nil))
            }
            sections.append((nil, [.signOut], failure))
        }
        sections.append((nil, [.version], nil))
        layoutHeader()
        tableView.reloadData()
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        layoutHeader()
    }

    /// Table headers size from their frame; fit it to the content.
    private func layoutHeader() {
        guard let header = tableView.tableHeaderView else { return }
        let size = header.systemLayoutSizeFitting(CGSize(width: tableView.bounds.width, height: 0), withHorizontalFittingPriority: .required, verticalFittingPriority: .fittingSizeLevel)
        if header.frame.size.height != size.height {
            header.frame.size = CGSize(width: tableView.bounds.width, height: size.height)
            tableView.tableHeaderView = header
        }
    }

    private func signInHeader(waiting: Bool, failure: String?) -> UIView {
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 8
        let title = UILabel(style: .title)
        title.text = L10n.accountSignInTitle
        title.accessibilityTraits = .header
        stack.addArrangedSubview(title)
        stack.setCustomSpacing(24, after: title)
        if waiting {
            let wait = UILabel(style: .body, color: Palette.secondaryForeground)
            wait.text = L10n.accountWaiting
            let cancel = OutlineButton(title: L10n.accountCancel, image: nil)
            cancel.addAction(UIAction { [weak self] _ in self?.account.cancel() }, for: .touchUpInside)
            stack.addArrangedSubview(wait)
            stack.setCustomSpacing(16, after: wait)
            stack.addArrangedSubview(cancel)
        } else {
            stack.addArrangedSubview(SignInMethods { [weak self] in self?.account.signIn(shareUsage: false) })
            stack.setCustomSpacing(16, after: stack.arrangedSubviews.last!)
            stack.addArrangedSubview(LinkedText(L10n.accountTermsNotice, links: [(L10n.accountTerms, SiteLinks.terms), (L10n.accountPrivacy, SiteLinks.privacy)]))
        }
        if let failure {
            let label = UILabel(style: .caption, color: Palette.danger)
            label.text = failure
            stack.setCustomSpacing(16, after: stack.arrangedSubviews.last!)
            stack.addArrangedSubview(label)
        }
        return wrap(stack, insets: UIEdgeInsets(top: 24, left: 20, bottom: 20, right: 20))
    }

    private func profileHeader(_ state: AccountState) -> UIView {
        let avatar = AvatarView(size: 72)
        avatar.show(state.profile)
        let name = UILabel(style: .title)
        name.textAlignment = .center
        name.text = state.profile.map { $0.name.isEmpty ? $0.email : $0.name }
        let detail = UILabel(style: .bodySmall, color: Palette.mutedForeground)
        detail.textAlignment = .center
        switch state.phase {
        case .restoring: detail.text = L10n.accountRestoring
        default: detail.text = state.profile?.email.isEmpty == false && state.profile?.name.isEmpty == false ? state.profile?.email : L10n.accountSignedIn
        }
        let stack = UIStackView(arrangedSubviews: [avatar, name, detail])
        stack.axis = .vertical
        stack.alignment = .center
        stack.spacing = 6
        stack.setCustomSpacing(14, after: avatar)
        return wrap(stack, insets: UIEdgeInsets(top: 16, left: 20, bottom: 12, right: 20))
    }

    private func wrap(_ stack: UIStackView, insets: UIEdgeInsets) -> UIView {
        let header = UIView()
        stack.translatesAutoresizingMaskIntoConstraints = false
        header.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: header.topAnchor, constant: insets.top),
            stack.bottomAnchor.constraint(equalTo: header.bottomAnchor, constant: -insets.bottom),
            // The header lines up with the inset cards below it.
            stack.leadingAnchor.constraint(equalTo: header.leadingAnchor, constant: insets.left),
            stack.trailingAnchor.constraint(equalTo: header.trailingAnchor, constant: -insets.right),
        ])
        return header
    }

    // MARK: Table

    override func numberOfSections(in tableView: UITableView) -> Int { sections.count }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { sections[section].rows.count }
    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? { sections[section].title }
    override func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? { sections[section].footer }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell()
        cell.backgroundColor = Palette.card
        var content = UIListContentConfiguration.cell()
        content.textProperties.font = TextStyle.body.font
        content.textProperties.color = Palette.foreground
        let state = account.state
        switch sections[indexPath.section].rows[indexPath.row] {
        case .share:
            content.text = L10n.accountUsageShare
            content.textProperties.numberOfLines = 0
            let toggle = UISwitch()
            toggle.isOn = state.usageSharing == .shared
            toggle.isEnabled = !state.busy && state.usageSharing != .loading
            toggle.addAction(UIAction { [weak self] action in
                self?.account.setUsageSharing((action.sender as? UISwitch)?.isOn ?? false)
            }, for: .valueChanged)
            cell.accessoryView = toggle
            cell.selectionStyle = .none
        case .portal(let title, _):
            content.text = title
            let mark = UIImageView(image: Icon.externalLink.image(size: 16))
            mark.tintColor = Palette.faint
            cell.accessoryView = mark
        case .version:
            content = UIListContentConfiguration.valueCell()
            content.text = L10n.settingsVersion
            content.textProperties.font = TextStyle.body.font
            content.secondaryText = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String
            content.secondaryTextProperties.font = TextStyle.body.font
            content.secondaryTextProperties.color = Palette.mutedForeground
            RowIcon.apply(.info, to: &content)
            cell.selectionStyle = .none
        case .signOut:
            content.text = L10n.accountSignOut
            content.textProperties.color = state.busy ? Palette.faint : Palette.danger
            content.textProperties.alignment = .center
        }
        cell.contentConfiguration = content
        return cell
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch sections[indexPath.section].rows[indexPath.row] {
        case .portal(_, let path):
            UIApplication.shared.open(AccountConfig().portal.appending(path: path))
        case .signOut where !account.state.busy || account.state.phase == .restoring:
            account.signOut()
        default:
            break
        }
    }
}

/// Muted caption text whose named words open documents.
final class LinkedText: UITextView {
    init(_ text: String, links: [(String, URL)]) {
        super.init(frame: .zero, textContainer: nil)
        let attributed = NSMutableAttributedString(string: text, attributes: [.font: TextStyle.caption.font, .foregroundColor: Palette.mutedForeground])
        for (word, url) in links {
            let range = (text as NSString).range(of: word)
            if range.location != NSNotFound { attributed.addAttribute(.link, value: url, range: range) }
        }
        attributedText = attributed
        linkTextAttributes = [.foregroundColor: Palette.foreground, .underlineStyle: NSUnderlineStyle.single.rawValue]
        isEditable = false
        isScrollEnabled = false
        backgroundColor = .clear
        textContainerInset = .zero
        textContainer.lineFragmentPadding = 0
        adjustsFontForContentSizeCategory = true
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}
