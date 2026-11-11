# Documents

Apps whose windows each show a file, such as a text editor, a drawing app
or a notes app, are document-based apps. `mygo.NewDocuments` gives them
what AppKit's `NSDocument` and SwiftUI's `DocumentGroup` give Mac apps, on
every platform:

- a window per document, with its name in the title, and on macOS the
  file's icon in the title bar and a dot in the close button while it has
  unsaved changes;
- New, Open, Open Recent, Save, Save As, Duplicate, Rename, Move To and
  Revert, in the File menu;
- autosave, and asking to save only where saving can't happen by itself;
- a recovery copy of every unsaved change, so that a crash or a quit loses
  nothing;
- the documents opened from Finder, Explorer, the Dock and the jump list;
- the files that other apps change while they are open.

A notes editor in [native UI](ui/README.md), the whole app:

```go
func main() {
	mygo.NewDocuments(mygo.DocumentsOptions[string]{
		Types: []mygo.DocumentType{{Name: "Markdown Document", Extensions: []string{"md"}}},
		Read: func(r io.Reader) (string, error) {
			b, err := io.ReadAll(r)
			return string(b), err
		},
		Write: func(w io.Writer, text string) error {
			_, err := io.WriteString(w, text)
			return err
		},
		Equal: func(a, b string) bool { return a == b },
		Window: func(doc *mygo.Document[string]) *mygo.Window {
			text := doc.Content()
			var win *mygo.Window
			win = mygo.NewWindow(mygo.WindowOptions{Content: ui.View(func(c *ui.Context) {
				ui.TextArea(c, &text).Fill().OnChange(func() { doc.Set(text) })
			})})
			doc.OnReload(func() { text = doc.Content(); win.Update(func() {}) })
			return win
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`examples/documents` is this editor with a status line and a bar for files
changed by other apps:

```sh
go run ./cmd/mygo dev examples/documents
```

## Documents and their content

`DocumentsOptions[T]` describes one kind of document, whose content is a
`T`:

| field | |
|---|---|
| `Types` | the kinds of files, by name and extensions, e.g. `{Name: "Markdown Document", Extensions: []string{"md", "markdown"}}`. Open dialogs show them; new documents are saved with the first extension |
| `Read` | reads a content from a file, on a goroutine |
| `Write` | writes a content to a file, on a goroutine |
| `Window` | creates the window of a document, on the main thread |
| `New` | the content of a new document; the zero `T` without it |
| `Equal` | compares contents, so that undoing back to what the file has leaves the document unedited |
| `Snapshot` | copies a mutable content before it is written |
| `Autosave` | `AutosaveInPlace` (the default) or `AutosaveOff` |
| `AutosaveDelay` | how long after the last edit an autosave happens: 5 seconds |
| `AskOnQuit` | asks to save at quit, rather than keeping the documents for the next launch |
| `NoUntitled` | no untitled document when the app starts with nothing to open |

Call `NewDocuments` before `App.Run`. When the app starts, it opens the
files the app was started with, the documents a crash or a quit left with
unsaved changes, or else a new untitled document. An app may make several
kinds of documents; the first is the one File > New creates.

A `Document[T]` is an open document. Its methods are safe from any
goroutine:

```go
doc.Content()   // the content
doc.Set(v)      // replaces it, which edits the document
doc.State()     // name, path, edited, read-only, conflict, deleted
doc.Window()    // its window
doc.Save()      // as File > Save
```

### Values and mutable contents

The content is best a value the app replaces with `Set`: a string, a
struct of values, or a persistent snapshot such as `ui.TextSnapshot` or
`editor.Snapshot`, which share what they don't change. Saving writes the
content as it was when saving started, on a goroutine, while the user goes
on editing, so a value that nothing changes is already a snapshot.

A mutable content, such as a pointer to a model the app changes in place,
gives `Snapshot`, which copies it on the main thread, where the app changes
it, and tells about changes with `Changed`:

```go
mygo.DocumentsOptions[*Drawing]{
	Snapshot: func(d *Drawing) *Drawing { return d.Clone() },
	// …
}

