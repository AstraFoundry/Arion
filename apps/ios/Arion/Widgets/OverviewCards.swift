import ArionKit
import UIKit

/// One total: an icon and label over a large figure with its unit.
final class StatCard: Card {
    init(icon kind: Icon, label: String, value: String, unit: String) {
        super.init(padding: 16, spacing: 10)
        let icon = UIImageView(icon: kind, size: 14, color: Palette.mutedForeground)
        let title = UILabel(style: .caption, color: Palette.mutedForeground)
        title.text = label
        title.numberOfLines = 1
        title.adjustsFontSizeToFitWidth = true
        let top = UIStackView(arrangedSubviews: [icon, title])
        top.spacing = 6
        top.alignment = .center
        let figure = UILabel(style: .title)
        figure.font = UIFontMetrics(forTextStyle: .title1).scaledFont(for: .monospacedDigitSystemFont(ofSize: 28, weight: .semibold))
        figure.text = value
        figure.numberOfLines = 1
        figure.adjustsFontSizeToFitWidth = true
        let unitLabel = UILabel(style: .caption, color: Palette.mutedForeground)
        unitLabel.text = unit
        unitLabel.numberOfLines = 1
        unitLabel.setContentCompressionResistancePriority(.required, for: .horizontal)
        figure.setContentHuggingPriority(.required, for: .horizontal)
        unitLabel.setContentHuggingPriority(.required, for: .horizontal)
        // The unit follows its figure; the space after them is empty.
        let rest = UIView()
        rest.setContentHuggingPriority(.defaultLow, for: .horizontal)
        let bottom = UIStackView(arrangedSubviews: [figure, unitLabel, rest])
        bottom.spacing = 4
        bottom.alignment = .firstBaseline
        stack.addArrangedSubview(top)
        stack.addArrangedSubview(bottom)
        isAccessibilityElement = true
        accessibilityLabel = label
        accessibilityValue = "\(value) \(unit)"
    }
}

/// Words per day for recent weeks, a week per column with Monday on top,
/// shaded by each day's rank among the days with dictation, as the desktop.
final class ActivityCard: Card {
    private let grid: HeatmapView
    private let legend = UIStackView()
    private let detail = UILabel(style: .caption, color: Palette.mutedForeground)

    init(days: [UsageDay], end: Date) {
        grid = HeatmapView(days: days, end: end)
        super.init(padding: 16, spacing: 12)
        let less = UILabel(style: .caption, color: Palette.mutedForeground)
        less.text = L10n.overviewLess
        let more = UILabel(style: .caption, color: Palette.mutedForeground)
        more.text = L10n.overviewMore
        legend.spacing = 3
        legend.alignment = .center
        legend.addArrangedSubview(less)
        legend.setCustomSpacing(6, after: less)
        for color in HeatmapView.shades {
            let swatch = UIView()
            swatch.backgroundColor = color
            swatch.layer.cornerRadius = 2
            swatch.widthAnchor.constraint(equalToConstant: 10).isActive = true
            swatch.heightAnchor.constraint(equalToConstant: 10).isActive = true
            legend.addArrangedSubview(swatch)
        }
        legend.setCustomSpacing(6, after: legend.arrangedSubviews.last!)
        legend.addArrangedSubview(more)
        legend.isAccessibilityElement = false
        detail.isHidden = true
        detail.numberOfLines = 1
        let trailing = UIStackView(arrangedSubviews: [legend, detail])
        header(L10n.overviewActivity, trailing: trailing)
        stack.addArrangedSubview(grid)
        // Touching a day names its words in place of the legend.
        grid.onSelect = { [weak self] text in
            guard let self else { return }
            self.detail.text = text
            self.detail.isHidden = text == nil
            self.legend.isHidden = text != nil
        }
    }
}

final class HeatmapView: UIView {
    static let shades: [UIColor] = [
        Palette.border,
        Palette.primary.withAlphaComponent(0.3),
        Palette.primary.withAlphaComponent(0.62),
        Palette.primary,
        Palette.accentForeground,
    ]
    private static let maxWeeks = 26
    private static let minCell: CGFloat = 11
    private static let maxPitch: CGFloat = 18
    private static let gap: CGFloat = 3
    private static let dayColumn: CGFloat = 22
    private static let monthRow: CGFloat = 18

    var onSelect: ((String?) -> Void)?
    private let days: [UsageDay]
    private let end: Date
    private var selected: Int?
    private var cells: [CALayer] = []
    private var labels: [CATextLayer] = []
    private var height: NSLayoutConstraint!
    private let calendar = Calendar.current

