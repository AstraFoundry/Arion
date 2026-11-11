# Watch

The watch plugin lets pages follow changes to the directories and files
the app lets them watch, such as a project open in an editor. Go code and
native UI don't need it: they call `mygo.WatchFiles` directly (see
[Watching files](../native.md#watching-files)), which the plugin uses too.

## Set up

Use the plugin in Go with the directories and files pages may watch, each
under a name:

```go
import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/watch"
)

func main() {
	mygo.Use(watch.New(watch.Options{
		Roots: map[string]watch.Root{
			"project": {Path: projectDir, Recursive: true},
			"settings": {Path: settingsFile},
		},
	}))
	// ...
}
```

and add its package to the frontend:

```sh
bun add @mygo-plugins/watch
```

Pages name a root by its name, never by a path, so they can't watch
anything else. A name is lowercase letters, digits and dashes, starting
with a letter. A path must be absolute and a directory or a file when
`mygo.Use` runs, which panics otherwise; it resolves the path's symlinks,
as macOS's `/tmp` is one. `watch.Plugin`, without roots, lets pages watch
nothing.

`Recursive` lets pages watch the directories in the directory too, and
what is in them; without it they can watch the directory itself and the
files in it. `Ignore` leaves out files and directories, by their path
relative to the root, as `mygo.FileWatchOptions.Ignore` does: a directory
it ignores, as `node_modules`, is never opened, and doesn't count in the
entries a watch may cover (`MaxEntries`: 8192, or 65536 on macOS).

```go
"project": {Path: projectDir, Recursive: true, Ignore: func(p string, isDir bool) bool {
	return isDir && (path.Base(p) == "node_modules" || path.Base(p) == ".git")
}},
```

Roots can also be granted while the app runs, such as a folder the user
chose: `Resolve` is asked for the root of a name that is not in `Roots`,
with the call's context, and returns false for names it doesn't know.

```go
var granted sync.Map // name → watch.Root

watch.New(watch.Options{
	Resolve: func(ctx context.Context, name string) (watch.Root, bool) {
		root, ok := granted.Load(name)
		if !ok {
			return watch.Root{}, false
		}
		return root.(watch.Root), true
	},
})

// After the user picked dir in a dialog, a bound method returns the name
// the page then watches.
granted.Store("picked-1", watch.Root{Path: dir, Recursive: true})
```

`Allow`, when set, is asked before each watch, with the call's context
(`mygo.CallerWindow(ctx)` is the window asking) and the `watch.Request`:
the root's name, the path in it and whether it is recursive. It can refuse
it:

```go
watch.New(watch.Options{
	Roots: roots,
	Allow: func(ctx context.Context, r watch.Request) bool {
		return mygo.CallerWindow(ctx) == editorWindow
	},
})
```

Each page can watch 8 directories and files at a time
(`MaxWatchesPerPage`), and all pages 128 (`MaxWatches`).

## Watching

`watch` starts a watch and resolves once everything is watched. Iterate it
for the changes, a batch of events each time they stop:

```ts
import { watch } from "@mygo-plugins/watch";

const watcher = await watch("project", { recursive: true });
for await (const events of watcher) {
  for (const event of events) console.log(event.op, event.path, event.oldPath);
}
```

`path` watches a directory or a file in the root rather than the root
itself, relative to it and separated by slashes (`"src"`,
`"package.json"`); it can't leave the root, nor go through a symlink.

An event has `op`, one of `create`, `write`, `remove` and `rename`, `path`,
relative to what is watched and separated by slashes (`.` is what is
watched itself), `oldPath`, where a rename moved from, and `isDir`. What is
already there when the watch starts is not reported. A file watched is
watched through its directory: a file replaced, as editors save, is a
`write` of `.`, one removed a `remove`, and one that appears again a
`create`. When what is watched, or the directory of a file watched, is
removed, renamed or replaced, the last batch has a `remove` of `.`, and the
iteration ends.

Changes wait for 50 ms of quiet before they are reported, and never more
than 500 ms while more keep coming; `debounceMs` sets the quiet, as Go's
`Debounce`: `0` is 50 ms, and a negative number reports them right away. A
watch compares what it sees with what it saw, so events say what is
different, not every step that made it so: a file written many times in a
row is one write, one replaced at its path by another, as atomic saves
replace it, is a write too, and one created and removed in between is
nothing. Treat
them as hints to read the files again. Go doesn't look again until the page
has taken a batch, so nothing is lost while the page is busy.

`watcher.close()`, `break` out of the loop, or an `AbortSignal` passed as
`signal` stop the watch, and so does the page navigating or closing.
`watcher.closed` settles once Go has stopped it, and rejects with the
error that stopped it. Errors are `WatchError`s, whose `code` is:

- `denied`: the name is not one of the roots, `Allow` refused, the path is
  not in the root, or the page asked for a recursive watch, or one of a
  directory in it, of a root without `Recursive`;
- `not-found`: the path doesn't exist;
- `limit`: the page or the app watch too many, or the directory has more
  than `MaxEntries` files and directories (8192, or 65536 on macOS);
- `unsupported`: the platform can't watch files;
- `path-encoding`: a file's name is not valid UTF-8, which pages can't be
  given;
- `io`: anything else, such as a root replaced since the app started.

## Platforms

Watches follow what is in the directory, not where symlinks and Windows
junctions in it point. Only expose directories the app controls: names
keep pages from choosing paths, but another program can still move or
replace what is in a directory.

- macOS watches with FSEvents: a watch is one event stream covering
  everything beneath its root, so it costs no file descriptor per file,
  and a file watched is covered by the stream of its directory. A
  write reports itself only when FSEvents says the content changed, since
  APFS also calls an O_TRUNC write a creation; the watch's comparison
  reports the rest.
- Linux watches each directory with inotify, through one instance for the
  app, whose limit is the system's (`fs.inotify.max_user_watches`).
- Windows watches a recursive directory with one `ReadDirectoryChangesW`,
  so the directories in it can be renamed, and also watches its parent to
  know when it is renamed. While a directory is watched, Windows doesn't let
  its parent and the directories above be renamed, as for every program
  that watches files. The root of a drive can be watched, and folders of
  cloud files, as OneDrive's.
