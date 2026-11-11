# Code editor

<!-- preview code-editor 320 -->

Package `ui/editor` is a code editor and a [diff viewer](diff-viewer.md)
for native UI, designed after Zed's: syntax highlighting, several carets
and selections, soft wraps, folding, finding and replacing, completions,
hovers, diagnostics and inlay hints from the app or a language server,
and the commands and keys of code editors. It lays out and paints only the
lines in view, so a file of a million lines scrolls and edits as one of
ten.

```go
import "github.com/egoist/mygo/ui/editor"

// Kept by the app, as its other state.
buf := editor.NewBuffer(source)
buf.SetLanguage(editor.LanguageFor("main.go"))
app.code = &editor.State{Buffer: buf, WrapGuides: []int{100}}

// In the view:
editor.View(c, app.code).Grow(1).FillWidth()
```

A `State` is the editor's state, kept across frames as a text input's
string is: its buffer, its options, and what the editor keeps of its own,
as its selections, folds and scroll position. Its options are fields set
before the editor is built; their zero values are the defaults, those of
Zed. `View` builds the editor of a state, which is a focusable element as
any other: give it a size, and `AutoFocus` to type in it at once.

The [code editor example](../../examples/code-editor) edits files of
several languages, with completions of the words of the file, hovers,
diagnostics of TODO comments, the changes in the gutter and their diff.

## Buffers

A `Buffer` is the text an editor edits, with its undo history and its
language. Several editors, and both sides of a diff, can show one buffer:
an edit in one shows in the others, their carets moving with the text
around them, and undo takes back the buffer's last edit, whichever editor
made it.

```go
buf.Edit(editor.Edit{Start: 0, End: 0, Text: "// Generated.\n"})
buf.Transact(func() { // one step of undo
	buf.Edit(editor.Edit{Start: a, End: b, Text: name})
	buf.Edit(editor.Edit{Start: c, End: d, Text: name})
})
if buf.Modified() {
	_, err := buf.WriteTo(file)
	if err == nil {
		buf.MarkSaved()
	}
}
```

Offsets count bytes of UTF-8, and a `Pos` is a line and the bytes into it,
both from 0: `Pos` and `Offset` convert between them, and `Line`, `Slice`
and `String` read the text. `Set` replaces the text as an edit, which undo
takes back, and `Reset` as a new document, without history.

Lines end in `\n`, as in Zed: a text with `\r\n` line endings is kept with
`\n` ones, and `LineEnding` says which it had, which `WriteTo` writes, so
that a file saves as it was; `SetLineEnding` changes it.

The text is a persistent tree of chunks, as Zed's rope: an edit copies a
few kilobytes and a path of the tree, whatever the size of the document,
and each version shares the rest with the one before. Undo keeps versions
rather than copies, and a `Snapshot` is a version that stays as it is
while the buffer changes, which other goroutines read without locks.
Buffers' methods are safe from any goroutine; change them with
`Window.Update` so the window shows the change.

## Editing

The editor edits as Zed does, with the commands bound to the same keys
(Control replaces Command on Linux and Windows, as in Visual Studio Code):

- **Several carets.** Cmd+D selects the next occurrence of the selection,
  or the word at the caret; Cmd+Shift+L all of them; Cmd+Alt+Up and Down
  add a caret on the line above or below; Alt+click adds a caret, and
  Alt+Shift+drag selects a column. Typing, deleting and every command act
  at each caret, and Escape goes back to one. Cmd+U undoes a change of the
  selections.
- **Selections by structure.** Ctrl+Shift+Right selects the word, then
  the string or brackets around it, and on out; Ctrl+Shift+Left goes back.
