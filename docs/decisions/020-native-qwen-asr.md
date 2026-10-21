# 020: Native Qwen3-ASR workers

## Decision

Use Qwen3-ASR 1.7B as the recognition model. Apple silicon uses the pinned Swift MLX implementation with the community affine 8-bit checkpoint. Windows x64 and macOS Intel use the separately converted speech.cpp Q8_0 checkpoint. The quantization formats and implementations differ; quality and latency evidence does not transfer between them.

The application packages native executables, shader/runtime resources and dependency notices. It downloads only the platform's pinned model files, outside the application bundle. No Python interpreter, ONNX Runtime, sherpa-onnx, X-ASR model or old inference fallback is shipped. Historical benchmark archives retain their original implementations and results.

Both workers use private stdio, protocol 3, mono PCM16 at 16 kHz, resident weights and cancellable requests. The Go adapter keeps capture independent of inference, requests chunked previews after one second of new audio, and closes stable chunks after 800 ms below -42 dBFS RMS. It cuts in the middle of the pause without discarding audio. A continuous segment is bounded at 20 seconds using the existing quiet-point splitter; stop flushes every uncommitted sample, split at 25 seconds if needed. These bounded batch previews are not causal streaming ASR. The first-partial target remains an open performance gate.

Model loading uses a private temporary directory containing links to verified catalog files only. Swift derives tokenizer.json there; cached tokenizer files and extra weight files cannot enter the loader. Upstream stdout is isolated from the protocol descriptor. Token-limit termination and worker failure are explicit errors, never successful truncated text.

## Adoption and acceptance

The user explicitly adopted MLX 1.7B 8-bit and retired the old stack. A catalog bundle marked `adopted` is the default on its declared platform even before it has production pipeline measurements. This supersedes ADR 004 and MOD-04 for these two native bundles only. It does not imply latency, quality, Windows or minimum-OS acceptance. Measured automatic ranking remains available for other catalogs. Known retired model overrides migrate to this platform default; unknown IDs remain explicit errors. Existing model caches are retained until explicit eviction.

Completed native-worker and coordinator evaluations are recorded separately. Python MLX compute results are historical evidence, not native-package acceptance. Runtime replacement requires new evidence, including cancellation, stale-result rejection, tail preservation, offline loading and relocation.

## Packaging

- Apple silicon: Xcode 26.4.1 / Swift 6.3.1, macOS deployment target 14.0, pinned Package.resolved, arion-asr, colocated mlx.metallib, SwiftPM bundles and the Swift compatibility-library closure. Sign nested code before the outer app and notarize through the normal build command.
- macOS Intel: pinned speech.cpp source and ggml submodule, CPU/Accelerate executable, generic CPU flags and macOS 14.0 deployment target. Cross-build inspection is not native-host acceptance.
- Windows: source-built CPU executable with the static MSVC runtime plus a digest-pinned Vulkan executable. Select Vulkan only when the system Vulkan loader is available. Failure of a selected runtime is surfaced. Vulkan drivers are installed by the GPU vendor. No CUDA or DirectML support is claimed for this package.
- Weights stay in the verified model store. Package verification rejects obsolete runtime files and verifies architecture, minimum OS, notices and signatures.

Windows publisher signing, actual Windows inference/driver acceptance, actual Intel inference and clean macOS 14 execution remain separate distribution gates. GitHub Actions secrets cannot be retrieved as plaintext; CI can use the configured update key directly, while local update signing requires the original private-key backup.
