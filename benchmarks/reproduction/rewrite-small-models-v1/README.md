# Latest small-model rewrite screen

This diagnostic follows the user's explicit constraints: no further4B inference and no older Qwen generation tests. Qwen3.5-0.8B and Qwen3.5-2B use text-only GGUF bundles with752,393,024 and1,881,825,088 stored tensor elements respectively. No vision encoder or multimodal projector is loaded. Upstream multimodal parameter totals are retained separately and must not be confused with the text-only bundle.

The revision-pinned community Q4_K_M exports identify their official Apache-2.0 upstream models. `model-provenance.json` records export/upstream revisions, expected LFS SHA-256 and byte sizes. `gguf_info.py` reads metadata and sums tensor dimensions; it does not execute model code. Official runtime release metadata retains the macOS arm64 asset digest. Weights and executable libraries stay in `.workspace/`.

Both models receive identical system text, demonstrations, labeled input serialization, CPU4 threads/GPU0, context4096, generation384, temperature0 and thinking disabled. Eight exposed probes and sixteen previously frozen but ungenerated probes are defined before inference. These are synthetic diagnostic inputs, not independent natural ASR test data. The four known-answer repair positives do not independently establish causal OCR benefit; matching empty-context counterfactuals are not part of this screen. No gold intended text or ratings enter the request. Full request messages are recorded per row and checked during replay.

No validator, retry or cleanup fallback filters generated bodies. This is a raw generation diagnostic. The protocol's generic phrase “preserved cleanup” does not imply a separate cleanup operation: these already-cleaned text fixtures are sent directly, and no cleanup inference is measured. Every response, finish reason and review remains visible, including failures.

The runtime and cache policy differ from historical ONNX experiments. Prompt caching is disabled (`cache_n=0` recorded); startup is separate, and the first request includes first-use effects after `--no-warmup`. Request timings are not ASR-to-insertion latency, and sampled RSS is not peak memory or weak-machine acceptance. No causal model-only speed comparison with historical prefix-cached ONNX is valid.

Replay, tests and archive verification use the unchanged portable wrapper:

```sh
node scripts/rewrite-evidence-command.mjs rewrite-small-models-v1 report --out .workspace/small-model-report-replay
node scripts/rewrite-evidence-command.mjs rewrite-small-models-v1 test_report
node scripts/rewrite-evidence-command.mjs rewrite-small-models-v1 verify
```

Inference uses `run.py --model <pinned-gguf> --runtime <pinned-llama-server> --out <new-directory>`. Frozen hashes are checked before loading. The server binds only loopback and terminates after the run. Models over2,000,000,000 tensor elements are rejected. Production integration and release acceptance remain separate gates.

Review outcome:0.8B retains meaning22/24 and useful scene organization20/24;2B retains meaning23/24 and useful organization23/24. Both match all four known-answer repair positives. Both guess the ambiguous function;0.8B additionally replaces one dictated message with the OCR name list.2B groups both owner/task/deadline probes, while0.8B groups neither. Four2B outputs carry a redundant single bullet (one also has the substantive ambiguity failure). These cosmetic bullets are visible issues but do not alone make a work message unusable; they never count as meaning errors. They are not hidden by a validator.

CPU cache-off request p95 is2029/4466ms and maximum sampled RSS is1.65/3.16GB respectively. Low-end-machine acceptance is not established. Continue small-model prompt/cache research; neither candidate is activated in production.
