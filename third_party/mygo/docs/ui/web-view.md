# Web view

`ui.WebView` shows a web page in a window of native UI: a document the web
engine renders beside native controls, a page per tab, a preview of HTML
the app makes. The page is the system's web view (WKWebView, WebView2,
WebKitGTK), made with `Window.NewWebView`, and works as a window's page
does: it loads the app's frontend, calls bound Go methods with the typed
client and hears events. What the view paints after the web view shows
over the page: a menu opening from a button above it, a select's options,
a dialog and its dimmed backdrop, tooltips, toasts, a badge over a corner.

```go
type browser struct {
	win  *mygo.Window
	docs *mygo.WebView
	menu bool
}

func (app *browser) view(c *ui.Context) {
	ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Row(c).Padding(8).Gap(8).Children(func() {
			menu := ui.Button(c, "Menu ▾")
			if menu.Clicked() {
				app.menu = !app.menu
			}
			// The popover shows over the page.
			ui.Popover(c, menu, &app.menu, func() { ui.Text(c, "Over the page") })
		})
		ui.WebView(c, app.docs).Grow(1).Radius(8)
	})
}

func main() {
	app := &browser{}
	mygo.App.WhenReady(func() {
		app.win = mygo.NewWindow(mygo.WindowOptions{Title: "Docs", Content: ui.View(app.view)})
		app.docs, _ = app.win.NewWebView(mygo.WebViewOptions{URL: "/docs"})
		app.win.Invalidate()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Making web views

`Window.NewWebView` makes a web view of a window showing `Content`, which
it closes with. Its options are the page's (`Page`, as a window's
`PageOptions`), the `URL` to load, a `BackgroundColor` to fill it with
until the page paints, `Transparent`, which shows what is behind the
web view through the transparent parts of its page: the window's
background or material, as native UI paints nothing under a web view,
and `ScrollToContainer`, which gives the wheel over the page to the
native UI around it ([below](#scrolling-the-native-ui-around-it)).
`WebView.Page` returns its page, whose methods are a window's page's:
`LoadURL`, `LoadHTML`, `Eval`, `GoBack`, `OnDidNavigate`, `OnWillNavigate`
and the others. `Focus` gives it the keyboard, `SetBackgroundColor`
changes the color it fills with, as when the app's own theme changes
(one given as `light-dark(<light>, <dark>)` follows the appearance by
itself), `CapturePage` takes a screenshot of its page, and `Destroy`
closes it.

A web view trusts the app's own pages as a window does, pages of custom
schemes included: show third-party or user-supplied HTML in an
[untrusted page](../frontend.md#untrusted-pages) (`PageOptions.Untrusted`),
or from an origin it does not trust, see [trust](../frontend.md#trust).

Make a web view once, as a window, not in the view: the view only shows
it. One the view does not build hides and keeps its page, so a view of tabs
builds the web view of the tab shown, and switching back finds the page
where it was.

```go
if app.tab == "docs" {
	ui.WebView(c, app.docs).Grow(1)
} else {
	ui.WebView(c, app.changelog).Grow(1)
}
```

## Its box

A web view has no size of its own. The element stretches across its
container (`AlignSelf(Stretch)`) and takes the room along it with `Grow`,
or a size. The page shows in the element's content box: inside its border
and padding, which paint around it with the element's background, rounded
as its `Radius` makes the border's inner edge. The page itself is cut to
those corners, so none of it shows outside them, even with nothing painted
under the web view, as in a transparent window, and the pointer goes to
the page only inside them. Scroll containers and clips around it cut it
as they cut any element, and a web view scrolled out of view hides.
`Opacity` fades the page with the element, as a page of a `Router` fading
in does: the page blends over what is painted under the element. With
nothing painted there, as in a transparent window, the page does not fade:
the system's web view has no opacity of its own that MyGo sets.

```go
ui.WebView(c, app.preview).Height(320).Radius(12).Border(1, t.Border)
```

### Scaled with the native UI around it

A transform scaling the web view, its own or one around it, shows the page
scaled as a picture of it: the page keeps the size it lays out at, the
element's content box before the transform, so its text does not wrap
again and `innerWidth` stays as it was while a panel zooms in or out. The
pointer reaches the page where it shows, at the page's own coordinates,
and inside its rounded corners alone. A peek panel opening over the
window, its web view and native controls scaling as one surface over a
dimmed backdrop:

```go
func (app *browser) peek(c *ui.Context) {
	scale := c.Root().AnimateTo("peek", app.peekScale(), ui.Animation{Duration: 250 * time.Millisecond})
	ui.Box(c).Absolute().Fill().Background(ui.RGBA(0, 0, 0, .4)) // the backdrop, unscaled
	ui.Column(c).Absolute().Left(80).Top(60).Right(80).Bottom(60).Scale(scale, scale).
		AlignItems(ui.Stretch).Radius(16).Background(t.Surface).Children(func() {
		toolbar(c)
		ui.WebView(c, app.peekPage).Grow(1).Radius(16)
	})
}
```

A transform that turns, skews or sees the web view in perspective shows
the page upright, laid out across the transformed box's bounds.

- **macOS** and **Windows** show the page scaled by the compositor: as a
  picture, sharp again once the animation ends at full size.
- **Linux**: GTK 3 cannot scale a widget. The web view is as large as the
  page shows, and WebKitGTK zooms the page to fit, which lays it out again
  at the same width in CSS pixels, sharp at every size: `innerWidth`
  stays the same, but the page hears a `resize` event as its zoom
  changes, and it scales by its width alone. Its corners are cut on whole
  pixels, without antialiasing.

## What shows over it

The web view is under the window's native UI: the element cuts a hole
through what the window paints, where the page shows. Everything painted
after the element paints over the page, as it would over any element: its
children, elements absolutely positioned over it, and the overlay's
popovers, menus, selects, comboboxes, dialogs, tooltips and toasts.
Shadows and translucent backgrounds blend over the page.

```go
ui.WebView(c, app.page).Grow(1).Children(func() {
	// A badge over the page's corner.
	ui.Text(c, "Draft").Absolute().Top(8).Right(8).Padding(2, 8).Radius(8).Background(t.Accent)
})
```

The pointer goes to the page where it shows and nothing painted over it
takes the pointer: a dialog's backdrop or a select's keeps the pointer
from the page while open, a popover keeps it only over its panel, and an
element with `PassThrough` lets it through. Files dragged there drop on
the page. A press on the page closes the popovers it is outside of
(`PressedOutside`), and gives the page the keyboard: the native UI's focus
goes to the web view's element, and the native UI gets the keyboard back
as the user clicks it. The page sets the cursor over it.

Tab stops at the web view among the native UI's controls: its page takes
the keyboard at its first element, or its last with Shift+Tab, Tab moves
through its elements, and past its last one goes on to the control after
the web view. `Focus` gives the page the keyboard where it had it.

## Keys the page leaves

A mail app shows a message's HTML in a web view beside a list that the
keyboard drives: J and K move through the messages, and Cmd+K (Ctrl+K on
Windows and Linux) opens a command palette. The user clicks into the
message to select text or follow a link, and the page has the keyboard,
but J, K and Cmd+K keep working: the keys the page leaves unhandled go to
the native UI as if the web view's element had the focus. The shortcuts
of the elements around it (`Element.Shortcut`) and the window's
(`c.Shortcut`) see them, the keys that scroll scroll the container around
it, and Escape closes a dialog or a flyout of native UI. The keyboard
stays with the page, which still selects, copies, types and follows its
links.

```go
func (app *mail) view(c *ui.Context) {
	if c.Shortcut(ui.Cmd, ui.KeyK) {
		app.palette = true
	}
	reader := ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		app.list(c)
		ui.WebView(c, app.message).Grow(1) // an untrusted page
	})
	if reader.Shortcut(0, ui.KeyJ) {
		app.next()
	}
	if reader.Shortcut(0, ui.KeyK) {
		app.previous()
	}
}
```

The page keeps a key when it:

- cancels it (`preventDefault` in a `keydown` handler);
- types or edits with it in a text field, a textarea or editable content,
  or an input method composes with it, or it types a character with
  AltGr, as @ on a German keyboard, wherever the focus is;
- acts with it: Enter or Space on a link or a button, the arrows on a
  slider, Ctrl+C copying its selection, the arrows, Space and Page Down
  scrolling it while it can scroll that way;
- or it is Tab, which moves through the page and out of it, or a modifier
  key on its own.

The keys of untrusted pages and of the frames in a page come too, and a
page cannot make up a key: the native UI gets only those the user
pressed.

- **macOS** and **Linux**: WebKit decides which keys the page handled,
  and sends the others on to the web view's parent (AppKit's responder
  chain, GTK's `key-press-event` once WebKit's web process answered),
  where MyGo takes them. On Linux WebKitGTK keeps the arrow keys, and
  the shortcuts of the menu items with a role, as Edit › Copy's, go on
  to the menu.
- **Windows**: WebView2 tells of no such key. A script in an isolated
  world of each document, of the page and of its frames, sees the page's
  keys after its handlers ran and tells MyGo of those it left, by the
  rules above. The page's own scripts reach neither the script nor its
  way to MyGo, and the script needs no bridge, so untrusted pages give
  their keys too. A frame of another site, which WebView2 runs in a
  process of its own, keeps its keys.
- **Browsers** (`GOOS=js`) have no web views.

Assistive technology finds the page inside the web view's element, where
it is among the native UI: VoiceOver and Orca read and move through it as
through the controls around it.

## Scrolling the native UI around it

A reader shows a document in a native scroll container: an article
between a native header and its comments, a message in a mail thread, a
rendered note among native cards. The web view is as tall as its page,
which has nothing of its own to scroll, and the wheel over the page should
scroll the container, as it does over the native elements around it.
`WebViewOptions.ScrollToContainer` does that: the wheel and the touchpad
over the web view scroll the native UI under the pointer as over any of
its elements, by lines for a mouse's wheel and by pixels for a touchpad,
in the same direction, with the modifier keys held. The page never
scrolls with them; it still takes the pointer's presses, the keyboard,
selection and a touchpad's pinch.

```go
article, _ := win.NewWebView(mygo.WebViewOptions{URL: "/article", ScrollToContainer: true})

