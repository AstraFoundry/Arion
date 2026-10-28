# 020: Native Qwen3-ASR workers

The macOS Intel support commitment below is superseded by [ADR 021](021-apple-silicon-only-macos.md). Its runtime and evaluation records remain historical evidence; future macOS releases support Apple silicon only.

## Decision

Use Qwen3-ASR 1.7B as the recognition model. Apple silicon uses the pinned Swift MLX implementation with the community affine 8-bit checkpoint. Windows x64 and macOS Intel use the separately converted speech.cpp Q8_0 checkpoint. The quantization formats and implementations differ; quality and latency evidence does not transfer between them.

The application packages native executables, shader/runtime resources and dependency notices. It downloads only the platform's pinned model files, outside the application bundle. No Python interpreter, ONNX Runtime, sherpa-onnx, X-ASR model or old inference fallback is shipped. Historical benchmark archives retain their original implementations and results.

Both workers use private stdio, protocol 3, mono PCM16 at 16 kHz, resident weights and cancellable requests. The Go adapter keeps capture independent of inference, requests chunked previews after one second of new audio, and closes stable chunks after 800 ms below -42 dBFS RMS. It cuts in the middle of the pause without discarding audio. A continuous segment is bounded at 20 seconds using the existing quiet-point splitter; stop flushes every uncommitted sample, split at 25 seconds if needed. These bounded batch previews are not causal streaming ASR. The first-partial target remains an open performance gate.

Model loading uses a private temporary directory containing links to verified catalog files only. Swift derives tokenizer.json there; cached tokenizer files and extra weight files cannot enter the loader. Upstream stdout is isolated from the protocol descriptor. Token-limit termination and worker failure are explicit errors, never successful truncated text.

Recognition receives a session snapshot of approved canonical hotwords. The Go adapter joins the selected spellings with spaces into each `transcribe.prompt`; aliases and replacement instructions do not enter recognition context. It sends an explicit empty string when no terms are selected, so a resident worker never inherits a previous request's context. This is separate from post-recognition correction and rewriting.

Protocol 3 supports this additive request option without a version bump. The Swift worker advertises `capabilities: ["prompt"]` in `ready` (`arion-mlx-2`). The pinned speech.cpp worker already advertises its supported options in `ready.model.options`; recognition requires an entry with `name: "prompt"`, `type: "string"` and `steers: true`. A nonempty context sent to a worker without its corresponding capability fails before audio is submitted; an empty-context request remains compatible with older workers.

In the pinned MLX implementation, `Qwen3ASRModel.generateStream(context:)` places the supplied context in the prompt's system turn. speech.cpp maps its per-request `prompt` option through `SPEECH_OPT_PROMPT` to the same Qwen recognition context. These are native offline paths, with no additional dependency or network inference. The Go selection budget is distinct from the model limit: the Swift worker tokenizes the complete prompt including actual audio tokens and reserves 128 generation tokens against the loaded `textConfig.maxPositionEmbeddings`; speech.cpp checks complete prompt/audio/generation positions against its decoder configuration. Neither worker silently truncates an oversized prompt. Context refusal uses the existing terminal error envelope.

## Adoption and acceptance

The user explicitly adopted MLX 1.7B 8-bit and retired the old stack. A catalog bundle marked `adopted` is the default on its declared platform even before it has production pipeline measurements. This supersedes ADR 004 and MOD-04 for these two native bundles only. It does not imply latency, quality, Windows or minimum-OS acceptance. Measured automatic ranking remains available for other catalogs. Known retired model overrides migrate to this platform default; unknown IDs remain explicit errors. Existing model caches are retained until explicit eviction.

Completed native-worker and coordinator evaluations are recorded separately. Python MLX compute results are historical evidence, not native-package acceptance. Runtime replacement requires new evidence, including cancellation, stale-result rejection, tail preservation, offline loading and relocation.

The [native prompt smoke](../../benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/README.md) retains the actual Apple Silicon build, failed sandbox launch, subsequent native capability/recognition checks, empty-context reuse and oversized-context refusal. It is protocol evidence using one existing synthetic development fixture, not hotword accuracy or latency acceptance. The speech.cpp option mapping and capability declaration were checked against pinned source; native Windows hotword inference remains unverified by this check.

## Packaging

- Apple silicon: the Xcode 26.4.1 application / Swift 6.3.1 (`xcode-select` must point at `Xcode.app`, not Command Line Tools), macOS deployment target 14.0, pinned Package.resolved, arion-asr, colocated mlx.metallib, SwiftPM bundles and the Swift compatibility-library closure. Sign nested code before the outer app and notarize through the normal build command.
- macOS Intel: pinned speech.cpp source and ggml submodule, CPU/Accelerate executable, generic CPU flags and macOS 14.0 deployment target. Cross-build inspection is not native-host acceptance.
- Windows: source-built CPU executable with the static MSVC runtime plus a digest-pinned Vulkan executable. Select Vulkan only when the system Vulkan loader is available. Failure of a selected runtime is surfaced. Vulkan drivers are installed by the GPU vendor. No CUDA or DirectML support is claimed for this package.
- Weights stay in the verified model store. Package verification rejects obsolete runtime files and verifies architecture, minimum OS, notices and signatures.

Windows publisher signing, actual Windows inference/driver acceptance, actual Intel inference and clean macOS 14 execution remain separate distribution gates. GitHub Actions secrets cannot be retrieved as plaintext; CI can use the configured update key directly, while local update signing requires the original private-key backup.
