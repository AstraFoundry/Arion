# ADR 002: Local inference and self-hosted encrypted memory

Status: Accepted product boundary; crypto/model/provider selection pending.

## Context

The user requires local inference for privacy/cost, realtime correction, memory sync and a local Jev-style candidate scorer. Hosted Jev clients do not satisfy offline inference. Streaming ASR alternatives and context quality constrain any ranker.

## Decision

Capture/ASR/ranking/correction remain local after explicit model provisioning. Evaluate sherpa-onnx streaming and Laya Multilingual Choice first; production checkpoints are not selected. Rank only existing candidates and keep drafts separate from target commits. Synchronize approved vocabulary/correction/preferences as client-encrypted records through a self-hosted relay. Audio and temporary context do not enter the sync outbox.

## Consequences

Native model/typed-head export parity, license provenance and simultaneous worker latency are release gates. Relay storage cannot resolve plaintext semantic conflicts; clients review sibling revisions. Key enrollment/recovery/revocation needs a vetted implementation ADR before real data exchange. Server unavailability must not block dictation.