- **Lines.** Alt+Up and Down move lines, Alt+Shift+Up and Down duplicate
  them, Cmd+Shift+K deletes them, Ctrl+J joins them, Cmd+/ comments them
  out, Cmd+] and Cmd+[ indent them; Cmd+Enter opens a line below.
- **Typing.** Brackets and quotes close as they are typed, and wrap the
  selection; Enter keeps the indentation, indents after an opening bracket
  and between a pair, and continues line comments; a closing bracket, or
  a word such as `end`, outdents its line as it is typed. Undo takes back
  typing a burst at a time.
- **Folding.** The gutter's markers fold the lines indented under a line,
  up to its closing bracket, or a run of imports, with the blank lines
  between them; Cmd+Alt+[ and ] fold and unfold, Cmd+K
  Cmd+0 folds everything and Cmd+K Cmd+J unfolds it, Cmd+K Cmd+1 to 9 by
  level.
- **Finding.** Cmd+F opens the find bar, with case, whole words, regular
  expressions and finding in the selection; Cmd+Alt+F replaces. Cmd+G goes
  to the next match, and Ctrl+G to a line.
- **The pointer.** A double click selects a word, a triple click a line,
  and dragging extends by them; dragging near the edges scrolls; Cmd+click
  goes to a definition.

`Run` runs a command, as a menu item or the app's own key does, and
`Bindings` binds keys: Mods and Key, then ThenMods and ThenKey for chords
such as Cmd+K Cmd+0. An app's bindings come before the defaults
(`DefaultBindings`), and a binding to `NoCommand` takes a key away from
the editor, for the app's own shortcut. `CommandNamed` finds a command by
the name keymaps give it, as "selectNext".

```go
app.code.Bindings = []editor.Binding{
	{Mods: ui.Cmd, Key: ui.KeyP, Command: editor.NoCommand}, // the app's
	{Mods: ui.Cmd | ui.Shift, Key: ui.KeyD, Command: editor.DuplicateSelection},
}
if ui.Button(c, "Format").Clicked() {
	app.code.Run(editor.SortLines)
}
```

`Selections`, `SetSelections`, `Select`, `Cursor` and `CursorPos` read
and set the carets and selections, `Insert` types at each of them, `GoTo`
moves the caret to a position and `ScrollTo` shows a line. `Find` opens
the find bar on a query and `FindMatches` counts its matches; `Fold`,
`Unfold` and `Folded` fold lines, and `FoldImports` the first run of
imports, as an editor can on opening a file; `Focused` reports whether the editor has
the keyboard focus, and `VisibleLines` the first and last lines the last
frame showed, as a half page of Vim's moves by. `ScrollY` and
`SetScrollY` read and set how far the lines are scrolled, in DIPs,
`ViewportHeight` how high the part of them in view is, and `LineTop`
where a line starts, below the first, `LineTop` of the count of lines
being the height of them all: editors side by side scroll together with
them, an app draws what joins their lines, as the bands between the
panes of a merge, and a minimap beside the editor shows where its view
is. The scroll bars' thumbs show while the pointer is over the editor, or
drags one, as a scroll container's do, or all the time with
`KeepScrollbars`; the marks of changes, matches and diagnostics on the
vertical one show all the time. `NoScrollbars` takes the bars and their
marks away, and gives their room to the text.

`OnInput` sees each input event before the editor does: keys, the text
typed, edit commands and the pointer. Returning true takes the event, and
the editor leaves it; the others go on as without it. An app's own mode of
editing, as Vim's normal mode, takes keys this way while the editor still
shows and moves the carets. Letters typed at the carets come as
`InputText`: on some systems their keys come first as `InputKeyDown` too,
and taking a key doesn't take its text, so a mode reads letters from
`InputText`. What an input method composes, commits or replaces, as the
accents of macOS's press and hold replace the letter before the caret,
goes to the editor alone. The key after the first of a chord of
`Bindings`, as Cmd+K Cmd+0, is the chord's, as VS Code resolves chords
before an extension's keys.

`NoInputMethod` turns the system's input methods off while the editor has
the keyboard, as Neovide does in Vim's normal mode: keys type the
characters of the keyboard's layout, which come to `OnInput` as
`InputText`, and nothing is composed, so j is j with a Chinese input
method on. Set it in the modes that read letters as commands and clear it
in insert mode. Browsers cannot turn input methods off: there they compose
in the text still.

```go
app.code.CursorShape = editor.CursorBlock // normal mode
app.code.NoInputMethod = true
app.code.OnInput = func(ev ui.InputEvent) bool {
	if ev.Kind == ui.InputText {
		return app.vim.normalKeys(ev.Text) // j, k, dd, …
	}
	return false
}
```

## Options

- `SoftWrap` wraps lines at the editor's width, or at `WrapColumn` columns
  when narrower; wrapped rows keep the line's indentation.
- `LineNumbers` shows the gutter's numbers, absolute, relative to the
  caret's line, or not at all; `GutterMarks` put dots of a color by lines,
  as breakpoints, and `OnGutterClick` reports clicks on numbers.
- `LineBackgrounds` fill lines, from `First` to `Last`, with a color under
  the text and the selections, as the statement a key would run; the
  gutter keeps its own, and lines a fold hides take no room to fill. They
  keep their line numbers as the text is edited, unlike diagnostics and
  inlays, which move with it: compute them again when the buffer's
  `Version` changes.
- `TextBackgrounds` fill ranges of the text, bytes `Start` to `End`, with
  a color under the text and the selections and over line backgrounds, as
  the words a merge changed within its lines. They follow the editor's
  layout: the rows a line wraps into, tabs, wide characters and scrolling
  sideways; lines a fold hides show none. As line backgrounds, they stay
  where they are as the text is edited, cut to its end until computed
  again when the buffer's `Version` changes.
- `TabSize` and `Indent` set indentation: tabs or spaces, as the
  language's files usually have them when zero (Go's tabs).
