# Arion product

Arion is a voice input product with a native Go desktop shell. macOS and Windows are the initial implementation targets; mobile work must preserve portable contracts without creating mobile apps yet.

## Confirmed scope

- Local streaming speech recognition and live correction, initially evaluated on Mandarin, English, and mixed technical vocabulary.
- Configurable input modes: global shortcut dictation and system IME. IMK/TSF are separate OS adapters sharing the same core.
- Local ranking of actual candidate words/segments, preserving candidate identity and text.
- User-controlled personal vocabulary, correction rules, and contextual preferences.
- Self-hosted encrypted memory sync, with offline dictation independent of account or server availability.
- MyGo native Go UI; no web frontend is required.

Language coverage and latency are evaluation targets, not measured product claims. User-facing UI copy remains English under repository conventions.

## Explicit boundaries

Raw microphone audio and temporary foreign-window context stay on the device. Persisting dictation history is separately opt-in and is not in the initial sync scope. Candidate sorting must not introduce unseen text or overwrite committed content. A full conventional Pinyin keyboard, cloud recognition, voice chat, autonomous commands, and mobile keyboard extensions require their own scope decision.

Jev is TypeSafe’s System One typed decision model. Laya Multilingual is the proposed independent local Choice implementation; exact checkpoint, native conversion and candidate accuracy require evaluation.

## Stack and decision status

| Area | Choice | Status |
| --- | --- | --- |
| Desktop UI | MyGo v0.4.6 native, Go 1.27.1 | Selected by user; pinned shell dependency |
| Shared domain/application | Portable Go module | Foundation |
| macOS | CoreAudio/AVFoundation capture; AX context; IMK composition | Adapter design, implementation pending |
| Windows | WASAPI capture; UI Automation context; TSF composition | Adapter design, implementation pending |
| ASR | sherpa-onnx streaming evaluation adapter | Proposed; production weights require benchmark/license gates |
| Correction | Conservative rules, optional local text model | Evaluation required |
| Candidate ranking | Laya Multilingual local Choice evaluation adapter | Proposed; production checkpoint/export requires evaluation |
| Local storage | SQLite with encrypted sensitive values and OS-kept keys | Proposed; concrete driver/migrations pending |
| Sync service | Self-hosted Go relay; ciphertext records | Selected deployment priority; protocol specified |
| Cryptography | Vetted AEAD/key lifecycle implementation | Provider decision pending; do not invent crypto |

See [specs](../../specs/README.md), [research](../research/README.md), [repository](repository.md), and [agent handoff](agent-handoff.md).
