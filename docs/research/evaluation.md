# Evaluation procedure

Model-comparison runs measure Arion's production pipeline on device (see [benchmark results](../../benchmarks/README.md)); their aggregates select the default bundle. Budgets in [006 Models](../../specs/006-models.md) remain provisional engineering gates.

## Record each run

Hardware/OS/architecture, RAM and power mode; runtime/engine commit and build flags; immutable checkpoint/tokenizer/conversion hashes; audio sample rate and capture/chunk/endpoint settings; candidate count/length and approved memory scope; warm/cold state; concurrent workers; fixture license/consent; raw metrics and rejected samples with reasons.

Use consented or synthetic fixtures containing Mandarin, English, mixed technical names, homophones, numbers, URLs, negation, disfluency, silence/noise and microphone discontinuities. Keep private corpora and recorded audio out of Git. Pin fixtures/hashes when sharing results.

## Measure separate stages

1. Capture queue age/discontinuities and resampling cost.
2. VAD speech onset/tail loss and false endpoints.
3. Streaming ASR first partial, revision churn, final CER/WER and real-time factor.
4. Candidate producer oracle recall before measuring ranking accuracy.
5. Source order + vocabulary boost vs Laya Choice and reranker top-1/MRR; candidate order sensitivity and typed-head reference parity.
6. Correction harmful edits and intent preservation, not just fluent output.
7. Stop-to-review and review-to-commit latency; no mistaken duplicate commit.
8. Resource contention with simultaneous ASR/ranker, thermal behavior, CPU/GPU memory and load cost.
9. IMK/TSF lifecycle/range safety on actual applications and OS versions.
10. Network disabled, content canaries in storage/logs and encrypted relay snapshot inspection.

Report p50/p95 rather than fastest examples. Choice confidence is conditional on its option set; do not compare scores across differently pruned candidate groups without calibration. Use held-out calibration and independently report candidate selection accuracy.

## Competitor follow-up boundaries

Installed-package evidence cannot prove latency, server models, traffic or successful insertion. If later runtime investigation is needed, use a controlled synthetic text editor/utterance, observe only the selected target, avoid personal logs/credentials and preserve exact app version/evidence. Native decompilation remains unperformed without a configured engine. Never describe endpoints/symbols as observed uploads.

## Executable automation

The [evaluation workspace](../../tools/evaluation/README.md) implements plans, corpus/hash validation, a live driver protocol, metric reports and paired bootstrap. See [paper methodology](benchmark-methodology.md), [streaming dataset research](streaming-datasets.md) and [008 Evaluation](../../specs/008-evaluation.md). `make ablation-plan` and `make stream-plan` perform no inference; `make bench-run`/`make stream-bench` require provisioned audio/models and a reviewed driver.
