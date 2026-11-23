import UIKit

/// What the typing keyboard reports; the input view controller edits the field.
@MainActor
public protocol TypingKeyboardViewDelegate: AnyObject {
    /// Text to insert into the editor.
    func typingInsert(_ text: String)
    /// Delete before the caret (nothing was composing).
    func typingDeleteBackward()
    /// The return key with nothing composing.
    func typingReturn()
    /// 中/EN changed; the controller remembers it.
    func typingLayoutChanged(_ layout: TypingLayout)
    /// The switch key: back to the microphone.
    func typingShowVoice()
    /// The hide key, top right on both panels.
    func typingDismiss()
    /// The space bar held and dragged: move the caret, as on the system keyboard.
    func typingMoveCaret(by offset: Int)
    /// The editor's text before the caret, for English completions.
    func typingTextBeforeCaret() -> String?
}

/// Pinyin, 9-key and English typing on the Rime engine (ADR 053). A
/// composition stays in the keyboard; only committed text reaches the editor.
/// The bottom row starts with the switch key, where the voice panel has its
/// own, so moving between voice and typing never moves it.
public final class TypingKeyboardView: UIView {
    public weak var delegate: TypingKeyboardViewDelegate?

    private let engine: TypingEngine
    private let candidateBar = CandidateBar()
    /// Every candidate in rows over the keys, opened by the chevron that
    /// takes the voice key's corner while composing.
    private let grid = CandidateGrid()
    private var expanded = false
    /// English: the word before the caret and the lexicon's candidates for it.
    private var englishWord = ""
    private var englishCandidates: [String] = []
    private let keys = UIStackView()
    private let syllables = SyllableColumn()
    /// Shift is one-shot; a quick second tap locks capitals, as on iOS.
    private enum Shift { case off, once, locked }
    private var shift = Shift.off
    private var shifted: Bool { shift != .off }
    private var lastShiftTap = Date.distantPast
    private var trackpadAnchor: CGFloat?
    /// The symbols page shown: none (letters), 123, or #+=.
    private enum Page { case letters, numbers, more }
    private var page = Page.letters
    private var symbolsShown: Bool { page != .letters }
    private let alternates = AlternatesPopup()
    private var deleteTimer: Timer?
    private var returnKeys: [FaceButton] = []
    /// Key widths relate keys to the rows stack, so they wait until the row is in it.
    private var pendingWidths: [NSLayoutConstraint] = []
    private let feedback = UIImpactFeedbackGenerator(style: .light)

    /// The globe key; the controller routes it to the system's keyboard list.
    public let globe = FaceButton(kind: .secondary, material: .raised, icon: .globe)
    /// Shown when the device needs its own next-keyboard key.
    public var showsGlobe = false { didSet { if oldValue != showsGlobe { rebuild() } } }
    /// The Chinese layout 中 opens: the user's choice in settings.
    public var chineseLayout: TypingLayout = .pinyin {
        didSet {
            guard chineseLayout != oldValue, engine.layout != .english else { return }
            select(chineseLayout)
        }
    }

    /// The field's return word (搜索, 发送…), or nil for a plain new line,
    /// which the key shows as an icon.
    public var returnTitle: String? { didSet { if returnTitle != oldValue { rebuild() } } }
    /// False when the field enables return only once it has text and it has none.
    public var returnEnabled = true { didSet { render() } }
    private let dismiss = KeyboardTopKey(icon: .chevronDown, label: L10n.keyboardDismiss)
    private let voice = KeyboardTopKey(icon: .mic, label: L10n.typingVoice)
    private let expand = KeyboardTopKey(icon: .chevronDown, label: L10n.typingMoreCandidates)

