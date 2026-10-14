# ADR 012: Scene selection from the focused field, site and app

## Status

Accepted (product decision). Extends [ADR 011](011-live-local-rewriting.md); its prompt `live-v2` stays unchanged.

## Context

Chatterfly rewrites dictation differently for email, workplace messages, coding-agent prompts and general typing, and picks the scene automatically. Its client sends the frontmost app's bundle ID and window title, the field type, text around the cursor, the clipboard and per-app input history to a server that classifies the scene and then rewrites for it ([research](../research/chatterfly.md#scene-recognition-and-switching)). Arion's live pipeline always asked its local model for plain chat text, except in benchmark fixtures that declared a scene.

Arion runs a 1.7B model on the device and reads no field text. A prompt that taught three more scenes (`live-v3`: work, prompt and search instructions plus six demonstrations) lowered the held-out live benchmark from 27 to 22 passes and raised final latency, so scene support cannot come from a longer prompt ([report](../../benchmarks/results/scene-switching-v1/README.md)).

## Decision

- **Signals.** The scene is decided once per session, after target capture, from metadata only: a search field (AX subrole), then the http(s) host of the page owning the focus (the outermost `AXWebArea`'s `AXURL`), then the app's bundle ID, else chat. Listed sites match exact hosts ignoring `www.`; workspace domains such as `feishu.cn` match every subdomain. The page host replaces Chatterfly's window title, which can contain mail subjects and document names. Cursor text, clipboard and history are not read for this.
- **Scenes.** Chat, email, work, prompt and search, mirroring Chatterfly's catalog plus its field-type signal; list remains for explicit list formatting.
- **Layout without a new prompt.** The model is asked only for layouts `live-v2` already teaches: email keeps `[email]`; work messages and coding prompts use `[list]` when the speaker enumerates points (第一…第二, 一是…二是, first…second) and `[chat]` otherwise; search uses `[chat]`. Mechanical layout is deterministic: a search query is delivered as one line without sentence-final punctuation, also when the rewrite was rejected. The validator lets a numbered list replace spoken ordinals when checking for dropped content.
- **Control.** "Match the app" is on by default and appears under Settings › Model while polishing is on. Off, every session is chat and the page host is not read.
- **Maintenance.** The built-in tables are defaults. The menu bar panel shows the scene of the app that was frontmost when it opened; choosing another scene stores it for that site or app, and it wins over the tables from the next dictation on. Chatterfly maintains its scene policy on its server; Arion keeps the user's corrections on the device and needs no network. Scene decisions are local, deterministic tables; nothing leaves the device and hosts and bundle IDs are never logged.

## Consequences

- On the held-out scene set, delivered passes rose from 14 to 21 of 30, mostly from email layout and search finishing; the live regression sets stayed at 27/30 and 17/24 ([report](../../benchmarks/results/scene-switching-v1/README.md)). Plain chat output is unchanged because the prompt and its KV prefix are unchanged.
- Numbered lines for enumerated work messages and coding requests depend on the 1.7B model following `[list]`, which it did not reliably do on new cases.
- The tables need upkeep: unlisted apps and sites fall back to chat, and scenes that depend on a page path (iCloud Mail, Feishu documents on the messenger's domain) cannot be told apart by host.
- Reading the page URL relies on the browser exposing its accessibility tree. It is not yet verified on a trusted build (SCN-A04); without it, browser targets fall back to app and field signals.
- Chatterfly's coding scene adds constraints from page and codebase context; Arion only lays out what was said.

## Alternatives considered

- **Teach every scene in the prompt** (`live-v3`): measured regression on the held-out live set and higher latency.
- **Window title as page identity** (Chatterfly): broader content exposure and needs Screen Recording permission for other apps' window names on current macOS.
- **LLM scene classification from field text:** more inference per session and requires reading field content, which 009 keeps behind a separate consent gate.
- **Remotely updated tables:** would refresh the defaults without a release, but adds a network dependency to a local-first feature; it needs its own decision.

## References

- [009 Scene selection](../../specs/009-contextual-rewrite.md#scene-selection), [Chatterfly research](../research/chatterfly.md), [scene benchmark methodology](../research/live-rewrite-benchmark.md#scene-selection).
