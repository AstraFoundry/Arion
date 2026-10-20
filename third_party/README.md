# Maintained dependency snapshots

`mygo/` is the complete Go module snapshot of `github.com/egoist/mygo v0.4.6`, under its original MIT licence. `mygo-upstream-sha256.json` records every original module file's SHA-256 before patching. The desktop's `go.mod` replaces that dependency with this directory; the module cache is never modified. Package notices identify the local patch.

The `prerelease-channel-v1` patch adds `Updater.CheckWithOptions` and `UpdateCheckOptions.IncludePrereleases`. GitHub channel checks resolve published releases through the existing resolver, always exclude drafts and prereleases without a manifest for the running platform, and expose `ErrNoPublishedUpdates` separately from transport errors. The ordinary `Check` API retains its existing behavior. Signature validation, archives, deltas and installation are unchanged. `App.Version()` prefers the linked complete semantic version over macOS's numeric bundle version, while preserving explicit overrides.

Run `make updater-check` or `pnpm updater:check` for upstream updater/signature/install tests plus channel regressions. Native application tests cover preference persistence, stale-check invalidation and the independent settings switch. When upgrading MyGo, compare the recorded original bytes, rebase the patch and run these gates before replacing the snapshot.