    public init(engine: TypingEngine) {
        self.engine = engine
        super.init(frame: .zero)
        build()
        rebuild()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    public var layout: TypingLayout { engine.layout }

    public func select(_ layout: TypingLayout) {
        engine.select(layout)
        page = .letters
        shift = .off
        rebuild()
        delegate?.typingLayoutChanged(layout)
    }

    /// Ends a composition without inserting it, e.g. when the field changes.
    public func reset() {
        engine.clear()
        render()
    }

    private func build() {
        dismiss.addAction(UIAction { [weak self] _ in self?.delegate?.typingDismiss() }, for: .touchUpInside)
        voice.addAction(UIAction { [weak self] _ in
            self?.commitComposition()
            self?.delegate?.typingShowVoice()
        }, for: .touchUpInside)
        globe.accessibilityLabel = L10n.keyboardNextKeyboard
        globe.sideInset = 0
        globe.minimumFaceHeight = 0
        globe.cornerRadius = 8
        globe.iconSize = Self.iconSize
        candidateBar.onChoose = { [weak self] index in self?.choose(index) }
        candidateBar.loadMore = { [weak self] offset in self?.engine.candidates(from: offset, limit: TypingEngine.barCandidates) ?? [] }
        grid.onChoose = { [weak self] index in self?.choose(index) }
        grid.loadMore = { [weak self] offset in self?.engine.candidates(from: offset, limit: TypingEngine.barCandidates) ?? [] }
        grid.setRail(up: railKey(.chevronUp, label: L10n.typingPreviousPage),
                     down: railKey(.chevronDown, label: L10n.typingNextPage),
                     delete: keyView(KeyLayout.Key(.delete)))
        expand.addAction(UIAction { [weak self] _ in
            guard let self else { return }
            self.expanded.toggle()
            self.render()
        }, for: .touchUpInside)
        syllables.onChoose = { [weak self] syllable in
            self?.engine.chooseSyllable(syllable)
            self?.render()
        }
        syllables.onPunctuation = { [weak self] text in self?.delegate?.typingInsert(text) }
        keys.axis = .vertical
        keys.spacing = KeyboardMetrics.rowGap
        grid.isHidden = true
        for view in [candidateBar, dismiss, voice, expand, keys, grid] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        let height = heightAnchor.constraint(equalToConstant: KeyboardMetrics.height)
        height.priority = .defaultHigh
        let inset = KeyboardMetrics.sideInset
        NSLayoutConstraint.activate([
            height,
            candidateBar.topAnchor.constraint(equalTo: topAnchor, constant: 4),
            candidateBar.leadingAnchor.constraint(equalTo: dismiss.trailingAnchor, constant: 4),
            candidateBar.trailingAnchor.constraint(equalTo: voice.leadingAnchor, constant: -4),
            candidateBar.heightAnchor.constraint(equalToConstant: 44),
            keys.topAnchor.constraint(equalTo: candidateBar.bottomAnchor, constant: 6),
            keys.leadingAnchor.constraint(equalTo: leadingAnchor, constant: inset),
            keys.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -inset),
            keys.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -KeyboardMetrics.bottomInset),
            grid.topAnchor.constraint(equalTo: keys.topAnchor),
            grid.leadingAnchor.constraint(equalTo: keys.leadingAnchor),
            grid.trailingAnchor.constraint(equalTo: keys.trailingAnchor),
            grid.bottomAnchor.constraint(equalTo: keys.bottomAnchor),
        ])
        dismiss.pin(in: self, trailing: false)
        voice.pin(in: self, trailing: true)
        expand.pin(in: self, trailing: true)
    }

    // MARK: Keys

    private func rebuild() {
        keys.arrangedSubviews.forEach { $0.removeFromSuperview() }
        returnKeys = []
        let chinese = engine.layout != .english
        if symbolsShown {
            for row in KeyLayout.symbols(chinese: chinese, more: page == .more) { keys.addArrangedSubview(rowView(row)) }
        } else if engine.layout == .pinyin9 {
            keys.addArrangedSubview(keypad())
        } else {
            for row in KeyLayout.qwerty(engine.layout) { keys.addArrangedSubview(rowView(row)) }
        }
        let keypadPage = engine.layout == .pinyin9 && !symbolsShown
        keys.addArrangedSubview(rowView(KeyLayout.bottom(symbols: symbolsShown, globe: showsGlobe, keypad: keypadPage)))
        NSLayoutConstraint.activate(pendingWidths)
        pendingWidths = []
        // Every row is one key high; the keypad block spans three.
        for view in keys.arrangedSubviews {
            let rows: CGFloat = view === keys.arrangedSubviews.first && keypadPage ? 3 : 1
            view.heightAnchor.constraint(equalToConstant: rows * KeyboardMetrics.rowHeight + (rows - 1) * KeyboardMetrics.rowGap).isActive = true
        }
        render()
    }

    /// 9-key: syllables or punctuation on the left, the keypad, then delete
    /// over a tall return, as on the system's 9-key keyboard.
    private func keypad() -> UIView {
        let grid = UIStackView(arrangedSubviews: KeyLayout.keypad.map { gridRow($0) })
        grid.axis = .vertical
        grid.spacing = keys.spacing
        grid.distribution = .fillEqually
        let delete = keyView(KeyLayout.Key(.delete))
        let enter = keyView(KeyLayout.Key(.return))
        let side = UIStackView(arrangedSubviews: [delete, enter])
        side.axis = .vertical
        side.spacing = keys.spacing
        enter.heightAnchor.constraint(equalTo: delete.heightAnchor, multiplier: 2, constant: keys.spacing).isActive = true
        let row = UIStackView(arrangedSubviews: [syllables, grid, side])
        row.axis = .horizontal
        row.spacing = 6
        syllables.widthAnchor.constraint(equalTo: row.widthAnchor, multiplier: 0.17).isActive = true
        side.widthAnchor.constraint(equalTo: row.widthAnchor, multiplier: 0.17).isActive = true
        return row
    }

    /// A row sized like the system keyboard's: keys in letter widths, the
    /// space bar taking the rest, Shift and Delete at the edges, and a row
    /// narrower than the keyboard centered.
    private func rowView(_ row: [KeyLayout.Key]) -> UIView {
        let stack = UIStackView()
        stack.axis = .horizontal
        stack.spacing = KeyboardMetrics.gap
        var views: [UIView] = []
        for key in row {
            let view: UIView = key.key == .globe ? globe : keyView(key)
            if key.width > 0 { pendingWidths.append(KeyboardMetrics.width(view, letters: key.width, of: keys)) }
            views.append(view)
        }
        if row.last?.key == .delete, views.count > 2 {
            // The system keyboard's third row: the left key and Delete against
            // the edges. Letters keep their width, centered; punctuation on the
            // symbols pages shares the space equally.
            let letters = UIStackView(arrangedSubviews: Array(views[1..<(views.count - 1)]))
            letters.axis = .horizontal
            letters.spacing = KeyboardMetrics.gap
            if row[1].width == 0 {
                letters.distribution = .fillEqually
                for view in [views[0], letters, views[views.count - 1]] { stack.addArrangedSubview(view) }
                return stack
            }
            let left = UIView(), right = UIView()
            for view in [views[0], left, letters, right, views[views.count - 1]] { stack.addArrangedSubview(view) }
            stack.spacing = 0
            right.widthAnchor.constraint(equalTo: left.widthAnchor).isActive = true
            return stack
        }
        views.forEach { stack.addArrangedSubview($0) }
        guard !row.contains(where: { $0.width == 0 }) else { return stack }
        let holder = UIView()
        stack.translatesAutoresizingMaskIntoConstraints = false
        holder.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: holder.topAnchor),
            stack.bottomAnchor.constraint(equalTo: holder.bottomAnchor),
            stack.centerXAnchor.constraint(equalTo: holder.centerXAnchor),
        ])
        return holder
    }

    /// The 9-key grid's rows: equal keys filling the grid's width.
    private func gridRow(_ row: [KeyLayout.Key]) -> UIView {
        let stack = UIStackView(arrangedSubviews: row.map { keyView($0) })
        stack.axis = .horizontal
        stack.spacing = KeyboardMetrics.gap
        stack.distribution = .fillEqually
        return stack
    }

    /// A raised key with only an icon, for the candidate grid's rail.
    private func railKey(_ icon: Icon, label: String) -> FaceButton {
        let button = FaceButton(kind: .secondary, material: .raised, icon: icon)
        button.sideInset = 0
        button.minimumFaceHeight = 0
        button.cornerRadius = 8
        button.iconSize = Self.iconSize
        button.accessibilityLabel = label
        button.addAction(UIAction { [weak self] _ in self?.feedback.impactOccurred() }, for: .touchDown)
        return button
    }

    private func keyView(_ spec: KeyLayout.Key) -> FaceButton {
        if spec.key == .globe { return globe }
        let button = FaceButton(kind: .secondary, material: .raised)
        button.sideInset = 0
        button.minimumFaceHeight = 0
        button.cornerRadius = 8
        button.iconSize = Self.iconSize
        configure(button, spec)
        button.titleLabel.font = Self.font(for: spec.key, title: button.title)
        if spec.key == .return { returnKeys.append(button) }
        button.addAction(UIAction { [weak self] _ in self?.press(spec.key) }, for: .touchUpInside)
        if spec.key == .delete {
            button.addTarget(self, action: #selector(deleteDown(_:)), for: .touchDown)
            button.addTarget(self, action: #selector(deleteUp(_:)), for: [.touchUpInside, .touchUpOutside, .touchCancel])
        }
        if let options = alternatesFor(spec), !options.isEmpty {
            let hold = UILongPressGestureRecognizer(target: self, action: #selector(alternatesHeld(_:)))
            hold.minimumPressDuration = 0.4
            button.addGestureRecognizer(hold)
        }
        if spec.key == .space {
            // Hold the space bar, then drag to move the caret, as on iOS.
            let hold = UILongPressGestureRecognizer(target: self, action: #selector(trackpad(_:)))
            hold.minimumPressDuration = 0.35
            button.addGestureRecognizer(hold)
        }
        return button
    }

    private func configure(_ button: FaceButton, _ spec: KeyLayout.Key) {
        switch spec.key {
        case .letter(let c):
            let label = spec.label ?? String(c)
            button.title = shifted ? label.uppercased() : label
        case .text(let text): button.title = spec.label ?? text
        case .separator: button.title = spec.label ?? L10n.typingSeparator
        case .shift:
            // Outline when off, filled while on, the barred arrow when locked.
            let icon: Icon = shift == .locked ? .arrowBigUpDash : .arrowBigUp
            button.icon = icon
            button.iconImage = shifted ? KeyGlyph.filled(icon, size: Self.iconSize) : nil
            button.accessibilityLabel = L10n.typingShift
            button.accessibilityTraits = shifted ? [.button, .selected] : .button
        case .delete:
            button.icon = .delete
            button.accessibilityLabel = L10n.keyboardDelete
        case .globe: break
        case .space:
            // As on the system keyboard: no word, a faint mode mark in the corner.
            button.accessibilityLabel = L10n.keyboardSpace
            let mark = UILabel()
            mark.text = engine.layout == .english ? L10n.typingEnglish : L10n.typingPinyinMark
            mark.font = .systemFont(ofSize: 11)
            mark.textColor = Palette.mutedForeground
            mark.isAccessibilityElement = false
            mark.translatesAutoresizingMaskIntoConstraints = false
            button.addSubview(mark)
            NSLayoutConstraint.activate([
                mark.trailingAnchor.constraint(equalTo: button.trailingAnchor, constant: -7),
                mark.bottomAnchor.constraint(equalTo: button.bottomAnchor, constant: -4),
            ])
        case .return: showReturn(on: button)
        case .symbols: button.title = L10n.typingSymbols
        case .moreSymbols: button.title = L10n.typingMoreSymbols
        case .letters: button.title = engine.layout == .english ? L10n.typingLettersEnglish : L10n.typingLettersChinese
        case .layout:
            button.title = engine.layout == .english ? L10n.typingEnglish : L10n.typingChinese
            button.accessibilityLabel = L10n.typingLayout
        }
    }

    /// One type scale for every key: a single character reads like a
    /// letter, a word like a label, and icons match the words.
    public static let iconSize: CGFloat = 20
    static func font(for key: TypingKey, title: String?) -> UIFont {
        let glyph: Bool
        switch key {
        case .letter, .text: glyph = (title?.count ?? 0) == 1
        default: glyph = false
        }
        return .systemFont(ofSize: glyph ? 22 : 17)
    }

    /// While Pinyin composes, return inserts the letters as typed (确认).
    /// Otherwise it follows the field like the system key: a named action
    /// (发送, 搜索…) is the yellow key, a plain new line a gray icon, and a
    /// field that needs text first keeps it dimmed until there is some.
    private func showReturn(on button: FaceButton) {
        let composing = engine.state.isComposing
        let title = composing ? L10n.typingConfirm : returnTitle
        button.kind = title == nil ? .secondary : .primary
        button.isEnabled = composing || returnEnabled
        button.title = title
        button.icon = title == nil ? .cornerDownLeft : nil
        button.accessibilityLabel = title ?? L10n.returnDefault
        button.titleLabel.font = Self.font(for: .return, title: title)
    }

    private func press(_ key: TypingKey) {
        feedback.impactOccurred()
        UIDevice.current.playInputClick()
        switch key {
        case .letter(let c):
            if shifted, engine.layout == .pinyin {
                // A capital in Pinyin is inserted as is, after what was composing.
                commitComposition()
                delegate?.typingInsert(c.uppercased())
            } else {
                let character = shifted ? Character(c.uppercased()) : c
                if let text = engine.type(character), !text.isEmpty { delegate?.typingInsert(text) }
            }
            if shift == .once { shift = .off; rebuild() }
        case .text(let text):
            commitComposition()
            delegate?.typingInsert(text)
        case .separator:
            if engine.state.isComposing { engine.type("'") } else if engine.layout == .pinyin9 { delegate?.typingInsert("，") } else { delegate?.typingInsert("'") }
        case .shift:
            let now = Date()
            if shift == .once, now.timeIntervalSince(lastShiftTap) < 0.3 { shift = .locked } else { shift = shift == .off ? .once : .off }
            lastShiftTap = now
            rebuild()
            return
        case .delete, .globe:
            break
        case .space:
            if engine.state.isComposing, !engine.state.candidates.isEmpty {
                choose(0)
                return
            }
            delegate?.typingInsert(" ")
        case .return:
            if engine.state.isComposing {
                if let raw = engine.commitRaw() { delegate?.typingInsert(raw) }
            } else {
                delegate?.typingReturn()
            }
        case .symbols:
            page = .numbers
            rebuild()
            return
        case .moreSymbols:
            page = .more
            rebuild()
            return
        case .letters:
            page = .letters
            rebuild()
            return
        case .layout:
            // 中 and EN only; full keyboard or 9-key is a setting, not a cycle.
            commitComposition()
            select(engine.layout == .english ? chineseLayout : .english)
            return
        }
        render()
    }

    /// Inserts what is composing as typed, before punctuation or a layout change.
    private func commitComposition() {
        if engine.state.isComposing, let raw = engine.commitRaw() { delegate?.typingInsert(raw) }
    }

    private func choose(_ index: Int) {
        if engine.layout == .english {
            // The chosen word replaces the one typed, then a space, as on the system keyboard.
            guard englishCandidates.indices.contains(index) else { return }
            let word = englishCandidates[index]
            for _ in englishWord { delegate?.typingDeleteBackward() }
            delegate?.typingInsert(word + " ")
            engine.confirmEnglish(word)
            render()
            return
        }
        if let text = engine.choose(index), !text.isEmpty { delegate?.typingInsert(text) }
        render()
    }

    private func deleteOnce() {
        if engine.backspace() {
            render()
        } else {
            delegate?.typingDeleteBackward()
            if engine.layout == .english { render() }
        }
    }

    @objc private func deleteDown(_ key: FaceButton) {
        key.iconImage = KeyGlyph.filled(.delete, size: Self.iconSize)
        deleteOnce()
        deleteTimer?.invalidate()
        deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.45, repeats: false) { [weak self] _ in
            MainActor.assumeIsolated {
                self?.deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.08, repeats: true) { _ in
                    MainActor.assumeIsolated { self?.deleteOnce() }
                }
            }
        }
    }

    @objc private func deleteUp(_ key: FaceButton) {
        key.iconImage = nil
        deleteTimer?.invalidate()
        deleteTimer = nil
    }

    /// The alternates of a key, shifted with it; Pinyin letters compose, so
    /// only English letters offer accents.
    private func alternatesFor(_ spec: KeyLayout.Key) -> [String]? {
        switch spec.key {
        case .letter(let c) where engine.layout == .english:
            let list = KeyLayout.alternates[String(c)]
            return shifted ? list?.map { $0.uppercased() } : list
        case .text(let text): return KeyLayout.alternates[text]
        default: return nil
        }
    }

    /// Hold a key for its alternates, slide to one and release to insert it,
    /// as on the system keyboard.
    @objc private func alternatesHeld(_ gesture: UILongPressGestureRecognizer) {
        guard let key = gesture.view as? FaceButton else { return }
        switch gesture.state {
        case .began:
            let base = key.title ?? ""
            let options = (base.count == 1 ? alternatesFor(KeyLayout.Key(.text(base))) : nil)
                ?? alternatesFor(KeyLayout.Key(.letter(Character(base.lowercased())))) ?? []
            guard !options.isEmpty else { return }
            feedback.impactOccurred()
            alternates.show(options, above: key, in: self)
            alternates.track(gesture.location(in: alternates))
        case .changed:
            alternates.track(gesture.location(in: alternates))
        case .ended:
            if let choice = alternates.choice {
                commitComposition()
                delegate?.typingInsert(choice)
                if shift == .once { shift = .off; rebuild() }
                render()
            }
            alternates.dismiss()
        default:
            alternates.dismiss()
        }
    }

    /// Trackpad mode: the keys fade, and every 9 points of drag moves the
    /// caret one character. A composition is inserted first, as on iOS.
    @objc private func trackpad(_ gesture: UILongPressGestureRecognizer) {
        let x = gesture.location(in: self).x
        switch gesture.state {
        case .began:
            commitComposition()
            render()
            trackpadAnchor = x
            feedback.impactOccurred()
            UIView.animate(withDuration: 0.15) { self.keys.alpha = 0.35 }
        case .changed:
            guard let anchor = trackpadAnchor else { return }
            let steps = Int((x - anchor) / 9)
            if steps != 0 {
                delegate?.typingMoveCaret(by: steps)
                trackpadAnchor = anchor + CGFloat(steps) * 9
            }
        default:
            trackpadAnchor = nil
            UIView.animate(withDuration: 0.15) { self.keys.alpha = 1 }
        }
    }

    /// The letters before the caret and their candidates: the word as typed
    /// first, as the system keyboard keeps it, then the lexicon's.
    private func refreshEnglish() {
        let before = symbolsShown ? "" : delegate?.typingTextBeforeCaret() ?? ""
        englishWord = String(before.reversed().prefix { $0.isASCII && $0.isLetter }.reversed())
        guard !englishWord.isEmpty else { englishCandidates = []; return }
        englishCandidates = [englishWord] + engine.englishCandidates(for: englishWord).filter { $0 != englishWord }
    }

    private func render() {
        let state = engine.state
        let choosing = state.isComposing && !state.candidates.isEmpty
        if !choosing { expanded = false }
        if engine.layout == .english {
            refreshEnglish()
            candidateBar.show(preedit: "", candidates: englishCandidates)
        } else {
            // Open, the grid continues from the first candidate; the bar keeps the composition.
            candidateBar.show(preedit: state.preedit, candidates: expanded ? [] : state.candidates)
        }
        voice.isHidden = choosing
        expand.isHidden = !choosing
        expand.setIcon(expanded ? .chevronUp : .chevronDown,
                       label: expanded ? L10n.typingFewerCandidates : L10n.typingMoreCandidates)
        keys.isHidden = expanded
        grid.isHidden = !expanded
        if expanded { grid.show(state.candidates) }
        if engine.layout == .pinyin9 { syllables.show(state.syllables, idle: KeyLayout.keypadPunctuation) }
        for key in returnKeys { showReturn(on: key) }
    }
}

/// The candidates above the keys: the composition, then a scrolling row.
final class CandidateBar: UIView {
    var onChoose: ((Int) -> Void)?
    var loadMore: ((Int) -> [String])?
    private let preedit = UILabel(style: .caption, color: Palette.mutedForeground)
    private let scroll = UIScrollView()
    private let row = UIStackView()
    private var count = 0

    override init(frame: CGRect) {
        super.init(frame: frame)
        preedit.numberOfLines = 1
        scroll.showsHorizontalScrollIndicator = false
        scroll.translatesAutoresizingMaskIntoConstraints = false
        row.axis = .horizontal
        row.spacing = 2
        row.translatesAutoresizingMaskIntoConstraints = false
        scroll.addSubview(row)
        let stack = UIStackView(arrangedSubviews: [preedit, scroll])
        stack.axis = .vertical
        stack.spacing = 0
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor),
            row.topAnchor.constraint(equalTo: scroll.contentLayoutGuide.topAnchor),
            row.bottomAnchor.constraint(equalTo: scroll.contentLayoutGuide.bottomAnchor),
            row.leadingAnchor.constraint(equalTo: scroll.contentLayoutGuide.leadingAnchor),
            row.trailingAnchor.constraint(equalTo: scroll.contentLayoutGuide.trailingAnchor),
            row.heightAnchor.constraint(equalTo: scroll.frameLayoutGuide.heightAnchor),
        ])
        scroll.delegate = self
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    func show(preedit text: String, candidates: [String]) {
        preedit.text = text
        preedit.isHidden = text.isEmpty
        row.arrangedSubviews.forEach { $0.removeFromSuperview() }
        count = 0
        append(candidates)
        scroll.contentOffset = .zero
    }

    private func append(_ candidates: [String]) {
        for text in candidates {
            let index = count
            let button = UIButton(configuration: .plain())
            button.configuration?.title = text
            button.configuration?.baseForegroundColor = index == 0 ? Palette.foreground : Palette.foreground.withAlphaComponent(0.85)
            button.configuration?.contentInsets = NSDirectionalEdgeInsets(top: 4, leading: 10, bottom: 4, trailing: 10)
            button.configuration?.titleTextAttributesTransformer = UIConfigurationTextAttributesTransformer { attributes in
                var next = attributes
                next.font = UIFont.systemFont(ofSize: 20, weight: index == 0 ? .medium : .regular)
                return next
            }
            button.addAction(UIAction { [weak self] _ in self?.onChoose?(index) }, for: .touchUpInside)
            row.addArrangedSubview(button)
            count += 1
        }
    }
}