// In the view:
ui.Scroll(c).Fill().Children(func() {
	header(c)
	ui.WebView(c, article).Height(app.articleHeight) // the page's height, which it tells the app
	comments(c)
})
```

Without it, which is the default, the page takes the wheel where it
shows, as a page that scrolls itself needs: docs, a browser's tab, an
editor.

- **macOS**: the web view's scroll events, a touchpad's momentum
  included, go to the native UI instead of WebKit.
- **Windows**: the window keeps the wheel messages over the web view
  instead of sending them on to WebView2. A touchpad's pinch, which
  Windows sends as the wheel with Ctrl held, still zooms the page.
- **Linux**: the web view's scroll events go to the native UI before
  WebKitGTK sees them.
- **Browsers** (`GOOS=js`) have no web views, so the option does nothing
  there.

## Its page

The page of a web view works as a window's does (see
[Bindings](../bindings.md)). `mygo.CallerPage(ctx)` returns the page that
called a bound method, and `Page.WebView` its web view, while
`mygo.CallerWindow(ctx)` returns the window the web view is in, which the
page's window controls (`mygo.window.minimize()`, drag regions) act on.
`Event.Emit` to the window reaches the pages of its web views,
`Event.EmitPage` one page, and `Broadcast` every page.

```go
func (s *Notes) Save(ctx context.Context, text string) error {
	page := mygo.CallerPage(ctx) // the web view's
	...
}
```

The page's `window.close()` leaves the window open. Content rules block
what a web view's page loads, as they do a window's
([blocking content](../windows.md#blocking-content)), from
`WebViewOptions.Page.ContentRules` or `Page.SetContentRules`.

### The typed client

The page calls Go with the client `mygo generate` writes, as a window's
page does: the client knows nothing of web views. A project of native UI
has no frontend, and so no client: give it one, say a Vite app in a
`frontend` directory, with the fields of [the
frontend](../configuration.md#development-and-the-frontend) in its
mygo.json:

```json
{
  "name": "Notes",
  "devUrl": "http://localhost:5173",
  "devCommand": "bun run --cwd frontend dev",
  "buildCommand": "bun run --cwd frontend build",
  "frontendDist": "frontend/dist"
}
```

The client is then `frontend/src/mygo.ts` (`bindings` changes where),
which `mygo dev` rewrites as the Go code changes and `mygo build` before
it builds the frontend; the frontend depends on `mygo-runtime`, which the
client imports. A web view loading `/` shows the frontend, from the dev
server during `mygo dev` and embedded in builds (see [how pages
load](../frontend.md#how-pages-load)), and its page calls the services
bound with `mygo.Bind` and hears the events declared with `mygo.NewEvent`:

```go
// Notes is bound for the pages.
type Notes struct{ likes atomic.Int64 }