- `Whitespace` shows spaces and tabs as dots and arrows: within selections
  by default, everywhere, at the ends of lines, or nowhere.
- `WrapGuides` draws guides at columns, as 80 and 100; `NoIndentGuides`
  hides those of indentation.
- `ReadOnly` lets the text be selected and copied, not edited.
- `FitContent` makes the editor as high as its lines, rather than
  scrolling them, to show code in a scroll view or a list.
- `KeepScrollbars` keeps the scroll bars' thumbs showing with the pointer
  away, as a scroll container's `KeepScrollbars`; `NoScrollbars` takes
  the bars away, with their room, the wheel and the keys still
  scrolling.
- `Font` and `LineHeight` set the text: the system's monospaced font of
  13 DIPs, 1.5 times as high, by default.
- `NoAutoClose`, `NoFolding`, `NoActiveLine` and `NoScrollPastEnd` turn
  those behaviors off. `Placeholder` shows while the buffer is empty.
- `CursorShape` paints carets as bars (`CursorBar`), as blocks over the
  character after them (`CursorBlock`), as Vim's normal mode does,
  translucent so that the character shows and outlined while the editor
  hasn't the keyboard, or as underlines (`CursorUnderline`), as its
  pending operators do.
- `DiffBase` is the text the buffer's changes show against in the gutter,
  as the file in Git: bars mark the lines added and changed, and a mark
  where lines were deleted. Cmd+F8 goes to the next change, and Cmd+Alt+Z
  reverts the one at the caret.
- `ContextMenu` adds items to the editor's context menu, after its own.

## Languages and highlighting

`LanguageFor` finds a file's language by its name, and `LanguageNamed` by
its own; `Languages` lists them. The editor knows Go, JavaScript,
TypeScript, JSON, Python, Rust, C, C++, Java, Kotlin, Scala, Swift, Dart,
C#, shell scripts, YAML, TOML, SQL, R, Ruby, Lua, PHP, CSS, HTML, XML,
Markdown, diffs, Dockerfiles, Makefiles, Protocol Buffers, GraphQL and Zig.
A `Language` says how to highlight, comment, match brackets and indent its
text, and with `Imports`, a pattern of a line that imports (`^import\b` for
Go), which lines fold as a run: build one for another language, or copy
one and change it.

A language highlights its text with a `Tokenizer` by default: a function
from a line, and the state the line before it ended in, to the line's
tokens and the state it ends in, as TextMate grammars and Monaco's
tokenizers work. The editor tokenizes the lines it shows, and after an
edit only the lines below it until one starts in the state it started in
before, a frame's time budget at a time.

```go
// Lines starting with # are comments.
lang := &editor.Language{
	Name:        "Notes",
	Extensions:  []string{".notes"},
	LineComment: "#",
	Tokenizer: editor.TokenizerFunc(func(line string, state uint64, tokens []editor.Token) ([]editor.Token, uint64) {
		kind := editor.TokenText
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			kind = editor.TokenComment
		}
		return append(tokens, editor.Token{End: len(line), Kind: kind}), 0
	}),
}
```

A token is a run of a line of one `TokenKind`, which the theme colors:
keywords, types, functions, strings, comments and the others.

### Highlighters

A language can instead bring a highlighter of its own, as one of a parser
such as tree-sitter: `Language.Highlighter` makes the `Highlighter` of each
buffer of the language, and the editor uses it in place of the tokenizer.
The tokenizer itself is one (`NewTokenHighlighter`), so a highlighter
plugs in without the editor changing. The text of a patch's side
(`DiffState.SetPatch`) is its hunks' lines one after another: the
tokenizer starts each hunk in state 0, while a highlighter of its own
sees one text, and highlights it as it sees fit.

```go
type Highlighter interface {
	// Edit follows an edit of the text, old and new its text before and
	// after, and changes what it replaced, in order.
	Edit(old, new editor.Snapshot, changes []editor.Change)
	// Highlight works toward the tokens of the lines up to last, for at
	// most budget, and reports whether they are ready.
	Highlight(last int, budget time.Duration) bool
	// Tokens appends the tokens of line i to dst, and reports whether
	// they are final rather than a guess made before Highlight is done.
	Tokens(i int, text string, dst []editor.Token) ([]editor.Token, bool)
}
```