    init(days log: [UsageDay], end: Date) {
        self.end = end
        // The last full weeks, Monday first, ending today.
        let today = (calendar.component(.weekday, from: end) + 5) % 7
        let count = (Self.maxWeeks - 1) * 7 + today + 1
        let byDate = Dictionary(log.map { ($0.date, $0) }, uniquingKeysWith: { a, _ in a })
        let formatter = Self.dayKey
        days = (0..<count).map { index in
            let date = Calendar.current.date(byAdding: .day, value: index - count + 1, to: end) ?? end
            let key = formatter.string(from: date)
            return byDate[key] ?? UsageDay(date: key)
        }
        super.init(frame: .zero)
        translatesAutoresizingMaskIntoConstraints = false
        height = heightAnchor.constraint(equalToConstant: Self.monthRow + 7 * Self.maxPitch)
        height.isActive = true
        isAccessibilityElement = true
        accessibilityLabel = L10n.overviewActivity
        let recent = days.suffix(7).reduce(0) { $0 + $1.words }
        accessibilityValue = "\(L10n.overviewRangeWeek): \(Figures.count(recent)) \(L10n.unitWords)"
        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (self: HeatmapView, _) in self.setNeedsLayout() }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    static let dayKey: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    private struct Grid { var x: CGFloat; var pitch: CGFloat; var cell: CGFloat; var weeks: Int }

    private func grid(for width: CGFloat) -> Grid {
        let room = width - Self.dayColumn + Self.gap
        let weeks = min(Self.maxWeeks, max(1, Int(room / (Self.minCell + Self.gap))))
        let pitch = min(room / CGFloat(weeks), Self.maxPitch)
        return Grid(x: Self.dayColumn + room - pitch * CGFloat(weeks), pitch: pitch, cell: pitch - Self.gap, weeks: weeks)
    }

    private func index(week: Int, day: Int, weeks: Int) -> Int? {
        let index = (Self.maxWeeks - weeks + week) * 7 + day
        return index < days.count ? index : nil
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        (cells + labels).forEach { $0.removeFromSuperlayer() }
        cells = []
        labels = []
        guard bounds.width > 0 else { return }
        let g = grid(for: bounds.width)
        height.constant = Self.monthRow + 7 * g.pitch - Self.gap
        var visible: [Int] = []
        for week in 0..<g.weeks {
            for day in 0..<7 { if let i = index(week: week, day: day, weeks: g.weeks), days[i].words > 0 { visible.append(days[i].words) } }
        }
        visible.sort()
        let scale = traitCollection.displayScale
        let caption = UIFont.systemFont(ofSize: 11)
        func label(_ text: String, x: CGFloat, y: CGFloat, trailing: Bool = false) {
            let layer = CATextLayer()
            layer.string = text
            layer.font = caption
            layer.fontSize = caption.pointSize
            layer.foregroundColor = Palette.mutedForeground.resolvedColor(with: traitCollection).cgColor
            layer.contentsScale = scale
            let size = (text as NSString).size(withAttributes: [.font: caption])
            // A weekday ends just before the grid, however wide the card.
            layer.frame = CGRect(x: trailing ? x - ceil(size.width) : x, y: y, width: ceil(size.width), height: ceil(size.height))
            self.layer.addSublayer(layer)
            labels.append(layer)
        }
        let weekdays = calendar.veryShortWeekdaySymbols
        for day in stride(from: 0, to: 7, by: 2) {
            label(weekdays[(day + 1) % 7], x: g.x - 6, y: Self.monthRow + CGFloat(day) * g.pitch + (g.cell - 13) / 2, trailing: true)
        }
        let months = calendar.shortMonthSymbols
        var previous = 0
        var labelEnd: CGFloat = -1
        for week in 0..<g.weeks {
            guard let i = index(week: week, day: 0, weeks: g.weeks), let date = Self.dayKey.date(from: days[i].date) else { continue }
            let month = calendar.component(.month, from: date)
            guard month != previous else { continue }
            previous = month
            let text = months[month - 1]
            let x = g.x + CGFloat(week) * g.pitch
            let width = (text as NSString).size(withAttributes: [.font: caption]).width
            if x > labelEnd + 6, x + width <= bounds.width {
                label(text, x: x, y: 0)
                labelEnd = x + width
            }
        }
        let radius = min(SunbeamTokens.Radius.sm / 2 + 1, g.cell / 4)
        for week in 0..<g.weeks {
            for day in 0..<7 {
                guard let i = index(week: week, day: day, weeks: g.weeks) else { continue }
                let cell = CALayer()
                cell.frame = CGRect(x: g.x + CGFloat(week) * g.pitch, y: Self.monthRow + CGFloat(day) * g.pitch, width: g.cell, height: g.cell)
                cell.cornerRadius = radius
                cell.backgroundColor = Self.shades[Self.level(days[i].words, visible)].resolvedColor(with: traitCollection).cgColor
                if i == selected {
                    cell.borderWidth = 1
                    cell.borderColor = Palette.foreground.withAlphaComponent(0.55).resolvedColor(with: traitCollection).cgColor
                }
                layer.addSublayer(cell)
                cells.append(cell)
            }
        }
    }

