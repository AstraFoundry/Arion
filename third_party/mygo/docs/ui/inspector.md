# Inspector

The inspector shows the elements of a window of native UI in a panel
docked beside the content, as Chrome's developer tools show a page's: the
tree of elements as markup, the styles, box model and properties of the
one chosen, highlighted over the content, how long frames take, and
issues.

Windows whose developer tools are on open it with View → Toggle Developer
Tools, `F12`, or `Alt+Cmd+I` on macOS and `Ctrl+Shift+I` elsewhere, and
`Shift+Cmd+C` (`Ctrl+Shift+C`) opens it picking an element. Developer
tools are on in development builds, as the web inspector is, and
`WindowOptions.Page.DevTools` turns them on or off for a window:

```go
mygo.NewWindow(mygo.WindowOptions{
	Content: ui.View(app.view),
	Page:    mygo.PageOptions{DevTools: mygo.DevToolsDisabled},
})
```

The content gives the panel the right of the window: the view gets a
narrower window, as `c.Size()` tells.

Production builds of `mygo build` leave the inspector out, as their
developer tools are off: it takes some 400 KB of a binary. `mygo build
-debug` keeps it, as `MYGO_INSPECTOR=1 mygo build` does in a production
build, for a window with `DevToolsEnabled` to open it there. Apps built
with `go build` keep it unless `-tags mygo_noinspector` leaves it out. Drag the panel's left edge to resize
it. The panel follows the appearance, light or dark.

The toolbar's RTL toggle lays the window out [right to left](right-to-left.md),
or back, to try an interface before it is translated; the panel itself
stays left to right. An element's computed styles show its `direction`.

## Elements

The tree shows each element as a tag: the widget it is, as `<Button>` or
`<List>`, else `<Row>`, `<Column>`, `<Text>` and the like, with its `key`
and `label` as attributes, and the text of a text inside it. Elements near
the root show what is inside them; the arrow of an element opens and
closes it, and Alt-click opens or closes everything inside.

- Hovering a row highlights its element over the content, as Chrome does:
  its content in blue, padding in green, border in yellow and margins in
  orange, with a tooltip telling its size, colors, font and spacing, and
  what assistive technology sees: its name and role, and whether the
  keyboard can focus it. Rows of elements with a
  [transition](transitions.md) say so.
- Clicking a row chooses its element; the arrows move the choice, Left
  and Right close and open the element chosen, or go to its parent and
  into it. The breadcrumbs below the tree show the path to it, and choose
  an element of the path.
- The picker of the toolbar, then a click on the content, chooses the
  element under the pointer: the click goes to the inspector, not to the
  element. Escape stops picking.
- `Cmd+F` (`Ctrl+F`) finds elements by their tag, text, label or key.

Below the tree, the element chosen has three tabs:

- **Styles**: the styles it sets, as a CSS rule, with where the app built
  it (the file and line of the call that created it, or gave it its
  `Key`), then the text styles it inherits, by the element setting them.
  MyGo's layout is CSS's, so the properties are CSS's: `flex-direction`,
  `gap`, `padding`, `flex-grow`, `width`, `color`, …
  [Change them](#changing-styles) to try a value in the window.
- **Computed**: the box model, its margins, border, padding and content,
  with their sizes, and the values the element has as laid out: its size
  and place in the window, its font, its scroll offset.
- **Properties**: what it is (its type, ID, key, text), its state
  (focused, under the pointer, pressed, disabled, moving), and what
  assistive technology sees of it.

## Changing styles

The Styles tab changes what it shows, as Chrome's does, for trying a value
without rebuilding the app:

- Click a value to type another: the window follows as you type, Enter or
  clicking away keeps it, and Escape goes back. `Up` and `Down` move its
  first number by 1, with `Shift` by 10 and with `Alt` by 0.1. Clearing a
  value turns it off.
- The checkbox before a declaration, which shows as the pointer is over it,
  turns it off and on. What an element inherits changes on the element
  setting it.
- The `+` of a rule adds declarations, typed as `opacity: 0.5; gap: 8px`.
  What MyGo does not understand shows struck through, with a warning.

Lengths are in DIPs, with or without `px`, or percentages where the element
takes them; colors are hex, `rgb()`, `oklch()` or a few names. Declarations
the tab cannot change, such as `overflow` or `transition`, have no checkbox.

The changes apply over what the view builds, whatever it builds, so they
outlast its state changing, and last until **Reset** or the inspector closes.
They change neither the code nor the view's state. **Copy as Go** puts what
you changed on the clipboard as calls to paste into the view, by element,
with where it is built when you chose it, and what to remove:

```go
// Column#box, main.go:42
// Remove border-radius: 10px
.Padding(26).Background(ui.Hex("#fde68a"))
```

## Forcing states

`:hov` at the top of the Styles tab forces states on the element chosen, to
style a hover or a press without holding the pointer there: `:hover`,
`:active`, `:focus`, `:focus-visible` and `:focus-within`. The element's
`Hovered`, `Pressed`, `Focused`, `FocusVisible` and `FocusWithin` then report
them, and the tree shows them beside the element. They only reach the element
the view asks: if a row asks whether it is hovered to color a text inside, force
the state on the row, not the text. The real focus and pointer stay where they
are, and the forced states go when the inspector closes.

## Showing layouts

Containers have a badge in the tree. Click it to keep their layout over the
content until you click it again or the inspector closes:

- **`grid`** shows the lines of the tracks, numbered as `ColumnStart` and
  `RowStart` count them, with the gaps hatched.
- **`flex`** shows the room the children leave, hatched: the gaps, the free
  space, and between wrapped lines. Every box lays out as flexbox, so only those
  with two children or more have the badge.

The layout cache is off while a badge is on, so the Performance tab's layout
times aren't those of the app.

## Opening the source

The link beside the element in the Styles tab, such as `main.go:42`, opens that
line in your editor. Set the editor in `MYGO_EDITOR`, or MyGo uses `VISUAL`,
then `EDITOR`:

```sh
MYGO_EDITOR=code mygo dev
```

VS Code and its forks (`code`, `cursor`, `windsurf`), Zed, Sublime Text,
JetBrains' IDEs (`goland`), TextMate, GVim and Emacs open at the line. Other
editors get the file alone, unless the command says where things go with
`{file}` and `{line}`:

```sh
MYGO_EDITOR='tmux new-window nvim +{line} {file}'
```

An app has no terminal to run an editor in, so `VISUAL` and `EDITOR` are skipped
when they name one, such as `vim` or `nano`. A command like the one above opens
one. Quote a path with spaces. With no editor set, the file opens at its top in
the system's default application. What goes wrong is listed under Issues.

The variables must reach the app, as they do from a shell running `mygo dev`, but
not when it starts from the Dock or a file manager. A build with `-trimpath`, as
`mygo build --debug` is, has no paths to open, so its location isn't a link.

## Performance and issues

**Performance** shows how long the last frames took to build, lay out and
paint, as bars against the time between two refreshes of the window's
display (8.3 ms at 120 Hz, 16.7 ms at 60 Hz), with the last, the average
and the slowest. Set `MYGO_FRAME_STATS` to log slow frames as well
(see [Rendering](rendering.md)).

**Issues** lists mistakes found while building, which the toolbar counts:
two elements given the same `Key` under one parent, say, share one state,
so that a click, the focus or scrolling meant for one goes to the other.
Apps log each one once; a [Tester](testing.md) panics where the second
key was given, so that the test fails there.