// Liked tells pages how many likes there are.
var Liked = mygo.NewEvent[int]("liked")

// Like counts a like and tells every page.
func (n *Notes) Like() int {
	likes := int(n.likes.Add(1))
	Liked.Broadcast(likes)
	return likes
}

var page *mygo.WebView

func view(c *ui.Context) {
	if page != nil {
		ui.WebView(c, page).Fill()
	}
}

func main() {
	mygo.Bind(&Notes{})
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{Title: "Notes", Content: ui.View(view)})
		page, _ = win.NewWebView(mygo.WebViewOptions{URL: "/"})
		win.Invalidate()
	})
	// ...
}
```

```ts
import { Notes, events } from "./mygo";

likeButton.onclick = async () => {
  count.textContent = String(await Notes.like());
};
events.liked.on((likes) => (count.textContent = String(likes)));
```

Calls with a context, errors, [channels](../bindings.md#channels) and
events work as in a window. A page without a build, such as HTML the app
loads with `LoadHTML`, calls by name with the runtime the bridge puts in
every page, `mygo.call("Notes.Like")` and `mygo.on("liked", …)` (see
[without the generated
client](../bindings.md#without-the-generated-client)). Pages that are not
the app's own, as a web view showing `https://example.com`, cannot call Go
at all: see [who may call](../bindings.md#who-may-call).

## Limits

The page is the system's web view, which the window composites with the
native UI; MyGo does not draw it. As with SwiftUI's `WebView`:

- Effects reading what is behind them, as the glass plugin's blur, see
  none of the page: it is not part of the window's frames. The glass
  plugin's `Glass{Native: true}` is the exception: the system's glass view
  sits between the page and the content, so a toolbar over a web view gets
  real Liquid Glass, on macOS 26 and later.
- `Window.CapturePage` of the window shows a hole where the page is;
  `WebView.CapturePage` captures the page.

On Windows, WebView2 keeps the elements of a page it shows this way under
a window of its own: Narrator finds the web view in place, and the page's
elements in that window, not inside it.

## Testing

In a `Tester`, `WebViewAt` returns the web view that takes the pointer at a
point of the window, as the last frame showed it: the one under it, unless
what is painted over it takes the pointer there. `ClickAt` and `Press` over
a web view press the page, which the view hears of as the platform tells
it.