The editor calls these methods on the main thread: `Edit` as the buffer
changes, `Highlight` before it paints, with the lines it shows, and again
in later frames while it reports false, and `Tokens` for each line it
paints. A `Change` gives the ranges an edit replaced in the old text and
the new, what an incremental parser takes to edit its tree. A parser can
work in the background instead: snapshots are safe to read from any
goroutine, `Chunks` reads them a piece at a time, and the `invalidate`
function the highlighter was made with draws the editors showing the
buffer again once a parse is done.

A highlighter that knows the structure of the text can tell the editor
more, by implementing any of these interfaces too:

- `SyntaxSelector`: `Enclosing` returns the smallest node holding more
  than a range, which Select Larger selects, rather than the word, string
  or brackets around it.
- `Folder`: `FoldEnd` returns the last line folding a line hides, rather
  than what indentation and brackets say.
- `BracketMatcher`: `EnclosingBrackets` returns the innermost pair around
  a range, which the editor highlights and moves between, as `do` and
  `end`.
- `Indenter`: `IndentAfter` returns how many levels more a line Enter
  opens is indented.

## Completions, hovers and diagnostics

The editor shows what the app or a language server knows of the code; the
app answers its requests, at once or later from another goroutine through
`Window.Update`:

```go
app.code.OnCompletion = func(r editor.CompletionRequest) {
	go func() {
		items := app.lsp.Complete(r.Offset) // a language server's
		app.win.Update(func() { app.code.ShowCompletions(r.Offset-len(r.Prefix), items) })
	}()
}
app.code.OnHover = func(r editor.HoverRequest) {
	if doc, start, end, ok := app.lsp.Hover(r.Offset); ok {
		app.code.ShowHover(start, end, doc, nil)
	}
}
app.code.SetDiagnostics(buf.Version(), app.lsp.Diagnostics())
```

- `OnCompletion` asks for completions as a word is typed, or Ctrl+Space
  pressed. `ShowCompletions` lists them below the caret, filtered and
  ranked as typing goes on by the word typed, with their kinds, details
  and documentation; Enter or Tab inserts the one chosen.
- `OnHover` asks what to show as the pointer rests on the text
  (`HoverDelay`, 600 ms by default), or for Cmd+K Cmd+I at the caret.
  `NoPointerHover` turns hovers from the resting pointer off, diagnostics'
  too, as VS Code's `editor.hover.enabled`; the keys still show one.
  `ShowHover` shows text in a popover over the range, or what a function
  builds there. The pointer can go into the
  hover, which stays while it is there, to select and copy its text; the
  hover goes a moment after the pointer leaves the text and the hover,
  and Escape closes it.
- `SetDiagnostics` underlines problems with wavy lines of their severity,
  shows their messages in the hover and marks them on the scroll bar. The
  version is the buffer's the diagnostics are of: the editor moves them
  through the edits made since.
- `SetInlays` shows inlay hints, text in the lines that is not theirs, as
  the types of variables; `SetSuggestion` shows text after the caret as a
  suggestion, which Tab accepts, as an AI's.
- `OnDefinition` is called for Cmd+click and F12 on a word.

## Code snippets

A completion's `Insert` is always plain text, even with
`Kind: editor.CompletionSnippet` (which chooses an icon). To insert a
code template with editable placeholders, parse one and set `Snippet`:

```go
snippet, err := editor.ParseSnippet("for ${1:index}, ${2:value} := range ${3:items} {\n\t$0\n}")
if err != nil {
    // Offer a plain-text completion instead when a server uses
    // snippet syntax the editor does not support.
    return err
}
code.ShowCompletions(start, []editor.Completion{{
    Label: "forrange", Kind: editor.CompletionSnippet, Snippet: snippet,
}})
// Or insert at every current selection, outside the completion list:
code.InsertSnippet(snippet)
```

Parse templates once; a `Snippet` is immutable and can be shared by
editors. `Text()` returns its initial text. The parser supports `$1`,
`${1}`, `${1:default}`, escaped dollars, braces and backslashes, and `$0`
as the final stop (the end when omitted). Tab visits positive numbers in
numeric order, with 0 last; Shift+Tab goes back, stopping at the first
field. Repeated numbers share their default, including forward references,
and edit together through the editor's normal multiple selections.
Coincident empty stops of the same number coalesce as carets do.
Scrolling and input methods follow the first occurrence at the original
primary insertion site.

