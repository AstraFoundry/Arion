import ArionKit
import UIKit

/// The polishing choice of onboarding and Settings: off, this device or a
/// cloud model, stored as the polish switch and the engine.
@MainActor
enum PolishChoice: CaseIterable {
    case off, device, cloud

    static func current(_ preferences: Preferences) -> PolishChoice {
        guard preferences.polishingEnabled else { return .off }
        return preferences.polishEngine == .cloud ? .cloud : .device
    }

    func apply(_ preferences: Preferences) {
        preferences.polishingEnabled = self != .off
        if self != .off { preferences.polishEngine = self == .cloud ? .cloud : .device }
    }

    var title: String {
        switch self {
        case .off: L10n.polishOff
        case .device: L10n.polishDevice
        case .cloud: L10n.polishCloud
        }
    }

    /// The option's detail; for this device, why it cannot run when it can't.
    var detail: String {
        switch self {
        case .off: L10n.polishOffDetail
        case .cloud: Self.cloudSummary ?? L10n.polishCloudDetail
        case .device:
            switch OnDeviceModel.availability {
            case .available: L10n.polishDeviceDetail
            case .turnedOff: L10n.settingsPolishTurnedOff
            case .unsupportedDevice: L10n.settingsPolishUnsupported
            case .preparing: L10n.settingsPolishPreparing
            }
        }
    }

    var available: Bool { self != .device || OnDeviceModel.availability != .unsupportedDevice }

    /// The provider and model in use, as one line.
    @MainActor static var cloudSummary: String? {
        guard let active = Services.cloud.active else { return nil }
        return [active.provider.name, active.model].compactMap(\.self).joined(separator: " · ")
    }
}

/// The three options as radio rows on a card, for onboarding.
final class PolishChoiceCard: Card {
    private var rows: [PolishChoice: UIControl] = [:]
    var onChoose: ((PolishChoice) -> Void)?
    private(set) var selected: PolishChoice

    init(selected: PolishChoice) {
        self.selected = selected
        super.init(padding: 0, spacing: 0)
        for (index, choice) in PolishChoice.allCases.enumerated() {
            if index > 0 {
                let line = UIView()
                line.backgroundColor = Palette.border
                line.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
                stack.addArrangedSubview(line)
            }
            let row = RadioRow(title: choice.title, detail: choice.detail)
            row.isEnabled = choice.available
            row.addAction(UIAction { [weak self] _ in self?.choose(choice) }, for: .touchUpInside)
            rows[choice] = row
            stack.addArrangedSubview(row)
        }
        paint()
    }

    private func choose(_ choice: PolishChoice) {
        selected = choice
        paint()
        onChoose?(choice)
    }

    private func paint() {
        for (choice, row) in rows { row.isSelected = choice == selected }
    }
}

/// A title and detail with a radio mark, the whole row one target.
final class RadioRow: UIControl {
    private let mark = UIImageView()
    private let title = UILabel(style: .body)
    private let detail = UILabel(style: .caption, color: Palette.mutedForeground)

    init(title text: String, detail note: String) {
        super.init(frame: .zero)
        title.text = text
        detail.text = note
        let labels = UIStackView(arrangedSubviews: [title, detail])
        labels.axis = .vertical
        labels.spacing = 2
        labels.isUserInteractionEnabled = false
        mark.tintColor = Palette.foreground
        mark.contentMode = .scaleAspectFit
        mark.translatesAutoresizingMaskIntoConstraints = false
        mark.widthAnchor.constraint(equalToConstant: 22).isActive = true
        mark.heightAnchor.constraint(equalToConstant: 22).isActive = true
        mark.setContentHuggingPriority(.required, for: .horizontal)
        let row = UIStackView(arrangedSubviews: [labels, mark])
        row.alignment = .center
        row.spacing = 12
        row.isUserInteractionEnabled = false
        row.translatesAutoresizingMaskIntoConstraints = false
        addSubview(row)
        NSLayoutConstraint.activate([
            row.topAnchor.constraint(equalTo: topAnchor, constant: 14),
            row.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -14),
            row.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            row.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
        ])
        isAccessibilityElement = true
        accessibilityLabel = text
        accessibilityHint = note
        paint()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override var isSelected: Bool { didSet { paint() } }
    override var isEnabled: Bool { didSet { paint() } }
    override var isHighlighted: Bool {
        didSet { backgroundColor = isHighlighted ? Palette.flatFace : .clear }
    }

    private func paint() {
        mark.image = (isSelected ? Icon.circleCheck : Icon.circle).image
        mark.tintColor = isSelected ? Palette.foreground : Palette.faint
        alpha = isEnabled ? 1 : 0.45
        accessibilityTraits = isSelected ? [.button, .selected] : .button
    }
}
