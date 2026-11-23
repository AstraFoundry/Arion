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
}

/// Pinyin, 9-key and English typing on the Rime engine (ADR 053). A
/// composition stays in the keyboard; only committed text reaches the editor.
/// The bottom row starts with the switch key, where the voice panel has its
/// own, so moving between voice and typing never moves it.
public final class TypingKeyboardView: UIView {
    public weak var delegate: TypingKeyboardViewDelegate?

    private let engine: TypingEngine
    private let candidateBar = CandidateBar()
    private let keys = UIStackView()
    private let syllables = SyllableColumn()
    private var shifted = false
    private var symbolsShown = false
    private var deleteTimer: Timer?
    private var returnKeys: [FaceButton] = []
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
    public var returnTitle: String? { didSet { rebuild() } }
    private let dismiss = KeyboardDismissKey()

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
        symbolsShown = false
        shifted = false
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
        globe.accessibilityLabel = L10n.keyboardNextKeyboard
        globe.sideInset = 0
        globe.minimumFaceHeight = 0
        globe.cornerRadius = 8
        globe.iconSize = Self.iconSize
        candidateBar.onChoose = { [weak self] index in self?.choose(index) }
        candidateBar.loadMore = { [weak self] offset in self?.engine.candidates(from: offset, limit: TypingEngine.barCandidates) ?? [] }
        syllables.onChoose = { [weak self] syllable in
            self?.engine.chooseSyllable(syllable)
            self?.render()
        }
        syllables.onPunctuation = { [weak self] text in self?.delegate?.typingInsert(text) }
        keys.axis = .vertical
        keys.spacing = 10
        for view in [candidateBar, dismiss, keys] as [UIView] {
            view.translatesAutoresizingMaskIntoConstraints = false
            addSubview(view)
        }
        let height = heightAnchor.constraint(equalToConstant: KeyboardMetrics.height)
        height.priority = .defaultHigh
        let inset = KeyboardMetrics.sideInset
        NSLayoutConstraint.activate([
            height,
            candidateBar.topAnchor.constraint(equalTo: topAnchor, constant: 4),
            candidateBar.leadingAnchor.constraint(equalTo: leadingAnchor, constant: inset),
            candidateBar.trailingAnchor.constraint(equalTo: dismiss.leadingAnchor, constant: -4),
            candidateBar.heightAnchor.constraint(equalToConstant: 44),
            dismiss.centerYAnchor.constraint(equalTo: candidateBar.centerYAnchor),
            dismiss.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -inset),
            keys.topAnchor.constraint(equalTo: candidateBar.bottomAnchor, constant: 6),
            keys.leadingAnchor.constraint(equalTo: leadingAnchor, constant: inset),
            keys.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -inset),
            keys.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -KeyboardMetrics.bottomInset),
        ])
    }

    // MARK: Keys

    private func rebuild() {
        keys.arrangedSubviews.forEach { $0.removeFromSuperview() }
        returnKeys = []
        let chinese = engine.layout != .english
        if symbolsShown {
            for row in KeyLayout.symbols(chinese: chinese) { keys.addArrangedSubview(rowView(row)) }
        } else if engine.layout == .pinyin9 {
            keys.addArrangedSubview(keypad())
        } else {
            for row in KeyLayout.qwerty(engine.layout) { keys.addArrangedSubview(rowView(row)) }
        }
        let keypadPage = engine.layout == .pinyin9 && !symbolsShown
        keys.addArrangedSubview(rowView(KeyLayout.bottom(symbols: symbolsShown, globe: showsGlobe, keypad: keypadPage)))
        // Every row is one key high; the keypad block spans three.
        for view in keys.arrangedSubviews {
            let rows: CGFloat = view === keys.arrangedSubviews.first && keypadPage ? 3 : 1
            view.heightAnchor.constraint(equalToConstant: rows * KeyboardMetrics.rowHeight + (rows - 1) * keys.spacing).isActive = true
        }
        render()
    }

    /// 9-key: syllables or punctuation on the left, the keypad, then delete
    /// over a tall return, as on the system's 9-key keyboard.
    private func keypad() -> UIView {
        let grid = UIStackView(arrangedSubviews: KeyLayout.keypad.map { rowView($0) })
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

    private func rowView(_ row: [KeyLayout.Key]) -> UIView {
        let stack = UIStackView()
        stack.axis = .horizontal
        stack.spacing = 6
        var first: (view: UIView, width: CGFloat)?
        for key in row {
            let view: UIView = key.key == .globe ? globe : keyView(key)
            stack.addArrangedSubview(view)
            if let fixed = key.fixed {
                view.widthAnchor.constraint(equalToConstant: fixed).isActive = true
            } else if let first {
                view.widthAnchor.constraint(equalTo: first.view.widthAnchor, multiplier: key.width / first.width).isActive = true
            } else {
                first = (view, key.width)
            }
        }
        // A shorter row (the second QWERTY row) centers with the first's key width.
        if row.count == 9, row.allSatisfy({ if case .letter = $0.key { true } else { false } }) {
            let holder = UIView()
            stack.translatesAutoresizingMaskIntoConstraints = false
            holder.addSubview(stack)
            NSLayoutConstraint.activate([
                stack.topAnchor.constraint(equalTo: holder.topAnchor),
                stack.bottomAnchor.constraint(equalTo: holder.bottomAnchor),
                stack.centerXAnchor.constraint(equalTo: holder.centerXAnchor),
                stack.widthAnchor.constraint(equalTo: holder.widthAnchor, multiplier: 0.9),
            ])
            return holder
        }
        return stack
    }

    private func keyView(_ spec: KeyLayout.Key) -> FaceButton {
        if spec.key == .globe { return globe }
        let button = FaceButton(kind: spec.key == .return ? .primary : .secondary, material: .raised)
        button.sideInset = 0
        button.minimumFaceHeight = 0
        button.cornerRadius = 8
        button.iconSize = Self.iconSize
        configure(button, spec)
        button.titleLabel.font = Self.font(for: spec.key, title: button.title)
        if spec.key == .return { returnKeys.append(button) }
        button.addAction(UIAction { [weak self] _ in self?.press(spec.key) }, for: .touchUpInside)
        if spec.key == .delete {
            button.addTarget(self, action: #selector(deleteDown), for: .touchDown)
            button.addTarget(self, action: #selector(deleteUp), for: [.touchUpInside, .touchUpOutside, .touchCancel])
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
            button.icon = shifted ? .arrowBigUpDash : .arrowBigUp
            button.accessibilityLabel = L10n.typingShift
        case .delete:
            button.icon = .delete
            button.accessibilityLabel = L10n.keyboardDelete
        case .voice:
            button.icon = .mic
            button.accessibilityLabel = L10n.typingVoice
        case .globe: break
        case .space: button.title = L10n.keyboardSpace
        case .return: showReturn(on: button)
        case .symbols: button.title = L10n.typingSymbols
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

    /// While Pinyin composes, return inserts the letters as typed (确认);
    /// otherwise it shows the field's word, or the new-line icon.
    private func showReturn(on button: FaceButton) {
        let title = engine.state.isComposing ? L10n.typingConfirm : returnTitle
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
            if shifted { shifted = false; rebuild() }
        case .text(let text):
            commitComposition()
            delegate?.typingInsert(text)
        case .separator:
            if engine.state.isComposing { engine.type("'") } else if engine.layout == .pinyin9 { delegate?.typingInsert("，") } else { delegate?.typingInsert("'") }
        case .shift:
            shifted.toggle()
            rebuild()
            return
        case .delete, .globe:
            break
        case .voice:
            commitComposition()
            delegate?.typingShowVoice()
            return
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
            symbolsShown = true
            rebuild()
            return
        case .letters:
            symbolsShown = false
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
        if let text = engine.choose(index), !text.isEmpty { delegate?.typingInsert(text) }
        render()
    }

    private func deleteOnce() {
        if engine.backspace() {
            render()
        } else {
            delegate?.typingDeleteBackward()
        }
    }

    @objc private func deleteDown() {
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

    @objc private func deleteUp() {
        deleteTimer?.invalidate()
        deleteTimer = nil
    }

    private func render() {
        let state = engine.state
        candidateBar.show(preedit: state.preedit, candidates: state.candidates)
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

/// Hides the keyboard: a bare chevron at the top right of both panels.
public final class KeyboardDismissKey: UIButton {
    public override init(frame: CGRect) {
        super.init(frame: frame)
        var configuration = UIButton.Configuration.plain()
        configuration.image = Icon.chevronDown.image(size: TypingKeyboardView.iconSize)
        configuration.baseForegroundColor = Palette.foreground
        self.configuration = configuration
        accessibilityLabel = L10n.keyboardDismiss
        translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            widthAnchor.constraint(equalToConstant: 44),
            heightAnchor.constraint(equalToConstant: 44),
        ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
}