extension CandidateBar: UIScrollViewDelegate {
    // Candidates load a page at a time as the row scrolls toward its end.
    func scrollViewDidScroll(_ scrollView: UIScrollView) {
        guard count > 0, scrollView.contentOffset.x + scrollView.bounds.width > scrollView.contentSize.width - 80,
              let more = loadMore?(count), !more.isEmpty else { return }
        append(more)
    }
}

/// The expanded candidates: rows of cells a quarter of the width wide, a
/// longer candidate taking as many quarters as it needs, as Rime frontends
/// (Hamster, Trime) and Sogou lay them out; Delete stays on the right.
/// More candidates load as the grid scrolls toward its end.
final class CandidateGrid: UIView {
    var onChoose: ((Int) -> Void)?
    var loadMore: ((Int) -> [String])?
    private let scroll = CandidateScroll()
    private let rail = UIView()
    private var cells: [UIButton] = []
    private var lines: [UIView] = []
    private static let columns: CGFloat = 4
    private static let rowHeight: CGFloat = 48
    private static let font = UIFont.systemFont(ofSize: 20)

    override init(frame: CGRect) {
        super.init(frame: frame)
        scroll.alwaysBounceVertical = true
        scroll.delaysContentTouches = false
        scroll.delegate = self
        for view in [scroll, rail] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        NSLayoutConstraint.activate([
            scroll.topAnchor.constraint(equalTo: topAnchor),
            scroll.bottomAnchor.constraint(equalTo: bottomAnchor),
            scroll.leadingAnchor.constraint(equalTo: leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: rail.leadingAnchor, constant: -KeyboardMetrics.gap),
            rail.topAnchor.constraint(equalTo: topAnchor),
            rail.bottomAnchor.constraint(equalTo: bottomAnchor),
            rail.trailingAnchor.constraint(equalTo: trailingAnchor),
            rail.widthAnchor.constraint(equalTo: widthAnchor, multiplier: 0.135, constant: -0.9 * KeyboardMetrics.gap * 1.35),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    /// The right rail: page up and page down, as Rime pages its menu, then
    /// Delete at the bottom, each one key high.
    func setRail(up: UIControl, down: UIControl, delete: UIView) {
        up.addAction(UIAction { [weak self] _ in self?.page(by: -1) }, for: .touchUpInside)
        down.addAction(UIAction { [weak self] _ in self?.page(by: 1) }, for: .touchUpInside)
        let stack = UIStackView(arrangedSubviews: [up, down, UIView(), delete])
        stack.axis = .vertical
        stack.spacing = KeyboardMetrics.rowGap
        stack.translatesAutoresizingMaskIntoConstraints = false
        rail.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: rail.topAnchor),
            stack.bottomAnchor.constraint(equalTo: rail.bottomAnchor),
            stack.leadingAnchor.constraint(equalTo: rail.leadingAnchor),
            stack.trailingAnchor.constraint(equalTo: rail.trailingAnchor),
        ] + [up, down, delete].map { $0.heightAnchor.constraint(equalToConstant: KeyboardMetrics.rowHeight) })
    }

