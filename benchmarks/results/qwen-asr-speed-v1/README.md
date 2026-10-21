# Qwen ASR size and quantization comparison

**Evaluate MLX 1.7B 8-bit for a macOS backend; keep smaller/4-bit models as speed options with explicit quality costs.** The earlier 1,820 ms measurement already used INT8. Reducing model size speeds the existing CPU path, but changing the Apple Silicon runtime gives a more promising quality/speed balance on this development corpus. Product defaults are unchanged; MLX is an experimental Python runner, not an integrated native desktop backend.

Apple M4, 32 GiB RAM. All five arms recognize the exact same 118 development clips: 48 Mandarin (including three nonspeech controls), 40 English, 30 synthetic Mandarin/English technical clips. Two separate train warmups per arm; 590 measured recognitions plus ten warmups. No reference text, language labels, memory, correction or hotwords reach inference. Error rates are micro-averaged on speech; nonspeech emissions use their separate three-case denominator. No token ceiling was reduced to obtain speed.

## Existing native CPU path

Unchanged production coordinator and driver, sherpa-onnx 1.13.8, four threads, resident models, same X-ASR 160 ms preview, 20 ms real-time replay, 800 ms endpoint, 128 new-token ceiling. Explicit-stop-to-review excludes microphone, rendering and editor insertion.

| Finalizer | Mandarin CER | English WER | Mixed MER | Stop → review p50 | p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1.7B INT8 | 40/635 = 6.30% | 16/620 = 2.58% | 104/368 = 28.26% | 757 ms | 1,765 ms |
| 0.6B INT8 | 54/635 = 8.50% | 18/620 = 2.90% | 110/368 = 29.89% | 412 ms | 977 ms |

The smaller model reduces native p95 by 44.6% in this paired run, with worse rates in all three language groups. It barely meets the provisional 1,000 ms p95 budget; this single development run is not release acceptance. Its p99 remains 1,196 ms. All 118 repeated 1.7B texts exactly match the previous run; p95 varies from the earlier 1,820 ms to 1,765 ms. Both native arms emit nothing on all three nonspeech controls.

## Apple GPU MLX diagnostic

MLX 0.32.4 / MLX Audio 0.5.9, Qwen implementation source `dad2087f790dfe96b07d5aac1405db4b578b411f`. Separate sequential processes, warm synchronized generation, greedy/seed 11, empty context, 128 tokens. Audio is already loaded. These are **engine-compute** measurements, excluding the streaming/coordinator path. They cannot be substituted for product stop-to-review latency or used to promise a native speedup ratio against the table above.

| Model | Mandarin CER | English WER | Mixed MER | Compute p50 | p95 | Weight file |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1.7B 8-bit | 34/635 = 5.35% | 12/620 = 1.94% | 79/368 = 21.47% | 458 ms | 968 ms | 2.46 GB |
| 1.7B 4-bit | 34/635 = 5.35% | 9/620 = 1.45% | 113/368 = 30.71% | 369 ms | 724 ms | 1.60 GB |
| 0.6B 4-bit | 38/635 = 5.98% | 15/620 = 2.42% | 123/368 = 33.42% | 234 ms | 448 ms | 0.71 GB |

Within MLX, 1.7B 4-bit reduces p95 compute by 25.2% and weight-file bytes by 34.9% versus 8-bit. Mandarin corpus error count is unchanged and English improves by three errors, but mixed speech worsens by 34 errors. Some mixed Mandarin utterances become English sentences: the onboarding case remains Mandarin with a wrong term in 8-bit but becomes “We need a better enrollment process.” in 4-bit. This is a language-preservation regression alongside lexical errors, on synthetic TTS fixtures.

Against 8-bit, 4-bit improves error counts in 4 cases and worsens 11, with equal counts in 103; equal counts do not mean equal words. Both 4-bit models also emit on one of three nonspeech controls (`I.` for 1.7B; `The.` for 0.6B), versus zero for 8-bit. All MLX runs have zero 128-token-limit hits. Weight-file size is not runtime RAM; no minimum-memory claim follows.

The [1.7B 4-bit](https://huggingface.co/mlx-community/Qwen3-ASR-1.7B-4bit) and [0.6B 4-bit](https://huggingface.co/mlx-community/Qwen3-ASR-0.6B-4bit) are Apache-2.0 community conversions of the official Qwen models. Their exact revisions, payload identities and quantization configurations are archived. Community “4-bit” labels do not mean every acoustic tensor is quantized. Cross-runtime differences change export/provider and decoding implementation as well as precision; the better MLX 8-bit rates cannot be attributed to a single component. Floating-point source parity is unmeasured.

## Decision and evidence

For Mandarin/English mixed technical dictation, MLX 1.7B 8-bit is the strongest candidate measured here. 1.7B 4-bit saves time and storage but materially worsens mixed recognition; 0.6B 4-bit is fastest but has the highest mixed error rate. The native 0.6B INT8 path is immediately testable with the existing optional bundle, with the measured Mandarin accuracy cost. No default, runtime packaging or Windows backend is changed by this experiment.

A production MLX choice still needs native integration, actual stop-to-review/streaming contention, cancellation/focus invalidation, offline deployment and target-memory validation. These are exposed development observations, not a held-out comparison, independent semantic review or microphone-to-editor acceptance. Mixed speech is synthetic; no population or significance claim is made. Native CPU provisioning downloads overlapped part of that run, but no two inference jobs ran concurrently; timing is from a shared machine without a controlled thermal/energy study.

[Every output, warmup, original failure and runtime identity](../../runs/qwen-asr-speed-v1/), [recomputed per-case summary](summary.json), [reproduction commands and source](../../reproduction/qwen-asr-speed-v1/README.md). Report replay verifies all output hashes and complete pairing without weights. The initial derived report that incorrectly included nonspeech insertions in speech rates is retained and marked invalid; published rates follow the existing evaluator's speech-only convention. An initial reporter guard also confused raw-file and compact-JSON corpus digests; both identities are now checked explicitly. Neither correction changes inference bytes.
