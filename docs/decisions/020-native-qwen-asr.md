# 020: Native Qwen3-ASR workers

The macOS Intel support commitment below is superseded by [ADR 021](021-apple-silicon-only-macos.md). Its runtime and evaluation records remain historical evidence; future macOS releases support Apple silicon only.

## Decision

Use Qwen3-ASR 1.7B as the recognition model. Supported platforms use the pinned speech.cpp implementation with the Q8_0 GGUF checkpoint. Apple silicon enables Metal and Accelerate; Windows x64 uses the source-built CPU worker or the separately verified Vulkan worker. The former Swift MLX affine 8-bit bundle is retained as historical provenance only and is not selectable on a supported platform; its quality and latency evidence does not transfer to speech.cpp.

The application packages native executables, shader/runtime resources and dependency notices. It downloads only the platform's pinned model files, outside the application bundle. No Python interpreter, ONNX Runtime, sherpa-onnx, X-ASR model or old inference fallback is shipped. Historical benchmark archives retain their original implementations and results.

Both workers use private stdio, protocol 3, mono PCM16 at 16 kHz, resident weights and cancellable requests. The Go adapter keeps capture independent of inference, requests chunked previews after one second of new audio, and closes stable chunks after 800 ms below -42 dBFS RMS. It cuts in the middle of the pause without discarding audio. A continuous segment is bounded at 20 seconds using the existing quiet-point splitter; stop flushes every uncommitted sample, split at 25 seconds if needed. These bounded batch previews are not causal streaming ASR. The first-partial target remains an open performance gate.

Model loading uses a private temporary directory containing links to verified catalog files only; cached tokenizer files and extra weight files cannot enter the loader. Upstream stdout is isolated from the protocol descriptor. Token-limit termination and worker failure are explicit errors, never successful truncated text.

Recognition receives a session snapshot of approved canonical hotwords. The Go adapter joins the selected spellings with spaces into each `transcribe.prompt`; aliases and replacement instructions do not enter recognition context. It sends an explicit empty string when no terms are selected, so a resident worker never inherits a previous request's context. This is separate from post-recognition correction and rewriting.

Quiet audio is not a recognition request. Before a prompt is sent, each window is checked with the same -42 dBFS floor used to close a segment, in 20 ms spans. A window with no span at or above that floor returns empty text and does not build a worker request, so Qwen3 cannot read the system-turn hotwords back as a transcript during the second before speech, during a quiet tail, or on a fully silent recording. Invalid terms and non-finite samples still fail on quiet audio. A window that contains speech, including breath that crosses the floor, still receives the full session snapshot on the first decode. If that transcript is an ordered copy of two or more contiguous snapshot terms — the whole transcript, a leading run, or a trailing run, ignoring punctuation and a leading `气声` label — it is not published. The same samples are decoded again with an empty prompt, and only that result becomes the hypothesis. One matching term stays. The floor is not raised to hide these copies.

Protocol 3 supports this additive request option without a version bump. The pinned speech.cpp worker advertises its supported options in `ready.model.options`; recognition requires an entry with `name: "prompt"`, `type: "string"` and `steers: true`. A nonempty context sent to a worker without its corresponding capability fails before audio is submitted; an empty-context request remains compatible with older workers.

speech.cpp maps its per-request `prompt` option through `SPEECH_OPT_PROMPT` to the Qwen recognition context. This is a native offline path, with no additional dependency or network inference. The Go selection budget is distinct from the model limit: speech.cpp checks complete prompt/audio/generation positions against its decoder configuration and does not silently truncate an oversized prompt. Context refusal uses the existing terminal error envelope.

## Adoption and acceptance

The user explicitly adopted the speech.cpp Q8_0 bundle and retired the old stack, including the MLX platform default. A catalog bundle marked `adopted` is the default on its declared platform even before it has production pipeline measurements. This supersedes ADR 004 and MOD-04 for these native bundles only. It does not imply latency, quality, Windows or minimum-OS acceptance. Measured automatic ranking remains available for other catalogs. Known retired model overrides migrate to this platform default; unknown IDs remain explicit errors. Existing model caches are retained until explicit eviction.

Completed native-worker and coordinator evaluations are recorded separately. Python MLX compute results are historical evidence, not native-package acceptance. Runtime replacement requires new evidence, including cancellation, stale-result rejection, tail preservation, offline loading and relocation.

The [native prompt smoke](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/README.md`) retains the actual Apple Silicon build, failed sandbox launch, subsequent native capability/recognition checks, empty-context reuse and oversized-context refusal. It is protocol evidence using one existing synthetic development fixture, not hotword accuracy or latency acceptance. The speech.cpp option mapping and capability declaration were checked against pinned source; native Windows hotword inference remains unverified by this check.

## Packaging

- Apple silicon: pinned speech.cpp source and ggml submodule, Metal/Accelerate enabled, arm64 executable and macOS 14.0 deployment target. Command Line Tools with the macOS SDK and CMake are sufficient; the Swift MLX worker, full Xcode 26 and a separately downloaded Metal Toolchain are not required.
- macOS Intel: pinned speech.cpp source and ggml submodule, CPU/Accelerate executable, generic CPU flags and macOS 14.0 deployment target. Cross-build inspection is not native-host acceptance.
- Windows: source-built CPU executable with the static MSVC runtime plus a digest-pinned Vulkan executable. Select Vulkan only when the system Vulkan loader is available. Failure of a selected runtime is surfaced. Vulkan drivers are installed by the GPU vendor. No CUDA or DirectML support is claimed for this package.
- Weights stay in the verified model store. Package verification rejects obsolete runtime files and verifies architecture, minimum OS, notices and signatures.

Windows publisher signing, actual Windows inference/driver acceptance, actual Intel inference and clean macOS 14 execution remain separate distribution gates. GitHub Actions secrets cannot be retrieved as plaintext; CI can use the configured update key directly, while local update signing requires the original private-key backup.
