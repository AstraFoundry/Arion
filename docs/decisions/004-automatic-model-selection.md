# ADR 004: The default model is chosen by measured evaluation

Status: Accepted for measured ranking; explicit native Qwen3-ASR adoption is governed by [ADR 020](020-native-qwen-asr.md).

## Context

The product must load the best recognition bundle by default instead of asking users to pick models. [006 Models](../../specs/006-models.md) requires provenance, license records and measured gates; research candidates differ in latency, accuracy and size.

## Decision

`models/catalog.json` pins every artifact (URL, SHA-256, size, member digests, licenses, file roles) and lists **bundles**: a streaming preview model plus an optional acoustic finalizer. Each bundle carries content-free evaluation summaries copied from completed benchmark reports (run ID, hardware class, per-language error rates, warm stop-to-review p95, real-time factor, observation count).

The domain rule `SelectDefaultModel` picks, for the running platform tag (`darwin-arm64`, `windows-amd64`, …):

1. Bundles whose evaluation on this hardware class passes the gates (stop-to-review p95 ≤ 1,000 ms, RTF < 1, ≥ 20 observations), ranked by the unweighted mean of Mandarin CER, English WER and mixed MER; ties prefer lower latency, then smaller downloads.
2. Only when no run exists for this hardware class: the most accurate bundle measured elsewhere, flagged `LatencyMeasured=false`, which the UI states.

Unevaluated bundles are never chosen. A user override is stored as an explicit preference and can be cleared ("Use recommended").

## Consequences

Changing the default requires a new benchmark report, not a code change. Evaluations from development hardware drive selection for other hardware classes until those are measured; the UI says so. Accuracy rankings assume identical weights and runtime produce the same transcripts across hardware.

## Alternatives considered

Hard-coding a model ID (no evidence trail), or letting the user choose at first run (the product requirement rejects this).
