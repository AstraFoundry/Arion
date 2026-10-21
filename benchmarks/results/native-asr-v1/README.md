# Native Qwen3-ASR migration acceptance

Qwen3-ASR 1.7B MLX affine 8-bit loads and transcribes through the native Swift worker. The product catalog retires ONNX Runtime, sherpa-onnx and every X-ASR bundle. The Apple-silicon default is explicitly adopted under [ADR 020](../../../docs/decisions/020-native-qwen-asr.md); this is not a claim that all release gates passed.

## Measured results

Apple M4 / 32 GiB, macOS arm64. Each accepted arm uses the same 118 development fixtures plus two excluded training warmups, greedy decoding, 128 output-token limit and empty context. References and language labels do not enter inference. Model payloads, audio and runtime files are hashed. No new test-split result is claimed.

| Accepted arm | Mandarin CER | English WER | Mixed MER | p50 / p95 |
| --- | ---: | ---: | ---: | ---: |
| Native worker compute | 5.35% | 1.61% | 22.01% | 533 / 1,198 ms |
| Final signed packaged worker, network denied | 5.35% | 1.61% | 22.01% | 490 / 1,110 ms |
| Production coordinator, real-time fixture replay | 5.35% | 1.61% | 22.01% | 559 / 1,387 ms after stop |

All three non-speech fixtures emit no text. Cancellation terminates the active request and the resident model serves another request. The offline run uses the signed worker inside Arion.app, a restricted network sandbox, an unrelated working directory and a minimal environment. The final signed-package run loads in 1,487 ms and records worker-process peak RSS of 2,168,127,488 bytes. The preceding signed-package run loaded in 2,173 ms and recorded 1,491,845,120 bytes; both original runs are retained. This process RSS does not establish total GPU/shared-memory use or minimum system RAM.

The coordinator's first-partial p95 is 1,440 ms, corpus RTF 0.571 and per-case RTF p95 0.811. The 500 ms first-partial and 1,000 ms stop-to-review targets both remain **failed performance gates**. Its original `peakRssBytes` measures only the Go driver; it must not be presented as whole-pipeline memory. See the separate worker RSS above.

The [historical ONNX 1.7B report](../qwen17-accessibility-v1/README.md) had approximately 1,820 ms stop-to-review p95. The new 1,387 ms is approximately 24% lower. This is a descriptive comparison with an earlier run, not a new interleaved paired treatment or proof that MLX alone caused the difference. Synthetic mixed-language fixtures do not establish natural conversational accuracy.

## Package acceptance and remaining gates

The canonical macOS arm64 App/DMG build passes Developer ID signatures, hardened runtime, notarization, stapled tickets and Gatekeeper. Its verifier requires the native executable, Metal shader, SwiftPM bundles, Swift compatibility-library closure and dependency notices, and rejects retired runtime files. The installer is approximately 34 MB; model weights are provisioned separately. The actual packaged bootstrap, with an isolated profile and offline provisioning, reaches `ready` for `qwen3-asr-1.7b-mlx-8bit` without a worker override or microphone capture.

Intel's native CPU executable was cross-built and inspected as x86_64. Windows adapter tests cross-compile; the pinned Vulkan executable is verified x64 and imports system DLLs plus `vulkan-1.dll`, without a separate MSVC runtime dependency. The source-built Windows CPU executable uses the static MSVC runtime. Actual Windows package/inference, physical Intel inference, clean macOS 14 execution, Windows publisher signing and microphone/editor acceptance remain open. No CUDA or DirectML package is claimed.

GH CLI confirms the repository update signing key is configured. GitHub does not return its plaintext. Local Apple signing and notarization work; the local update private key is absent, so local App/DMG acceptance does not establish a signed updater archive. Organization-secret visibility was denied; CI Apple credential visibility was not independently established by that query.

## Retained evidence

- [Descriptive summary](summary.json), [live coordinator report](../../runs/native-asr-v1/pipeline-v2/summary.md) and [per-case outputs](../../runs/native-asr-v1/pipeline-v2/results.jsonl).
- [Final signed offline worker run](../../runs/native-asr-v1/packaged-offline-final/run.json) and its original stdout, warmups, outputs and lifecycle replies.
- [Reproduction sources](../../reproduction/native-asr-v1/README.md), immutable source snapshot, exact old runner bytes, runtime pins and archive verification.

The first compute attempt completed its 118 inference rows but was rejected because upstream tokenizer generation printed a buffered non-protocol line at shutdown. Its original metadata, failure log and outputs are retained; a dedicated stdout descriptor fixes the defect. A first coordinator launch was rejected before inference because single-backend validation had not declared its design. A first signing attempt failed at Apple's timestamp service; the retry retained timestamping and completed notarization. These attempts are not accepted runs. The first packaged-app launch encountered the user's already-running instance; the isolated bootstrap acceptance uses the supported runtime directory separation and leaves that instance running.
