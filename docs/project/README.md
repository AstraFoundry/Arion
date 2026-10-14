# Arion product

Arion is a voice input product with a native Go desktop shell. macOS and Windows are the initial implementation targets; mobile work must preserve portable contracts without creating mobile apps yet.

## Confirmed scope

- Local streaming speech recognition and live correction, initially evaluated on Mandarin, English, and mixed technical vocabulary.
- Configurable input modes: global shortcut dictation and system IME. IMK/TSF are separate OS adapters sharing the same core.
- Local ranking of actual candidate words/segments, preserving candidate identity and text.
- User-controlled personal vocabulary, correction rules, and contextual preferences.
- Self-hosted encrypted memory sync, with offline dictation independent of account or server availability.
- MyGo native Go UI for the desktop app; the only web code is the static marketing site in `apps/site` ([ADR 016](../decisions/016-marketing-site.md)).
- The recognition model is not a user decision: the best evaluated bundle downloads and loads automatically; another evaluated bundle can be chosen explicitly.

Language coverage and latency are evaluation targets, not measured product claims. The interface defaults to English and supports Simplified Chinese through a persisted language picker; interface language is independent of recognition language ([i18n](../frontend/i18n.md)).

## Explicit boundaries

Raw microphone audio and temporary foreign-window context stay on the device. Persisting dictation history is separately opt-in and is not in the initial sync scope. Candidate sorting must not introduce unseen text or overwrite committed content. A full conventional Pinyin keyboard, cloud recognition, voice chat, autonomous commands, and mobile keyboard extensions require their own scope decision.

Jev is TypeSafe’s System One typed decision model. Laya Multilingual is the proposed independent local Choice implementation; exact checkpoint, native conversion and candidate accuracy require evaluation.

## Stack and decision status

| Area | Choice | Status |
| --- | --- | --- |
| Desktop UI | MyGo v0.4.6 native, Go 1.27.1; Sunbeam design system with coss anatomy in Go ([ADR 007](../decisions/007-sunbeam-design-system.md)) | Implemented: menu bar panel, overlay pill, settings, onboarding |
| Shared domain/application | Portable Go module | Implemented: session aggregate, coordinator, correction rules, model selection |
| macOS | miniaudio/CoreAudio capture; AX target and insertion; InputMethodKit host | Implemented and tested on macOS arm64 |
| Windows | miniaudio/WASAPI capture; SendInput insertion; DPAPI | Code compiled for windows/amd64; not run on Windows; TSF pending |
| ASR | sherpa-onnx 1.13.8 via Go/cgo; streaming preview plus acoustic finalizer | Default bundle chosen by measured evaluation ([ADR 004](../decisions/004-automatic-model-selection.md)) |
| Correction | Conservative rules (`arion-rules-v1`) and approved vocabulary, including entries learned from the user's corrections | Implemented ([ADR 014](../decisions/014-edit-learning.md)); local LLM correction not adopted |
| Candidate ranking | Laya Multilingual local Choice evaluation adapter | Not implemented: recognizers expose one hypothesis |
| Local storage | Preferences JSON; AES-256-GCM vocabulary with OS-kept key | Implemented ([ADR 005](../decisions/005-local-encryption.md)); SQLite/outbox pending |
| Sync service | Self-hosted Go relay; ciphertext records | Protocol specified; not implemented |
| Marketing site | Next.js static export, React 19, Tailwind v4, next-intl en/zh/ja; Cloudflare Workers static assets ([ADR 016](../decisions/016-marketing-site.md)) | Implemented in `apps/site`; vendored Sunbeam/beUI until `@astrafoundry/design` is published |
| Cryptography | Go standard library AES-GCM; Keychain/DPAPI keys | Local values only; key exchange for sync pending |
| Account sign-in | Public OIDC client `arion-desktop`, loopback PKCE, refresh token in Keychain or Credential Manager | Service, onboarding step and Settings › Account implemented ([account sign-in](account-signin.md)); device enrollment open |

See [specs](../../specs/README.md), [research](../research/README.md), [repository](repository.md), [account sign-in](account-signin.md), and [agent handoff](agent-handoff.md).
