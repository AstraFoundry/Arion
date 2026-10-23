# Decisions

This section records architecture decision records (ADRs). Each ADR explains why a significant decision was made and what alternatives were considered.

## When to write an ADR

Write an ADR when the decision:

- Is hard to reverse.
- Affects multiple layers or teams.
- Has non-obvious trade-offs.

## Existing decisions

- [001: Desktop native boundaries](001-desktop-native-boundaries.md).
- [002: Local pipeline and self-hosted memory](002-local-pipeline-and-memory.md).
- [003: Shortcut-mode insertion routes](003-insertion-routes.md).
- [004: The default model is chosen by measured evaluation](004-automatic-model-selection.md).
- [005: Local encryption of approved vocabulary](005-local-encryption.md).
- [006: macOS input-method host and local IPC](006-macos-input-method-host.md).
- [007: Sunbeam design system and coss anatomy in Go](007-sunbeam-design-system.md).
- [008: Contextual rewrite evaluation](008-contextual-rewrite-evaluation.md) — evaluation boundary; its review-before-insert rule is superseded by ADR 011.
- [009: AGPL-3.0 license](009-agpl-license.md) — open source with a commercial hosted edition kept possible.
- [010: In-app updates from GitHub releases](010-in-app-updates.md) — signed MyGo updates, automatic checks on by default; the update source is superseded by ADR 019.
- [011: Live local rewriting](011-live-local-rewriting.md) — inserts one validated rewrite after release; the mid-speech text-model preview is superseded by ADR 023, and the runtime and templates by ADR 018.
- [012: Scene selection from the focused field, site and app](012-context-scene-selection.md) — local metadata picks email, work, prompt, search or chat; formatting follows ADR 018.
- [013: Rewrite context from the cursor, the screen and recent dictations](013-rewrite-context-on-device.md) — on by default and on-device only; the selector may see it, and the formatter does not.
- [014: Learn vocabulary from the user's corrections](014-edit-learning.md) — on-device edit tracking, one-shot learning with undo, one-pass recall and bounded prompt terms.
- [015: Hosted relay account authorization layer](015-hosted-relay-account-authorization.md) — `AccountGate` (`noop` self-host, `oidc` hosted) on top of device auth, `(issuer, subject)` tenants, security events; login never grants decryption.
- [016: Marketing site in `apps/site`](016-marketing-site.md) — static Next.js export with the shared design package; ADR 001's no-React rule covers the desktop app only.
- [017: Offline Pinyin keyboard in the macOS IMK host](017-offline-pinyin-keyboard.md) — local conversion, independent composition ownership and encrypted learning.
- [019: Update channels served by the Arion center](019-center-update-channels.md) — `stable`/`beta` manifests and downloads from `api.arion.astrafoundry.si`, staged by CI and published from the AstraFoundry admin; supersedes ADR 010's GitHub source.
- [018: Staged GGUF rewriting](018-staged-gguf-rewriting.md) — selected Qwen3.5-2B artifact, two-stage prompts and accepted delivery tradeoff.
- [020: Native Qwen3-ASR workers](020-native-qwen-asr.md) — native Swift MLX on Apple silicon and native Q8_0 on Intel/Windows, with explicit adoption and separate acceptance.
- [021: Support macOS on Apple silicon only](021-apple-silicon-only-macos.md) — retires Intel macOS product support, CI and future release artifacts; Windows x64 remains in scope.
- [022: Optional cloud polishing through Republic](022-cloud-polishing.md) — ChatGPT sign-in or one API key, final text and recalled terms only, no context; local stays the default.
- [023: Final-only polishing](023-final-only-polishing.md) — local and cloud rewriting both run once after the dictation key is released; segmentation stays, and no polished preview is shown during speech.
- [`adr-template.md`](adr-template.md) — template for new ADRs.

## Naming

Use a sequential number and a short kebab-case title:

```
decisions/
  001-use-uuid-for-ids.md
  002-choose-postgresql.md
```
