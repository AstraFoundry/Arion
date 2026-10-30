# ADR 036: Two-stage local keyboard ranking

## Status

Accepted. Supersedes the single-worker scheduling and context scope in [ADR 031](031-local-keyboard-neural-ranking.md).

## Decision

The verified connection advertises `keyboard-ranking-v2`, so old hosts that accept only one proposal do not receive this protocol. Use separate latest-only scheduler lanes: the pinned `chinese-ime-lm` worker starts on each eligible composition revision, with a 100 ms budget and no debounce; the resident local polishing model starts after a 150 ms pause, with a 900 ms total budget. These are configured limits, not measured latency guarantees. The native candidate menu appears immediately. Both passes receive the original snapshot; a settled result supersedes a late fast result. Native hosts rebuild from engine identities before applying either promotion. Navigation, commit, cancellation, voice takeover and focus changes revoke both lanes.

The architecture follows the inspected [msime runtime](https://github.com/metasequoiaime/msime/blob/0e2824c8a2bff6a010e7a40525ff348d0a1a231b/crates/input-runtime/src/runtime.rs). The fast model uses its `chinese-ime-lm` reference implementation and `best_where` policy: only whole-input candidates compete, and a trusted dictionary leader is retained. The settled pass reuses Arion's existing Qwen worker, rather than msime's desktop model. This architecture is not a claim of equal quality to msime or a completed model-selection benchmark.

The fast worker is built from locked Cargo dependencies. Weights are separately downloaded on explicit request through the existing verified file-model store; their URL, size, digest and license are owned by [the model catalog](../../models/catalog.json). The worker requires embedded Apache-2.0 attribution. No weights are committed or silently downloaded while typing. Both workers release when their capabilities are disabled. Speech suspends ranking. A bounded line protocol carries only current candidates and approved context; a one-second watchdog terminates a hung fast worker. Cancellation drains the bounded response before another request so replies cannot cross revisions.

Keyboard cursor and window-context switches default off and are independent of voice polishing. On macOS, reuse the existing Accessibility target adapter, not OCR. Each enabled pass captures the current field, checks its application against the verified native activation, refuses secure/denied/unsupported targets, and releases the target after reading. Cursor reads must end in the native host’s exact marked text; that uncommitted suffix is removed before scoring, and mismatches reject the enhancement. Cursor text is available to the fast pass; the settled pass may additionally read current-window Accessibility text. Context reads occur off the input thread and are discarded on cancellation or changed focus. No screenshots, screen-recording request, cached cross-field context, logs, sync, cloud credentials or network inference are introduced. Windows retains confirmed-commit context; these extra switches are hidden until its adapter exposes a verified application and caret identity.

The completion prompt receives separate preceding, following and window fields. Bounds are 2,048 UTF-8 bytes per caret side and 8,192 bytes for window text; oversized window context rejects that enhancement rather than silently truncating it. First compositions can reach the fast model without a previous commit. Empty context still makes the general completion pass abstain. Both models may legitimately retain the producer's first candidate.

## Validation

Public regression tests cover two independent lanes, late-result suppression, repeated native promotions from original engine IDs, target/consent admission, bounded worker failure and an opt-in real fast-model smoke. UI snapshots cover both supported locales. Synthetic smoke cases establish plumbing, not broad accuracy or installed-editor acceptance. Windows native execution remains a separate target-OS check.
