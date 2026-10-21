# 006 Model selection and evaluation

The native Qwen3-ASR bundles are explicitly adopted platform defaults under [ADR 020](../docs/decisions/020-native-qwen-asr.md); adoption does not assert that evaluation or distribution gates passed. For other catalogs, the default recognition bundle is selected automatically from measured evaluations recorded in the catalog ([ADR 004](../docs/decisions/004-automatic-model-selection.md)); users do not have to choose a model and may override the choice among evaluated bundles. Runtime licenses do not license model weights, tokenizer files, training data or converted artifacts.

## Provisioning

MOD-01: Every approved artifact needs ID, purpose, runtime/provider compatibility, immutable upstream revision, artifact SHA-256/size, source URL, model and runtime license references, supported languages/platforms, audio format if applicable, tokenizer/scoring configuration, quantization and conversion provenance. See [models](../models/README.md).

MOD-02: Download to a temporary file with bounded size, verify checksum and publish atomically. Corrupt/unapproved artifacts never load. Keep the prior verified version until replacement succeeds. Signature/trust root design for remotely updated catalogs is a separate release gate.

MOD-03: Cache eviction is explicit and never removes an actively used model. Missing/offline provisioning is surfaced to the user. Version change requires benchmark comparison and rollback path.

## Evaluation gates

| Dimension | Evidence required |
| --- | --- |
| Recognition | Mandarin CER, English WER, mixed-language token errors, names/numbers/negation preservation |
| Streaming | First partial p50/p95, draft churn, speech-end-to-final p50/p95, real-time factor |
| Correction | Harmful rewrite rate vs raw ASR and user-preferred correction rate |
| Candidate ranking | Candidate oracle recall, source top-1 vs ranker top-1/MRR, homophones, user override rate |
| Resource cost | Cold/warm load, RAM, CPU/GPU, energy, sustained capture contention |
| Privacy | Offline/content-network check, log/storage canary checks |
| Compatibility | Native architecture/package/model ABI on each supported OS |
| License | Checkpoint-specific permission, converted artifact attribution and distribution gate |

Provisional engineering budgets: warm first partial p95 ≤ 500 ms from first speech sample; speech-end-to-review p95 ≤ 1,000 ms; interactive candidate ranking p95 ≤ 50 ms for the declared candidate count; real-time factor < 1 under sustained load. These are unmeasured targets. Record hardware, candidate count/token lengths, recognizer configuration and cold-start separately. Budget violations require revising the design or explicitly reporting a blocked gate, never omitting slow samples.

Required benchmark coverage: macOS Apple silicon; Windows x64 CPU; Windows optional GPU separately. Intel macOS is outside future support and release gates ([ADR 021](../docs/decisions/021-apple-silicon-only-macos.md)). Windows arm64 is P2 until MyGo and chosen model providers ship/test compatible artifacts. Minimum OS and RAM promises require real measurements.

MOD-04: Except for the explicit native adoption in ADR 020, the default is the evaluated bundle with the lowest mean of Mandarin CER, English WER and mixed MER among bundles whose evaluation on the same hardware class passes the provisional latency and real-time-factor budgets with at least 20 observations. Without a run on the hardware class, the most accurate bundle measured elsewhere is chosen and the UI states that its speed is unmeasured. Unevaluated bundles are never the default.

MOD-A01: corrupt artifact rejected; MOD-A02: no license/provenance → no distribution; MOD-A03: concurrent ranker/ASR benchmark reports contention; MOD-A04: model replacement preserves rollback; MOD-A05: offline repeated inference works; MOD-A06: candidate ranker output/reference scoring parity is measured.
