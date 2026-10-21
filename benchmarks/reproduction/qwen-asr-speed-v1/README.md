# Qwen ASR size and quantization diagnostic

This follow-up uses the exact [ASR v1 development audio](../../corpora/asr-v1/corpus.json). It compares native CPU 0.6B/1.7B INT8 finalizers and separate Apple GPU MLX 1.7B 8-bit, 1.7B 4-bit and 0.6B 4-bit recognition. The exposed development set is not a new release test.

Native inference uses the unchanged production coordinator and the same pinned driver binary as the [previous native comparison](../qwen17-accessibility-v1/README.md). Replay is real time, X-ASR streaming is fixed, four CPU threads, no correction/ranking/memory. The original 1.7B measurement was already INT8. Both native arms are rerun here. The native process retains an earlier model in its cumulative RSS high-water mark; no isolated model RAM claim is made.

MLX runs use separate sequential processes after native inference finishes. Each model warms on the two train fixtures and recognizes all 118 dev clips with seed 11, greedy generation, empty context and a 128-token ceiling. Input objects contain only audio and case identity; reference/language labels are used only by the scorer. Each local payload is checked against revision-pinned Hugging Face metadata. Runtime Python/native source hashes, hardware/power state and package versions are retained per run. Model weights and environments remain private.

MLX latency is synchronized, warm **engine compute**, with audio already loaded. It excludes streaming, coordinator, microphone, UI rendering and insertion, and cannot establish product stop-to-review latency. Comparing native INT8 with MLX changes runtime/export/provider as well as precision. MLX 8-bit versus 4-bit holds the runtime and base model size fixed, but source-checkpoint floating-point parity is unmeasured. Community quantization labels do not imply that every acoustic tensor is low precision.

## Run and replay

Provision the exact revisions from the archived `*-metadata.json` files into a separate private directory, checking LFS SHA-256 and Git blob identities. The original provisioning source is retained. Install an isolated MLX Audio environment matching the recorded package versions and source hashes. No model downloads occur during inference.

Use the existing native runner with `native-config.json` and a newly resolved private driver manifest matching `native-driver.json`. The manifest pins executable, native libraries, model payloads and catalog. Its original absolute paths are historical machine identities, not portable installation paths. Select a new output directory:

```sh
pnpm bench:run --config benchmarks/reproduction/qwen-asr-speed-v1/native-config.json \
  --corpus benchmarks/corpora/asr-v1/corpus.json --driver /absolute/private/driver.json \
  --out /absolute/private/new-run/native-paired
```

Run `mlx-run.py --model /absolute/local/model --metadata <archived-metadata> --corpus benchmarks/corpora/asr-v1/corpus.json --out /absolute/private/new-run/<arm>` separately for `mlx17-8bit`, `mlx17-4bit`, and `mlx06-4bit`, using `HF_HUB_OFFLINE=1`. Existing output directories fail. Run one inference process at a time.

The model-free report reads archived observations, verifies completion, audio identities, unique complete pairing and result/warmup hashes, and reuses the repository CER/WER/MER and quantile implementation. It does not need weights or MLX:

```sh
node benchmarks/reproduction/qwen-asr-speed-v1/report.mjs \
  benchmarks/corpora/asr-v1/corpus.json benchmarks/runs/qwen-asr-speed-v1 \
  /absolute/private/new-summary.json
pnpm bench:report --results benchmarks/runs/qwen-asr-speed-v1/native-paired
```

The first native command was rejected before inference because a one-variant configuration is not an ablation. Its failure log remains archived; the completed run uses both model sizes. Preparation downloaded weights while the native CPU run was active; no second inference job ran concurrently. These are shared-machine measurements without a controlled energy/thermal study.

Verify the immutable archive and shared corpus/scorer identities with `python3 benchmarks/reproduction/qwen-asr-speed-v1/verify.py`. Four retained negative checks reject modified text, removed rows with a rebound hash, altered audio identity with a rebound hash, and incomplete status.
