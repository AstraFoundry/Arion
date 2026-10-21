# macOS keyboard shortcut incident

## Impact and outcome

Short physical Shift taps failed to switch Chinese/English in Codex and Cursor, although Shift+Space and an owned AppKit editor worked. Right Option also failed to start dictation after the initial update. Both actions subsequently passed user-assisted acceptance on this Mac with the signed desktop app and updated input method.

The repairs apply to the shared IMK modifier handler and global dictation listener. They contain no Codex/Cursor application-name checks. This establishes local acceptance in the reported editors, not compatibility with every application.

## Findings

| Failure | Observed evidence | Cause |
| --- | --- | --- |
| Standalone Shift did not switch | Physical taps delivered `down/down/up/up` in the reported editors; duplicate presses arrived about 4–14 ms apart with identical keycode and flags. The owned AppKit fixture delivered one pair. | The second press cleared the pending tap as interference, so release could not switch. Existing tests did not replay duplicate host notifications. |
| Right Option registration appeared successful without events | Input Monitoring preflight denied access; the tap was disabled and its effective event mask was `0x1000`, rather than the requested `0x1400` for key-down and modifier changes. | Startup treated a non-null event-tap object as proof of readiness. |
| System Settings showed an enabled Arion entry while runtime consent remained denied | Relaunch did not repair consent; TCC diagnostics referenced the older `local.arion.desktop` development identity. Removing the Arion entry and adding the current `/Applications/Arion.app` through System Settings restored granted consent and an enabled `0x1400` tap. | The existing privacy entry did not authorize the current signed application. The exact internal TCC record was not inspected, so its internal corruption mechanism is unproven. |

Review also found a listener lifecycle race: the native worker was detached, and stopping did not wait for callbacks or initial run-loop startup to finish. This was repaired and regression-tested; it was not established as the trigger of the reported physical-key failure.

## Changes

- Ignore duplicate Shift presses with the same side and unchanged flags, preserving the original 0.3-second deadline. Duplicate releases cannot switch twice. Chords, opposite-side Shift, long holds, cancellation and deactivation retain their invalidation rules.
- Recognize missing IMK keycodes `0`/`255` only when the modifier transition unambiguously identifies Shift. This compatibility behavior aligns with the retained [Chatterfly evidence](../research/shift-language-switch.md); missing keycodes were not the cause of the captured physical-key failure.
- Require actual Input Monitoring consent and an enabled event tap before reporting listener readiness. Distinguish permission denial from adapter failure, restart the selected key after consent, and stop/cancel on revocation.
- Publish readiness from the running native loop, retire the tap on that loop, join its worker before release, and reject queued events from retired listener generations.
- With explicit user authorization and user-completed macOS authentication, re-add only the current Arion application in System Settings and use Quit & Reopen. Other applications' permissions were unchanged; the TCC database was not edited.

## Verification

| Check | Result and scope |
| --- | --- |
| Native typing regressions | 36/36 on arm64 and 36/36 x86_64 under Rosetta 2, including duplicate events, anonymous keycodes, deadline preservation and cancelled taps. Physical Intel hardware was not tested. |
| Native listener harness | Passed denied/disabled/creation failures, 100 immediate start/stop cycles, callback delivery, timeout recovery and consent revocation/recovery using the production listener with a controlled Core Graphics boundary and real run loop/thread lifecycle. It does not grant OS permissions. |
| Negative controls | The previous typing implementation failed the two new host-event cases; the previous listener failed the denied-consent-before-creation assertion. |
| Desktop checks | Go tests and vet passed; race checks passed for bootstrap, hold-key interpretation and the macOS adapter. |
| Native window E2E | All six enabled cases passed. Five explicitly opt-in scenarios were skipped and are not claimed as passes. |
| Installed build | Signed app and input method passed strict code-signature verification. Installed IME executable SHA-256: `cb19a5d1fa40c9300cbd88d8cd8436abde48bbeaba09387940e762a11b2ea40a`. This was a local signed build, not a notarized release. |
| Physical acceptance | User reported Shift and Right Option working in the reported editors. Production diagnostics recorded key start, microphone capture, key stop and a completed speech session with `commit_result=applied`; runtime consent was granted and the tap enabled with mask `0x1400`. |

## Prevention and limits

`pnpm desktop:check` now includes the native listener harness. The [E2E guide](e2e.md) distinguishes deterministic native regressions from physical keyboard and OS permission acceptance. [Local development troubleshooting](../operations/local-dev.md#troubleshooting) documents safe recovery of stale application permission entries.

An owned native editor alone is insufficient evidence for host event delivery. Future modifier changes must replay captured duplicate notifications and retain chord/deadline tests. Listener readiness must continue to reflect runtime permission and enabled state. Windows, physical Intel hosts and unrelated editors remain outside this incident's acceptance evidence. Personal audio, text and raw diagnostic traces remain outside the committed report.
