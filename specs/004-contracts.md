# 004 Portable contracts

Version 1 uses transport-neutral Go DTOs in [core contracts](../packages/core/contracts/contracts.go). Future IPC/HTTP encodings must preserve these meanings and add documented fixtures before they ship. This foundation does not claim a ready HTTP or worker protocol.

## Envelope and concurrency

Events carry `version`, `session_id`, `generation`, increasing `sequence`, `kind`, and optional `revision` for draft/candidate events. The consuming application validates supported version/kind and rejects stale session/generation/sequence/revision. Monotonic clocks measure latency; wall clocks never decide draft or sync conflict ordering.

Kinds are `session_state`, `audio_discontinuity`, `audio_level`, `draft_replaced`, `candidates_ranked`, `correction_ready`, `commit_result`, and `error`. Event-specific DTOs remain typed; do not use unvalidated arbitrary maps as the application API. `audio_level` carries only a 0–1 meter value. `correction_ready` carries the raw recognizer text and the corrected text separately. An `error` may carry a content-free diagnostic category (`detail`, e.g. `microphone` or `accessibility` for `PERMISSION_DENIED`).

Session states: `idle → preparing → capturing → finalizing → reviewing → committing → completed`. Active states may become `cancelled` or `failed`; terminal states cannot revive. New dictation creates a new session. IME draft updates during capture do not constitute final target commit.

## Ports

[Domain-owned ports](../packages/core/domain/ports/ports.go) define recognizer, corrector, ranker and input-target operations. They accept `context.Context`, carry session/revision identity and return explicit errors. The concrete adapter defines microphone/VAD and model lifecycle while preserving the core contract.

- Target capabilities distinguish context read, single insertion, owned composition and candidates. Unsupported is a declared capability, not an inferred editor property. A `TargetSnapshot` taken at session start carries the opaque token, capabilities, a `sensitive` flag for known secure fields and the application identity for local policy only.
- `TargetToken` is opaque. OS-native handles and editor pointers must not appear in JSON or core DTOs.
- Text offsets in portable DTOs count Unicode scalar values in the exact source text. Convert explicitly to UTF-16, byte offsets or host ranges inside the OS adapter; include source revision to prevent index drift.
- Candidate IDs are producer-owned and unique within a revision. Ranking returns IDs and scores only; selected text remains the exact producer text.
- `CommitResult` separates `applied`, `not_applied`, and `unknown`. Never retry an `unknown` insertion automatically.

## Error taxonomy

Stable codes include `PERMISSION_DENIED`, `MODEL_MISSING`, `MODEL_UNSUPPORTED`, `AUDIO_DISCONTINUITY`, `TARGET_CHANGED`, `TARGET_UNSUPPORTED`, `STALE_REVISION`, `CANCELLED`, `INSERTION_UNCERTAIN`, `SYNC_UNAUTHORIZED`, `SYNC_CONFLICT`, and `CONTRACT_UNSUPPORTED`, plus `DEVICE_UNAVAILABLE`, `INVALID_TRANSITION`, `SESSION_BUSY`, `NO_SPEECH`, `ARTIFACT_UNVERIFIED`, `PROVISIONING_FAILED`, `SECURE_FIELD_BLOCKED` and `EVALUATION_UNAVAILABLE`.

Adapters may attach diagnostic categories, not transcript/context/token bodies. UI maps codes to centralized copy. No silent catches or success defaults. Future relay HTTP uses the [API conventions](../docs/backend/api-conventions.md).

## Acceptance

CTR-A01: invalid state transition rejected; CTR-A02: terminal session cannot resume; CTR-A03: rank result cannot add/drop/duplicate IDs; CTR-A04: stale revision/generation ignored; CTR-A05: UTF-16/rune conversion tested with emoji and CJK; CTR-A06: unknown insertion result cannot retry automatically.