A completion list takes Tab before the snippet's navigation, so a field
can itself be completed. Escape ends the session after closing any open
completion list or other editor overlay. Moving outside the active
fields, editing the buffer externally, undo/redo or changing buffers
ends the session too. Each field's typing is a separate undo group;
undoing the insertion restores the old text and selections. Redo restores
the text and selections without restarting snippet navigation. Input
methods compose and commit at mirrored fields as at ordinary carets.
A new snippet replaces the previous session.

This is the tab-stop subset of TextMate/LSP syntax: nested placeholders,
variables, choices, transformations, conflicting defaults for one number,
and invalid UTF-8 return errors. Text keeps the template's indentation;
CRLF is normalized to LF. `InsertSnippet` leaves a read-only editor
unchanged. State methods, including snippet insertion, run on the UI
thread as the rest of editor interaction does.

Run `go run ./examples/code-editor`, type `funcdef` or `forrange` in a Go
file, then press Tab to try it.

## Styling

### Themes

`Theme` colors the editor: its background, text, gutter, selections,
carets, guides, diagnostics, changes and each kind of token. Colors left
zero take those of `DefaultTheme`, which follows the window's theme and
appearance, light or dark.

```go
th := editor.DefaultTheme(c.Theme())
th.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: ui.RGB(0xc6, 0x78, 0xdd), Bold: true}
app.code.Theme = th
```

### Bars and popups

The find bar, the go to line bar, the completions and the hovers are
parts the app can build itself, as it styles ui's
[bases](custom-widgets.md): `RenderFindBar`, `RenderGoToLine`,
`RenderCompletions` and `RenderHover` are each given the element the
editor places (a toolbar over its top right corner, a bar at its top, a
list below the caret, a card above the text hovered), which keeps the
presses from the text under it and the keyboard where the part needs it
(in the editor while the completions show, in the hover as its text is
selected), and a handle to what the part shows and does. Style the element and build its
content; its position can change too. The editor's own looks are
`DefaultFindBar`, `DefaultGoToLine`, `DefaultCompletions` and
`DefaultHover`, which a render function can call and then restyle.

```go
app.code.RenderFindBar = func(c *ui.Context, bar ui.Element, f *editor.FindBar) {
	t := c.Theme()
	bar.Row().Gap(8).Padding(6).Background(t.Surface).Radius(12).Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, 0.2))
	bar.Children(func() {
		f.QueryInput(ui.TextInputBase(c, f.Query()).Placeholder("Search").Width(180))
		ui.Toggle(c, f.Regexp(), ".*").KeepFocus()
		ui.Text(c, f.Status()).TextColor(t.TextMuted)
		if ui.Button(c, "Next").KeepFocus().Clicked() {
			f.Next()
		}
	})
}
app.code.RenderHover = func(c *ui.Context, panel ui.Element, h *editor.HoverCard) {
	editor.DefaultHover(c, panel, h) // the editor's look, on another background
	panel.Background(app.hoverColor)
}
```

- `FindBar` binds fields and toggles: `Query` and `Replacement`, and
  `CaseSensitive`, `WholeWord`, `Regexp` and `InSelection`, are pointers,
  and the matches follow them as they change. `QueryInput` and
  `ReplacementInput` give inputs the bar's keys: the query takes the
  focus as the bar opens, Enter and Shift+Enter go through the matches,
  Alt+Enter selects them all, Enter and Cmd+Enter in the replacement
  replace one or all, and Escape closes. `Matches` and `Status` count the
  matches, and `Next`, `Previous`, `SelectAll`, `Replace`, `ReplaceAll`
  and `Close` act; `Replacing` says whether the bar replaces too, as
  Cmd+Alt+F opens it.
- `GoToLineBar` has `Text` for its field, which `Input` gives the bar's
  keys, `Cursor`, `LineCount` and `Target` to tell where it goes, and
  `Go` and `Close`.
- `CompletionMenu` has the `Items` shown, best first, and the one
  `Selected`. `Item(i)` makes the row of an item, selected as the arrows
  move, scrolled into view and inserting the item when clicked; style it
  and give it content. `Font`, `LineHeight` and `Theme` show code as the
  editor does, and a `CompletionKind` has an `Icon` and the `Token` whose
  color tints it.
- `HoverCard` has the `Diagnostics` of the text hovered, the `Text` the
  app gave `ShowHover`, and `Content`, which builds what its function
  builds.

## Accessibility

Assistive technology sees a multi-line text field, named by `Label`, with
the text of the buffer and the newest selection, which it moves; the
completions are a list, and the find bar a toolbar that tells the count
of matches. The editor's menu, its find and go to line bars and their
names are in the window's language
([MyGo's own strings](../i18n.md#mygos-own-strings)). Input methods compose in the editor as in other text fields,
with their candidate window at the caret.