    /// Scrolls a page of whole rows, loading more at the end.
    private func page(by direction: CGFloat) {
        let rows = max(1, (scroll.bounds.height / Self.rowHeight).rounded(.down))
        if direction > 0, let more = loadMore?(cells.count), !more.isEmpty {
            append(more)
            place()
        }
        let limit = max(0, scroll.contentSize.height - scroll.bounds.height)
        let target = min(limit, max(0, scroll.contentOffset.y + direction * rows * Self.rowHeight))
        scroll.setContentOffset(CGPoint(x: 0, y: target), animated: !UIAccessibility.isReduceMotionEnabled)
    }

    func show(_ candidates: [String]) {
        cells.forEach { $0.removeFromSuperview() }
        cells = []
        append(candidates)
        scroll.contentOffset = .zero
        setNeedsLayout()
    }

    private func append(_ candidates: [String]) {
        for text in candidates {
            let index = cells.count
            let button = UIButton(configuration: .plain())
            button.configuration?.title = text
            button.configuration?.baseForegroundColor = Palette.foreground
            button.configuration?.titleLineBreakMode = .byTruncatingTail
            button.configuration?.titleTextAttributesTransformer = UIConfigurationTextAttributesTransformer { attributes in
                var next = attributes
                next.font = Self.font
                return next
            }
            button.addAction(UIAction { [weak self] _ in self?.onChoose?(index) }, for: .touchUpInside)
            scroll.addSubview(button)
            cells.append(button)
        }
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        place()
        // Fill at least two screens so the grid has somewhere to scroll.
        while scroll.contentSize.height < 2 * scroll.bounds.height, let more = loadMore?(cells.count), !more.isEmpty {
            append(more)
            place()
        }
    }

