# Automated end-to-end tests

## Implemented native lifecycle

`make e2e` / `pnpm e2e:native` runs build-tagged Go tests in an actual macOS/Windows desktop session. The app main thread runs MyGo's native loop while the test uses the same production `bootstrap.WindowOptions()` and UI. Only window state persistence is disabled for fixture isolation.

Checks: visible window, native Content rather than a Page, nonblank decoded PNG content capture, minimum geometry, resizing, destruction and one closed callback. App teardown terminates the event loop. A missing OS/session is a failure, not a success-shaped skip. Ordinary `pnpm test` excludes interactive tests; CI native jobs explicitly run them. Local macOS execution passed; Windows CI is configured but local Windows execution is unavailable.

MyGo `CapturePage()` renders Content; it does not verify the OS compositor, GPU pixels, accessibility tree, text injection, microphone or IME. Captures are saved under ignored `.workspace/e2e/<platform>/`. These checks establish native shell lifecycle only.

## Product scenario suite

[16 executable scenarios](../../tests/e2e/desktop-scenarios.json) define actions and observable assertions for shortcut/offline exact-once insertion, cancellation/stale callbacks, focus changes, secure context, denied microphone, IME registration/lifecycle/Unicode/user edits, candidate selection, uncertain commit, encrypted relay conflicts, tampering/revocation/tombstones and offline outbox restart.

Run `make e2e-scenarios DRIVER=... OUT=...` after implementing the OS driver. The [JSONL contract](../../tools/evaluation/README.md) uses actual application/native host binaries and isolated fixtures. The suite withholds expected assertions from the driver. Each result needs live observations, `fixtureOwnership:"isolated"` and nonempty `evidence:[{path,sha256}]` files; the harness verifies hashes and values, not a driver `passed` flag. Artifact checks are required before launch.

The harness requests `cleanup` on success/failure and requires confirmation before a passing report; drivers also handle parent exit/signals. Reset only owned fixtures between scenarios and restore owned network fault injection afterwards, including on failure. Use dedicated test profiles/VMs for microphone denial and IME installation; do not alter a user's real permission database or installed input sources. Editors must report their real text, host selection/range, composition and insertion counts. Network assertions use process-scoped controlled observation. Relay fixtures expose actual ciphertext database/log snapshots; mock persistence cannot fulfill sync acceptance.

Capabilities missing from the hello handshake fail explicitly. Current shell implements none of these product adapters, so this suite has not passed. A synthetic transport unit test checks the harness only. The native-shell test remains independently runnable.

## Coverage ownership

OS drivers belong to MAC-01/WIN-01/IMK-01/TSF-01 with EVAL-01 integration. Application tests still need complete accepted scenarios, including clipboard races, unplug/sleep, missing ranker model, PCM cleanup and key loss. The suite provides concrete boundary cases; it is not a claim that every spec is covered. Expand it alongside each adapter and use acceptance IDs accurately.

On Windows without GNU Make, use pnpm scripts directly. Host claims require actual target OS execution; cross-compiling a test executable is build evidence only.
