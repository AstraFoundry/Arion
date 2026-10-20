# Native UI in browsers

An app of native UI also runs in a browser, compiled to WebAssembly: its
windows show on canvases of a page, drawn with WebGL 2 by the same
renderer as on Linux, with the same shaders. Nothing changes in the app:

```go
func main() {
	s := &counter{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Counter", Content: ui.View(s.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`mygo dev -web` develops it in the browser: it compiles the app to
WebAssembly, serves it at `http://localhost:5174`, opens it, and rebuilds it
as you edit, which reloads the page. `mygo build -platform web` builds it
into `<out>/web`, a static site to serve from anywhere: `index.html`, Go's
`wasm_exec.js` and the app's WebAssembly (see [the CLI](../cli.md#mygo-build)).

```sh
mygo dev -web
mygo build -platform web
```

Without the CLI, build it for `js/wasm`, with Go's loader of WebAssembly
beside it:

```sh
GOOS=js GOARCH=wasm go build -o app.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
```

and load it from a page:

```html
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>html, body { margin: 0; height: 100%; }</style>
    <script src="wasm_exec.js"></script>
    <script>
      const go = new Go();
      WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then((r) => go.run(r.instance));
    </script>
  </head>
  <body></body>
</html>
```

Serve the directory over HTTP, with WebAssembly as `application/wasm`, as
most servers do. A window fills the element with the ID `mygo`, or else
the page; a window opened later covers the earlier ones. The page sets the
window's size: the size, position and title bar options do nothing, and
the title of the window opened last is the page's.

## What works

- **Drawing.** Frames draw with WebGL 2 at the device pixel ratio, once
  for each of the browser's frames that something changed in; effects, as
  [Liquid Glass](../plugins/glass.md), draw as on the desktop. Where the
  browser has no WebGL 2, frames draw in memory and show through a 2D
  canvas. A lost WebGL context draws again once the browser restores it.
- **Text** comes from the browser, which finds the fonts, falls back for
  what one lacks and shapes, as for its pages: `system-ui` is the system's
  font, as a page's. Lines break and runs order (bidi) as in package text
  everywhere. Text is not drawn with subpixel antialiasing.
- **Input**: the pointer, with the clicks counted; the wheel and touchpads;
  keys; input methods, which compose in a text area out of sight at the
  caret; and the clipboard's text, through the browser's copy, cut and
  paste. On a Mac, ⌘ is Control for the view's shortcuts, which take a
  browser for a system other than macOS.
- **Routers.** The first [router](navigation.md) a window shows follows
  the tab's history, in the URL's fragment, as `#/notes/42`: the browser's
  back and forward buttons go through its pages, and a link to a page, or
  a reload, opens the app there.
- **Files dropped** on a window: `DroppedFiles`, `FileDragOver` and
  `OnFileDrop` work as on desktops. Browsers give no paths, so the
  backend reads the files as they drop and keeps them at paths of its own,
  as `/mygo-drop/3/photo.png`, which `os.Open`, `os.ReadFile`, `os.Stat`
  and `os.ReadDir` read; the files of the last 16 drops stay.
- **Context menus** show as menus of HTML on the page.
- **Dark mode**, reduced motion and more contrast follow the browser's
  settings, and change with them. A page can choose the appearance itself
  with `data-mygo-theme="dark"` or `"light"` on its root element, as one
  framing the app does to give it its own theme (a frame's
  `prefers-color-scheme` follows the browser, not the page around it).
- `Dialog.Message` asks with the browser's `alert` and `confirm`, and
  `Shell.OpenExternal` opens a tab.

## What does not

Windows of web pages, application and Dock menus, trays, notifications,
global shortcuts, file dialogs and the drag and drop of data other than
files have nothing
in a browser to stand for them, and report so. Assistive technology does
not see the content yet. Of a style's OpenType features, canvases apply
`kern` only. A font added with `ui.RegisterFont` needs its family, and
draws once the browser loaded it. The clipboard's text comes from the
system when a paste brings it: the view's own Paste command pastes what
the app copied last.

## Size

The counter example is 8.2 MB of WebAssembly, 2.2 MB compressed with
gzip, and the previews of these docs, with every component, 9.8 MB
(2.6 MB). Builds for browsers leave out what package `mygo` has for web
pages, which they do not show: its IPC, custom schemes and their HTTP, the
TypeScript generator, the updater, with the JSON and the cryptography
they take. Their API stays, doing nothing: `Bind` binds nothing, events
reach no page, `Updater.Check` reports `ErrUpdatesDisabled`. Only
`Protocol.Handle`, `HandleFunc` and `FileServer`, which take net/http's
types, are missing, so that code using them is for desktops
(`//go:build !js`). The tag `mygo_noinspector`, which `mygo build` sets
for production, leaves out the inspector of native UI too, some 350 KB.
Building for browsers adds nothing to apps for desktops: the browser's
code is only in `js/wasm` builds.

## Under the hood

Go's main goroutine runs the event loop; the browser's events reach it
through the functions JavaScript calls, which wait for it to handle them,
so that they can still prevent the browser's defaults. The architecture
guide's [Browsers](../architecture.md#browsers-internalweb) section has
the details.
