# Shift Chinese/English switching

## Scope and evidence limits

Focused static inspection of installed macOS Doubao **1.0.2 (1)** and Chatterfly **1.0.8.13459 (13459)**. These are version-bound instruction observations, not runtime tests. No competitor was launched, no input source or preference was changed, and no private user storage was read. The [ledger](evidence/shift-language-switch.json) records artifact identities and instruction anchors. Complete assembly stays in ignored `.workspace/research/shift-switch/`.

The REA skill guided the investigation. Connected REA tools were unavailable; existing artifact records and ordinary LLVM instruction listings were reused, with focused instructions checked against installed arm64 bytes. These observations are not new REA Evidence IDs or decompiled original source.

## Observed switching paths

| Behavior | Doubao | Chatterfly |
| --- | --- | --- |
| Trigger | A short standalone left/right Shift press, evaluated on release | A short standalone left/right Shift press, evaluated on release |
| Duration gate | Strictly less than 0.35 seconds | Strictly less than 0.3 seconds |
| Press action | Records pending state and press time | Records press time; ShiftDown maps to a null action |
| Release action | Toggles the Chinese/English mode flag | ShiftUp maps to `funcSwitchCnEn`; updates input status |
| Another key during the press | Clears pending Shift state | Clears the Shift timestamp |
| Mode lifetime | Toggle remains after release; no hold-to-English/restore pair in this path | Toggle remains after release; no hold-to-English/restore pair in this path |

This establishes the implemented short-press paths, not whether each user's settings enable them. Default preference values, every host-specific exception, dual-Shift interactions and runtime IMK delivery were not established.

### Doubao anchors

`handleShiftKeyEvent(event:source:)` at `0x100141bfc` accepts keycodes 56 and 60. The down branch at `0x1001420a0` records a timestamp and pending flag. Release reaches the elapsed-time comparison at `0x1001424e8–0x1001424fc`, whose immediate double is 0.35. The success path calls the mode toggle at `0x100142798`. Other physical keys clear pending state at `0x10014657c`; the clear routine is `0x100148760`. A global event path has the same cancellation at `0x1000903a8`.

The handler also rejects duplicate modifier events and includes a 0.05-second debounce comparison. A separate recent-success comparison uses 0.1 seconds to avoid handling duplicate release. These constants serve different gates and are not interchangeable with the 0.35-second short-press threshold.

The mode toggle `0x100195e64` calls `enterCommit(commitEngine: false)` at `0x100195ef4` before inverting the English flag at `0x100195f14–0x100195f24`, then resets the engine session. The commit routine combines selected segment strings with the remaining `input_` buffer and calls the host insertion path at `0x10018871c`. Thus this path preserves unfinished input through an explicit commit, rather than simply discarding it; the exact visible result depends on the current composition.

### Chatterfly anchors

Left/right down/up handlers and their condition dispatch distinguish press from release. The release handlers compare elapsed time with the 0.3-second double at `0x101daf128`. Only a successful condition dispatch invokes the bound switch action. `condPressAnyKey` clears the timestamp at `0x1002ca074–0x1002ca08c`, suppressing a switch after typing another key with Shift held.

The retained `buildCurrentKeyItem:` instructions also normalize modifier events whose keycode is `0` or `255`: they compare the current and previous device-independent modifier flags and map a single changed Shift flag to keycode 56 (`0x1002cb660–0x1002cb79c`). This is a bounded compatibility path; ambiguous simultaneous changes do not identify a Shift tap.

The Pinyin switch path calls `deactivate:withIdentifier:isMaskFlagKey:` at `0x1002dd328`. Its ordinary composition branch obtains `realComposingText` at `0x1002d9280` and calls `pyForceCommitInline:sender:isMaskFlagKey:` at `0x1002d92ec`. The helper applies `pyConvertCommitText:isForceCommit:` before insertion and has commit-in-progress and host-specific guards. This is evidence of a composition-preserving force-commit path, not a guarantee that every host or composition inserts identical raw Pinyin.

## Implication for Arion

Both inspected implementations support a standalone short Shift press that switches on release and keeps the new mode. A Shift chord must invalidate the pending switch so uppercase letters and shortcuts do not change language. This is separate from a temporary English mode while Shift is held.

Arion now supports standalone left/right Shift release in less than 0.3 seconds, with chord, cancellation and deactivation invalidation; Shift+Space and the input menu remain available. The IMK controller requests modifier events and the typing owner tracks the pending press. For missing keycodes `0`/`255`, it recognizes Shift only from an unambiguous Shift-flag transition; known left/right keycodes retain their side checks. A separate caret badge provides transient feedback. The badge now uses an original 23 × 23 point solid rendering, no window shadow and a one-second lifetime, aligned with the subsequently inspected [Chatterfly default path](chatterfly-language-hint.md).

KBD-04 now commits pending raw input before standalone Shift switches Chinese to English. Shift+Space and input-menu switching retain their discard policy. The competitor observations above remain version-bound evidence, separate from Arion's selected behavior and timing threshold.

## Arion host regression

A local physical-keyboard trace reproduced the reported failure in Codex and Cursor: one short Shift tap delivered down/down/up/up to the IMK controller. The second down arrived about 4–14 ms after the first with the same keycode and flags. Arion treated that duplicate as interference and cleared `shiftPending`, so neither release switched mode. The owned AppKit editor delivered one down/up pair and switched correctly. This is Arion runtime evidence, separate from the version-bound competitor findings above.

The typing owner now ignores a repeated down for the same side and unchanged flags without resetting its original deadline or restoring a cancelled tap. Duplicate releases cannot switch twice; the other Shift side and intervening keys/modifiers still invalidate the pending switch. Native engine/AppKit regressions replay the captured sequence, long holds with a late duplicate, chords and anonymous modifier keycodes. Codex/Cursor acceptance additionally requires observing exactly one successful switch per physical tap in those applications.

The signed repair was installed locally through the desktop and its input-method update flow. The user then reported normal physical Shift switching after the update. Separately, rebuilding the obsolete Input Monitoring entry for the current signed desktop identity changed the app’s observed consent from denied to granted and yielded an enabled native tap with both requested event types. Production diagnostics recorded Right Option start, microphone capture, stop and a completed speech session with `commit_result=applied`. This user-assisted local acceptance does not establish every application or physical Intel/Windows behavior.