    /// The busiest quarter of the visible days is level 4.
    static func level(_ words: Int, _ active: [Int]) -> Int {
        guard words > 0, !active.isEmpty else { return 0 }
        let rank = active.firstIndex(where: { $0 > words }) ?? active.count
        return min(4, max(1, Int(ceil(Double(4 * rank) / Double(active.count)))))
    }

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent?) { select(touches) }
    override func touchesMoved(_ touches: Set<UITouch>, with event: UIEvent?) { select(touches) }

    private func select(_ touches: Set<UITouch>) {
        guard let point = touches.first?.location(in: self) else { return }
        let g = grid(for: bounds.width)
        let week = Int((point.x - g.x) / g.pitch)
        let day = Int((point.y - Self.monthRow) / g.pitch)
        guard point.x >= g.x, point.y >= Self.monthRow, week < g.weeks, day < 7, let i = index(week: week, day: day, weeks: g.weeks) else { return }
        if selected == i {
            selected = nil
            onSelect?(nil)
        } else {
            selected = i
            let date = Self.dayKey.date(from: days[i].date).map { $0.formatted(.dateTime.month().day()) } ?? days[i].date
            onSelect?(L10n.overviewDayWords(date: date, words: Figures.count(days[i].words)))
        }
        setNeedsLayout()
    }
}

/// Where a period's words went: polished words by scene, then the words left
/// as recognized.
final class ScenesCard: Card {
    enum Period: CaseIterable {
        case today, week, all

        var title: String {
            switch self {
            case .today: L10n.overviewRangeToday
            case .week: L10n.overviewRangeWeek
            case .all: L10n.overviewRangeAll
            }
        }

        func start(_ now: Date) -> Date? {
            switch self {
            case .today: now
            case .week: Calendar.current.date(byAdding: .day, value: -6, to: now)
            case .all: nil
            }
        }
    }

    private let rows = UIStackView()
    private let picker = UIButton(type: .system)
    private let usage: UsageReading
    private var period: Period

    init(usage: UsageReading, period: Period, onChange: @escaping (Period) -> Void) {
        self.usage = usage
        self.period = period
        super.init(padding: 16, spacing: 14)
        var config = UIButton.Configuration.plain()
        config.contentInsets = .zero
        config.imagePadding = 4
        config.image = Icon.chevronsUpDown.image(size: 14)
        config.imagePlacement = .trailing
        config.baseForegroundColor = Palette.mutedForeground
        picker.configuration = config
        picker.showsMenuAsPrimaryAction = true
        picker.setContentHuggingPriority(.required, for: .horizontal)
        header(L10n.overviewScenes, trailing: picker)
        rows.axis = .vertical
        rows.spacing = 14
        stack.addArrangedSubview(rows)
        configure(onChange: onChange)
    }

    private func configure(onChange: @escaping (Period) -> Void) {
        var title = AttributedString(period.title)
        title.font = TextStyle.caption.font
        picker.configuration?.attributedTitle = title
        picker.menu = UIMenu(children: Period.allCases.map { option in
            UIAction(title: option.title, state: option == period ? .on : .off) { [weak self] _ in
                guard let self else { return }
                self.period = option
                onChange(option)
                self.configure(onChange: onChange)
            }
        })
        fill()
    }

    private func fill() {
        rows.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let total = usage.total(since: period.start(.now))
        guard total.words > 0 else {
            let empty = UILabel(style: .bodySmall, color: Palette.mutedForeground)
            empty.text = L10n.overviewScenesEmpty
            rows.addArrangedSubview(empty)
            return
        }
        var shares: [(Icon, String, Int)] = total.scenes
            .sorted { $0.value != $1.value ? $0.value > $1.value : $0.key < $1.key }
            .compactMap { key, words in Scene(rawValue: key).map { (SceneName.icon($0), SceneName.label($0), words) } }
        let plain = total.words - total.polishedWords
        if plain > 0 { shares.append((.type, L10n.overviewUnpolished, plain)) }
        for (icon, label, words) in shares {
            rows.addArrangedSubview(ShareRow(icon: icon, label: label, words: words, part: CGFloat(words) / CGFloat(total.words)))
        }
    }
}