drawing.Shapes = append(drawing.Shapes, shape)
doc.Changed()
```

### The edited state

A document is edited when its content differs from its file's:

- `Set` and `Changed` edit it. With `Equal`, a `Set` of the content the
  file has makes it unedited again, which is what undoing back to it, or
  typing a character and deleting it, does: the undo of text widgets just
  works.
- `SetEdited` sets it, for an app that knows better, such as one keeping
  its own undo history: `false` makes the content what the file has,
  without writing it.
- Saving makes it unedited, unless it changed while it was written: those
  changes stay unsaved.

`OnStateChange` tells about changes of the state, on the main thread.

## Windows

The window of a document shows its name: the file's name, or "Untitled",
"Untitled 2"… in the language the app shows. The title of a document's window is its name on macOS, and
on Linux and Windows the name and the app's, as Notepad and gedit show
them. A page's `<title>` doesn't change it.

While the document has unsaved changes, macOS shows a dot in the window's
close button and Linux and Windows a `*` before the title. On macOS the
title bar also shows the file's icon, which the user drags to another app,
and Command-clicking the title lists the folders the file is in.

Windows of any app can show the same, with `Window.SetRepresentedFile` and
`Window.SetDocumentEdited`. `Window.Title` returns the title without the
`*`.

## Saving

| | |
|---|---|
| `Save()` | writes the document to its file. An untitled document asks where, a read-only one asks for another file, and one whose file another app changed asks whether to overwrite it. `ErrCanceled` tells that the user canceled |
| `SaveAs(path)` | writes it to another file, which the document then is; `""` asks where |
| `SaveAnyway()` | writes it even when another app changed its file |
| `Duplicate()` | opens an untitled copy, named "Notes copy" |
| `Rename(name)` | renames its file, in its folder; `""` asks |
| `MoveTo(dir)` | moves its file to another folder; `""` asks |
| `Revert()` | reads its file again, dropping its unsaved changes |
| `Close()` | closes its window, which asks to save first where it must |

Files are written as `NSDocument` and GIO write them: into a new file next
to the document, flushed to the disk, which then replaces it, so that a
crash or a full disk never leaves half a file. The file keeps its metadata:
on macOS through `NSFileManager`, which keeps its permissions, creation
date, Finder tags and other extended attributes, ACLs, and its identity, so
aliases still find it; on Windows through `ReplaceFileW`, which keeps its
security descriptor, attributes, alternate data streams and creation time,
and waits a moment for another program holding it open; on Linux with its
permissions, owner and group. A symbolic link's target is replaced, and a
file with other hard links, or in a folder the user can't create files in,
is written in place. When `ReplaceFileW` stops halfway, having removed the
file, the new one is moved into its place; if even that fails, the new file
stays next to where the document was, and the save's error names it.

Before writing, the file is compared with what the document last read or
wrote (its modification time and size): when another app changed it,
`Save` asks, and autosave writes nothing.

## Autosave

With `AutosaveInPlace`, the default, a document is saved to its file a
moment after the user stops editing (`AutosaveDelay`, and at least every
30 seconds while the user keeps editing), when its window loses the focus,
and when it closes, as Mac apps do. Saved documents never ask to be saved.

Documents that can't be saved in place aren't: untitled documents,
documents whose files are read-only, were deleted, or were changed by
another app. Those ask when they close.

With `AutosaveOff`, documents are saved when the user saves them, and ask
when they close, as Windows and Linux apps traditionally do.

## Recovery copies

Every document with unsaved changes that autosave doesn't write keeps a
recovery copy, a second after the user stops editing, as VS Code's backups
and Zed's database keep theirs: an untitled document, a document of an app
without autosave, a document in conflict. The copies are in
`Recovery/<kind>` in `PathUserData`, written by `Write`, with the file's
modification time and size, and go when the document is saved, reverted or
closed without saving.

When the app starts, the documents a crash or a forced quit left come back,
edited. A document whose file changed meanwhile is in conflict. Each
running instance of the app keeps its copies in a session of its own,
locked while it runs, so another instance never opens them.

## Closing and quitting

| | asks |
|---|---|
| closing an unedited document | nothing |
| closing a saved, edited document, with autosave | nothing: it is saved first |
| closing an edited untitled document, with autosave | "Do you want to keep this new document “Untitled”?" Save…, Delete, Cancel |
| closing an edited document without autosave, or one that can't be saved in place | "Do you want to save the changes made to the document “Notes.md”?" Save, Don't Save, Cancel |
| quitting | nothing: the documents are saved in place or kept, and come back edited at the next launch |
| quitting with `AskOnQuit`, several documents | macOS: "You have 3 documents with unsaved changes. Do you want to review these changes before quitting?" Review Changes…, Discard Changes, Cancel, then a sheet for each. Linux and Windows: one dialog naming them, Save All, Don't Save, Cancel |

The questions, their buttons, and the names of untitled documents are
[MyGo's own strings](i18n.md#mygos-own-strings), in the language the app
shows. On macOS the questions are sheets of the document's window. A close that
must ask keeps the window open until the user answers; a quit waits for the
documents, then quits, and Cancel keeps every window open. The quit's
`OnBeforeQuit` listeners run again then.

A window destroyed without asking (`Window.Destroy`, `App.Exit`) keeps the
recovery copy of its document's unsaved changes, which come back at the
next launch.

## Files changed by other apps

Documents watch their files (see [watching
files](native.md#watching-files)), and look at their file when their window
is focused, as VS Code and Notepad++ do, since watches miss changes:

- a document without unsaved changes reads its file again, and
  `OnReload` tells the app to show the new content;
- a document with unsaved changes is in conflict: its state says so, and
  saving asks whether to keep it or the other app's ("Save Anyway",
  "Revert");
- a file renamed by another app, in its folder, is followed;
- a deleted file leaves its document open, `Deleted` in its state, and
  saving writes the file again.

## The File menu

The menu roles of documents act on the document of the focused window, and
are enabled for it:

| role | macOS | Linux, Windows |
|---|---|---|
| `RoleNew` | New ⌘N | New Ctrl+N |
| `RoleOpen` | Open… ⌘O | Open… Ctrl+O |
| `RoleOpenRecent` | Open Recent | Open Recent |
| `RoleSave` | Save ⌘S | Save Ctrl+S |
| `RoleSaveAs` | Save As… ⌥⇧⌘S | Save As… Ctrl+Shift+S |
| `RoleDuplicate` | Duplicate ⇧⌘S | Duplicate |
| `RoleRename` | Rename… | Rename… |
| `RoleMoveTo` | Move To… | Move To… |
| `RoleRevert` | Revert to Saved | Revert |

`RoleFileMenu` holds the File menu of each platform when the app has
documents: DocumentGroup's on macOS (New, Open, Open Recent, Close, Save,
Duplicate, Rename, Move To, Revert to Saved), Notepad's and KDE's
elsewhere (New, Open, Open Recent, Save, Save As, Revert, Close, Quit).
Apps with documents get a menu bar on Linux and Windows too, with File and
Edit menus, until they set their own.

Open Recent lists the documents the app opened and saved lately, newest
first, which `App.RecentDocuments` returns: documents record their files
with `App.AddRecentDocument`, which also records them with the system (the
Dock menu on macOS, the jump list on Windows, the recently used files on
Linux), and its Clear Menu calls `App.ClearRecentDocuments`. See
[recent documents](app.md#recent-documents).

## Opening files

Documents open the files the system asks the app to open (`App.OnOpenFile`)
of their extensions: from Finder or Explorer, the Dock or the jump list, or
the command line. List the types in `fileAssociations` of the
[configuration](configuration.md), which makes the system offer the app
for them:

```json
"fileAssociations": [
  { "ext": ["md", "markdown"], "name": "Markdown Document", "mimeType": "text/markdown" }
]
```

On Linux and Windows every file opened starts the app again: call
`App.RequestSingleInstanceLock` so that the first instance opens them.

A file that is open already focuses its window. On macOS the app stays open
without windows, as document apps do, and clicking its Dock icon makes an
untitled document.

## Web pages

The pages of document windows get their documents through the generated
TypeScript client: bind the service of the documents under the name the
client calls it by,

```go
notes := mygo.NewDocuments(mygo.DocumentsOptions[Note]{ /* … */ })
mygo.BindAs("Note", notes.Service())
```

and the page reads and replaces the content of its window's document, and
follows its state:

```ts
import { Note } from "./mygo"; // generated
import { onDocumentReload, onDocumentState } from "mygo-runtime";

