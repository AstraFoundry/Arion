# Launcher

An app launcher in the manner of Spotlight and Raycast, in native UI. Run
from the repository root:

```sh
go run ./examples/launcher
```

Press **Alt+Shift+Space** (⌥⇧Space on macOS) anywhere to show the panel,
over every app, full-screen ones too, on the display with the pointer.
Type to find an installed app: names starting with what you type come
first, then those with a word starting with it, those containing it, and
those with its letters in order ("vsc" finds Visual Studio Code). Up and
Down choose, Enter or a double click opens the app, Escape clears the
query, then hides the panel, as clicking elsewhere and pressing the
shortcut again do. Where another app took the shortcut, it is
Cmd+Alt+Shift+Space (Ctrl+Alt+Shift+Space on Linux and Windows).

On macOS the panel takes the keyboard without activating the launcher:
the app you were in keeps its menu bar, and has the keyboard back once the
panel hides. There is no Dock icon or taskbar button. The tray icon's menu
opens the panel, starts the launcher at login, and quits it; on Linux it
needs `libayatana-appindicator3`.

The example needs Go alone. To package it, as an app without a Dock icon
from the start (`LSUIElement` in `mygo.json`):

```sh
go run ./cmd/mygo build examples/launcher
```

See [the Launchers guide](../../docs/launchers.md) for how it is made.