    private func place() {
        let width = scroll.bounds.width
        guard width > 0 else { return }
        let unit = width / Self.columns
        var column: CGFloat = 0
        var row: CGFloat = 0
        for cell in cells {
            let text = cell.configuration?.title ?? ""
            let needed = (text as NSString).size(withAttributes: [.font: Self.font]).width + 24
            let span = min(Self.columns, max(1, (needed / unit).rounded(.up)))
            if column + span > Self.columns {
                column = 0
                row += 1
            }
            cell.frame = CGRect(x: column * unit, y: row * Self.rowHeight, width: span * unit, height: Self.rowHeight)
            column += span
        }
        let rows = cells.isEmpty ? 0 : Int(row) + 1
        // A hairline under each row.
        while lines.count < rows {
            let line = UIView()
            line.backgroundColor = Palette.border
            scroll.addSubview(line)
            lines.append(line)
        }
        for (index, line) in lines.enumerated() {
            line.isHidden = index >= rows
            line.frame = CGRect(x: 0, y: CGFloat(index + 1) * Self.rowHeight - 0.5, width: width, height: 0.5)
        }
        scroll.contentSize = CGSize(width: width, height: CGFloat(rows) * Self.rowHeight)
    }
}

/// Lets a drag that starts on a candidate scroll instead of choosing it.
final class CandidateScroll: UIScrollView {
    override func touchesShouldCancel(in view: UIView) -> Bool { true }
}