editor.value = (await Note.content()).body;
editor.oninput = () => Note.setContent({ body: editor.value });
onDocumentState(({ name, edited }) => { status.textContent = edited ? "Edited" : "Saved" });
onDocumentReload(async () => { editor.value = (await Note.content()).body });
```

The service has `content`, `setContent`, `state`, `save`, `saveAs`,
`duplicate`, `rename`, `moveTo` and `revert`, for the window that calls
them. A page sending its whole content on every keystroke suits notes and
small documents; for large ones, send it when the user pauses.

## Platform differences

| | macOS | Linux | Windows |
|---|---|---|---|
| edited mark | dot in the close button | `*` before the title | `*` before the title |
| file in the title bar | proxy icon, Command-click for its folders | none | none |
| title | the name | name - app | name - app |
| replacing files | `NSFileManager` `replaceItemAtURL:` | rename, with mode, owner and group | `ReplaceFileW` |
| questions | sheets of the window | dialogs | dialogs |
| without windows | the app stays open; the Dock icon makes a document | the app quits | the app quits |

## What is left out

What only a real `NSDocument` does on macOS: renaming in the title bar's
popover, "— Edited" next to the title, browsing versions and Revert To a
version, locking files, and drafts in iCloud. Documents that are folders
(packages) aren't supported yet, and browsers have no files to open.