private final class ShareRow: UIStackView {
    private let fill = UIView()
    private let part: CGFloat
    private var width: NSLayoutConstraint?

    init(icon kind: Icon, label text: String, words: Int, part: CGFloat) {
        self.part = part
        super.init(frame: .zero)
        axis = .vertical
        spacing = 6
        let icon = UIImageView(icon: kind, size: 14, color: Palette.mutedForeground)
        let label = UILabel(style: .bodySmall)
        label.text = text
        label.numberOfLines = 1
        let count = UILabel(style: .caption, color: Palette.mutedForeground)
        count.text = Figures.count(words)
        count.font = UIFontMetrics(forTextStyle: .footnote).scaledFont(for: .monospacedDigitSystemFont(ofSize: 13, weight: .medium))
        count.setContentHuggingPriority(.required, for: .horizontal)
        let top = UIStackView(arrangedSubviews: [icon, label, count])
        top.spacing = 6
        top.alignment = .center
        let track = UIView()
        track.backgroundColor = Palette.border
        track.layer.cornerRadius = 3
        track.heightAnchor.constraint(equalToConstant: 6).isActive = true
        fill.backgroundColor = Palette.primary
        fill.layer.cornerRadius = 3
        fill.translatesAutoresizingMaskIntoConstraints = false
        track.addSubview(fill)
        NSLayoutConstraint.activate([
            fill.leadingAnchor.constraint(equalTo: track.leadingAnchor),
            fill.topAnchor.constraint(equalTo: track.topAnchor),
            fill.bottomAnchor.constraint(equalTo: track.bottomAnchor),
        ])
        width = fill.widthAnchor.constraint(equalTo: track.widthAnchor, multiplier: max(part, 0.02))
        width?.isActive = true
        addArrangedSubview(top)
        addArrangedSubview(track)
        isAccessibilityElement = true
        accessibilityLabel = text
        accessibilityValue = "\(Figures.count(words)) \(L10n.unitWords)"
    }

    @available(*, unavailable)
    required init(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}

/// Scene names and symbols, as the desktop's scene feature.
enum SceneName {
    static func label(_ scene: Scene) -> String {
        switch scene {
        case .chat: L10n.sceneChat
        case .work: L10n.sceneWork
        case .email: L10n.sceneEmail
        case .prompt: L10n.scenePrompt
        case .list: L10n.sceneList
        case .search: L10n.sceneSearch
        }
    }

    static func icon(_ scene: Scene) -> Icon {
        switch scene {
        case .chat: .messageCircle
        case .work: .briefcase
        case .email: .mail
        case .prompt: .code
        case .list: .list
        case .search: .search
        }
    }
}

/// Stands in for the totals until the first dictation: what is missing and
/// what to do, in one two-tone thought, a field to try it, and the way in.
final class FirstRunCard: Card {
    init() {
        super.init(padding: 20, spacing: 0)
        let title = UILabel(style: .title)
        title.text = L10n.overviewFirstTitle
        let prompt = UILabel(style: .title, color: Palette.mutedForeground)
        prompt.text = L10n.overviewFirstPrompt
        stack.addArrangedSubview(title)
        stack.addArrangedSubview(prompt)
        stack.setCustomSpacing(20, after: prompt)
        let field = PracticeField(placeholder: L10n.overviewFirstHint)
        field.backgroundColor = Palette.background
        field.heightAnchor.constraint(equalToConstant: 112).isActive = true
        stack.addArrangedSubview(field)
    }
}

/// The latest vocabulary terms and the way to edit them.
final class VocabularyCard: Card {
    init(terms: [String], edit: @escaping () -> Void) {
        super.init(padding: 16, spacing: 10)
        var config = UIButton.Configuration.plain()
        config.contentInsets = .zero
        var title = AttributedString(L10n.overviewVocabularyEdit)
        title.font = TextStyle.label.font
        config.attributedTitle = title
        config.baseForegroundColor = Palette.foreground
        let button = UIButton(configuration: config, primaryAction: UIAction { _ in edit() })
        button.setContentHuggingPriority(.required, for: .horizontal)
        header(L10n.overviewVocabulary, trailing: button)
        if terms.isEmpty {
            let empty = UILabel(style: .bodySmall, color: Palette.mutedForeground)
            empty.text = L10n.overviewVocabularyEmptyTitle
            stack.addArrangedSubview(empty)
            return
        }
        for term in terms.suffix(5).reversed() {
            let label = UILabel(style: .body)
            label.text = term
            label.numberOfLines = 1
            stack.addArrangedSubview(label)
        }
    }
}