extension CandidateGrid: UIScrollViewDelegate {
    func scrollViewDidScroll(_ scrollView: UIScrollView) {
        guard !cells.isEmpty, scrollView.contentOffset.y + scrollView.bounds.height > scrollView.contentSize.height - 2 * Self.rowHeight,
              let more = loadMore?(cells.count), !more.isEmpty else { return }
        append(more)
        place()
    }
}

/// 9-key's left column: the spellings of the pending digits, or punctuation.
final class SyllableColumn: UIView {
    var onChoose: ((String) -> Void)?
    var onPunctuation: ((String) -> Void)?
    private let scroll = UIScrollView()
    private let column = UIStackView()

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = Palette.wellFace
        layer.cornerRadius = 8
        layer.cornerCurve = .continuous
        clipsToBounds = true
        scroll.showsVerticalScrollIndicator = false
        scroll.translatesAutoresizingMaskIntoConstraints = false
        column.axis = .vertical
        column.translatesAutoresizingMaskIntoConstraints = false
        scroll.addSubview(column)
        addSubview(scroll)
        NSLayoutConstraint.activate([
            scroll.topAnchor.constraint(equalTo: topAnchor),
            scroll.bottomAnchor.constraint(equalTo: bottomAnchor),
            scroll.leadingAnchor.constraint(equalTo: leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: trailingAnchor),
            column.topAnchor.constraint(equalTo: scroll.contentLayoutGuide.topAnchor),
            column.bottomAnchor.constraint(equalTo: scroll.contentLayoutGuide.bottomAnchor),
            column.leadingAnchor.constraint(equalTo: scroll.contentLayoutGuide.leadingAnchor),
            column.trailingAnchor.constraint(equalTo: scroll.contentLayoutGuide.trailingAnchor),
            column.widthAnchor.constraint(equalTo: scroll.frameLayoutGuide.widthAnchor),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    /// Syllables while composing; otherwise `idle` inserts punctuation.
    func show(_ syllables: [String], idle: [String]) {
        column.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let composing = !syllables.isEmpty
        for text in composing ? syllables : idle {
            let button = UIButton(configuration: .plain())
            button.configuration?.title = text
            button.configuration?.baseForegroundColor = Palette.foreground
            button.heightAnchor.constraint(equalToConstant: 40).isActive = true
            button.addAction(UIAction { [weak self] _ in
                if composing { self?.onChoose?(text) } else { self?.onPunctuation?(text) }
            }, for: .touchUpInside)
            column.addArrangedSubview(button)
        }
        scroll.contentOffset = .zero
    }
}

/// A bare icon in the top corners shared by both panels: hide (top left)
/// and the voice/typing switch (top right), so neither moves between them.
public final class KeyboardTopKey: UIButton {
    public init(icon: Icon, label: String) {
        super.init(frame: .zero)
        var configuration = UIButton.Configuration.plain()
        configuration.image = icon.image(size: TypingKeyboardView.iconSize)
        configuration.baseForegroundColor = Palette.foreground
        self.configuration = configuration
        accessibilityLabel = label
        translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            widthAnchor.constraint(equalToConstant: 44),
            heightAnchor.constraint(equalToConstant: 44),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    func setIcon(_ icon: Icon, label: String) {
        configuration?.image = icon.image(size: TypingKeyboardView.iconSize)
        accessibilityLabel = label
    }

    /// Pins the key into a panel's top-left or top-right corner.
    func pin(in panel: UIView, trailing: Bool) {
        let side = trailing
            ? trailingAnchor.constraint(equalTo: panel.trailingAnchor, constant: -KeyboardMetrics.sideInset)
            : leadingAnchor.constraint(equalTo: panel.leadingAnchor, constant: KeyboardMetrics.sideInset)
        NSLayoutConstraint.activate([topAnchor.constraint(equalTo: panel.topAnchor, constant: 4), side])
    }
}

/// The alternates bubble above a held key; the option under the finger is
/// highlighted and inserted on release.
final class AlternatesPopup: UIView {
    private let row = UIStackView()
    private var labels: [UILabel] = []
    private(set) var choice: String?

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = Palette.raisedFaceFrom
        layer.cornerRadius = 10
        layer.cornerCurve = .continuous
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOpacity = 0.2
        layer.shadowRadius = 6
        layer.shadowOffset = CGSize(width: 0, height: 2)
        row.axis = .horizontal
        row.distribution = .fillEqually
        row.translatesAutoresizingMaskIntoConstraints = false
        addSubview(row)
        NSLayoutConstraint.activate([
            row.topAnchor.constraint(equalTo: topAnchor, constant: 4),
            row.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -4),
            row.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 4),
            row.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -4),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    func show(_ options: [String], above key: UIView, in panel: UIView) {
        labels.forEach { $0.removeFromSuperview() }
        labels = options.map { text in
            let label = UILabel()
            label.text = text
            label.font = .systemFont(ofSize: 22)
            label.textAlignment = .center
            label.textColor = Palette.foreground
            label.layer.cornerRadius = 6
            label.layer.masksToBounds = true
            row.addArrangedSubview(label)
            return label
        }
        let cell: CGFloat = 36
        let width = cell * CGFloat(options.count) + 8
        let frame = key.convert(key.bounds, to: panel)
        let x = min(max(KeyboardMetrics.sideInset, frame.midX - width / 2), panel.bounds.width - KeyboardMetrics.sideInset - width)
        self.frame = CGRect(x: x, y: frame.minY - 52, width: width, height: 48)
        panel.addSubview(self)
        choice = nil
    }

    /// Highlights the option under the finger (in this view's coordinates).
    func track(_ point: CGPoint) {
        guard !labels.isEmpty else { return }
        let index = max(0, min(labels.count - 1, Int((point.x - 4) / ((bounds.width - 8) / CGFloat(labels.count)))))
        for (i, label) in labels.enumerated() {
            label.backgroundColor = i == index ? Palette.primary : .clear
        }
        choice = labels[index].text
    }

    func dismiss() {
        removeFromSuperview()
        choice = nil
    }
}
