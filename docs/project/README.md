# Arion product

Arion is a voice input product with a native Go desktop shell. macOS on Apple silicon and Windows x64 are the supported implementation targets ([ADR 021](../decisions/021-apple-silicon-only-macos.md)); mobile work must preserve portable contracts without creating mobile apps yet.

## Confirmed scope

- Optional cloud polishing with the user's ChatGPT plan or API key, chosen explicitly; audio stays local, while enabled context can accompany cloud polishing ([ADR 022](../decisions/022-cloud-polishing.md)).
- Local streaming speech recognition and live correction, initially evaluated on Mandarin, English, and mixed technical vocabulary.
- Unified dictation with automatic delivery through the selected Arion input method or the external editor. IMK/TSF are separate OS adapters sharing the same core.
- Local ranking of actual candidate words/segments, preserving candidate identity and text.
- User-controlled personal vocabulary, correction rules, and contextual preferences.
- Encrypted memory sync: standard packages select AstraFoundry's center; source builds can select a self-hosted center through build-time environment configuration (SYNC-11, build injection implemented; desktop sync pending). Offline dictation is independent of account or server availability.
- MyGo native Go UI for the desktop app; the only web code is the static marketing site in `apps/site` ([ADR 016](../decisions/016-marketing-site.md)).
- The recognition model is not a user decision: the best evaluated bundle downloads and loads automatically; another evaluated bundle can be chosen explicitly.

Language coverage and latency are evaluation targets, not measured product claims. The interface defaults to English and supports Simplified Chinese through a persisted language picker; interface language is independent of recognition language ([i18n](../frontend/i18n.md)).

## Explicit boundaries

Raw microphone audio stays on the device. Temporary foreign-window context stays local unless the user selects cloud polishing with its source switch enabled ([ADR 024](../decisions/024-shared-polishing-pipeline.md)). Persisting dictation history is separately opt-in and is not in the initial sync scope. Candidate sorting must not introduce unseen text or overwrite committed content. Offline Simplified Chinese full-Pinyin typing in the macOS IMK host is now in scope ([ADR 017](../decisions/017-offline-pinyin-keyboard.md)); the native implementation and isolated engine/AppKit/storage checks are complete, while system input-source acceptance remains open. Cloud recognition, voice chat, autonomous commands, additional keyboard schemes and mobile keyboard extensions require their own scope decision.

Jev is TypeSafe’s System One typed decision model. Laya Multilingual is the proposed independent local Choice implementation; exact checkpoint, native conversion and candidate accuracy require evaluation.

## Stack and decision status

| Area | Choice | Status |
| --- | --- | --- |
| Desktop UI | MyGo v0.4.6 native, Go 1.27.1; Sunbeam design system with coss anatomy in Go ([ADR 007](../decisions/007-sunbeam-design-system.md)) | Implemented: menu bar panel, overlay pill, settings, onboarding |
| Shared domain/application | Portable Go module | Implemented: session aggregate, coordinator, correction rules, model selection |
| macOS | miniaudio/CoreAudio capture; AX target and insertion; offline Pinyin/voice InputMethodKit host | Native arm64 and Rosetta x86_64 runs each pass 36 engine/AppKit cases; Intel macOS is unsupported; clean-machine IMK acceptance remains open |
| Windows | miniaudio/WASAPI capture; SendInput insertion; DPAPI | Actual Windows x64 native, microphone, installed speech and package/update checks pass on one development host; [limits](../quality/windows-validation.md); TSF pending |
| ASR | Native Qwen3-ASR 1.7B workers: Swift MLX 8-bit on Apple silicon; speech.cpp Q8_0 CPU/Vulkan on Windows x64 | Explicit platform adoption; independent native evaluation and packaging gates ([ADR 020](../decisions/020-native-qwen-asr.md)) |
| Correction | Conservative rules (`arion-rules-v4`) and approved vocabulary, including entries learned from the user's corrections. Standalone hesitation is deleted by speech cleanup when polishing runs | Implemented ([ADR 014](../decisions/014-edit-learning.md)); optional Qwen3.5-2B staged local rewriting adopted on macOS Apple Silicon; known development errors accepted ([configuration](../operations/local-rewrite.md)) |
| Candidate ranking | Laya Multilingual local Choice evaluation adapter | Not implemented: recognizers expose one hypothesis |
| Local storage | Preferences JSON; AES-256-GCM vocabulary with OS-kept key | Implemented ([ADR 005](../decisions/005-local-encryption.md)); SQLite/outbox pending |
| Sync service | Self-hosted/hosted Go relay; ciphertext records | Relay implemented; desktop sync/outbox, pairing and key delivery pending |
| Marketing site | Next.js static export, React 19, Tailwind v4, next-intl en/zh/ja; Cloudflare Workers static assets ([ADR 016](../decisions/016-marketing-site.md)) | Implemented in `apps/site`; vendored Sunbeam/beUI until `@astrafoundry/design` is published |
| Cryptography | Go standard library AES-GCM; Keychain/DPAPI keys | Local values only; key exchange for sync pending |
| Cloud polishing | Republic providers: ChatGPT sign-in or one API key (OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter); shared staged polishing with enabled context | Optional, explicit engine choice ([ADR 022](../decisions/022-cloud-polishing.md), [cloud polishing](cloud-polishing.md)); real-service verification open |
| Account sign-in | Public OIDC client `arion-desktop`, loopback PKCE, refresh token in Keychain or Credential Manager | Service, onboarding step and Settings › Account implemented ([account sign-in](account-signin.md)); device enrollment open |

See [specs](../../specs/README.md), [research](../research/README.md), [repository](repository.md), [account sign-in](account-signin.md), [cloud polishing](cloud-polishing.md), and [agent handoff](agent-handoff.md).

[Central sync design proposal](central-sync-design.md) compares the recommended data allowlist with competitor evidence and specifies hosted deployment, client delivery, key lifecycle and release gaps. P1/P2 extensions remain proposals.
