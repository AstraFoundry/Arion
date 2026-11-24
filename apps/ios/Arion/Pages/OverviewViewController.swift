import ArionKit
import UIKit

/// What the user has dictated, as the desktop's Overview: totals, words per
/// day, where polishing went and the latest vocabulary. Before the first
/// dictation, a first-run card with a field to try the keyboard in.
final class OverviewViewController: UIViewController {
    private let scroll = UIScrollView()
    private let content = UIStackView()
    private var period: ScenesCard.Period = .week
    private var shown: [UsageDay]?

    override func viewDidLoad() {
        super.viewDidLoad()
        title = L10n.tabOverview
        view.backgroundColor = Palette.background
        scroll.translatesAutoresizingMaskIntoConstraints = false
        scroll.alwaysBounceVertical = true
        scroll.keyboardDismissMode = .interactive
        content.axis = .vertical
        content.spacing = 12
        content.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scroll)
        scroll.addSubview(content)
        let frame = scroll.frameLayoutGuide
        let area = scroll.contentLayoutGuide
        NSLayoutConstraint.activate([
            scroll.topAnchor.constraint(equalTo: view.topAnchor),
            scroll.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            scroll.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            content.topAnchor.constraint(equalTo: area.topAnchor, constant: 8),
            content.bottomAnchor.constraint(equalTo: area.bottomAnchor, constant: -24),
            content.leadingAnchor.constraint(equalTo: frame.leadingAnchor, constant: 16),
            content.trailingAnchor.constraint(equalTo: frame.trailingAnchor, constant: -16),
        ])
        NotificationCenter.default.addObserver(self, selector: #selector(becameActive), name: UIApplication.didBecomeActiveNotification, object: nil)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        render(entering: true)
    }

    /// The keyboard records dictations while the app is away.
    @objc private func becameActive() {
        guard viewIfLoaded?.window != nil, Services.usage.days() != shown else { return }
        render(entering: true)
    }

    private func render(entering: Bool) {
        content.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let usage = Services.usage
        let days = usage.days()
        shown = days
        let terms = (try? Vocabulary().terms()) ?? []
        // Editing the vocabulary opens its tab.
        let edit = { [weak self] in
            guard let tabs = self?.tabBarController else { return }
            tabs.selectedIndex = tabs.viewControllers?.firstIndex { ($0 as? UINavigationController)?.viewControllers.first is VocabularyViewController } ?? tabs.selectedIndex
        }
        let total = usage.total(since: nil)
        guard total.sessions > 0 else {
            let cards = [FirstRunCard(), VocabularyCard(terms: terms, edit: edit)]
            cards.forEach(content.addArrangedSubview)
            if entering { Entrance.play(cards) }
            return
        }
        let speech = Figures.duration(milliseconds: total.speechMs)
        let stats = [
            StatCard(icon: .clock, label: L10n.overviewSpeechTime, value: speech.value, unit: speech.unit),
            StatCard(icon: .type, label: L10n.overviewWords, value: Figures.count(total.words), unit: L10n.unitWords),
            StatCard(icon: .sparkles, label: L10n.overviewPolished, value: Figures.count(total.polishedWords), unit: L10n.unitWords),
            StatCard(icon: .gauge, label: L10n.overviewSpeed, value: Figures.count(total.wordsPerMinute), unit: L10n.unitWordsPerMinute),
        ]
        for pair in [stats[0...1], stats[2...3]] {
            let row = UIStackView(arrangedSubviews: Array(pair))
            row.spacing = 12
            row.distribution = .fillEqually
            row.alignment = .fill
            content.addArrangedSubview(row)
        }
        let activity = ActivityCard(days: days, end: .now)
        let scenes = ScenesCard(usage: usage, period: period) { [weak self] in self?.period = $0 }
        let vocabulary = VocabularyCard(terms: terms, edit: edit)
        [activity, scenes, vocabulary].forEach(content.addArrangedSubview)
        if entering { Entrance.play(stats + [activity, scenes, vocabulary]) }
    }
}
