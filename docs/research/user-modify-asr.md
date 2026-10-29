# Recognition-stage correction memory

Status: feasibility review completed before the subsequent [recognition-stage experiment](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v1/README.md`). The user requires memory to participate in audio recognition and explicitly chose model research first. This review records source/interface findings; the linked experiment separately records actual model provisioning and inference. No text-stage substitute experiment or product integration was performed. [Source revisions, observed source hashes and upstream model metadata](evidence/user-modify-asr-feasibility.json) accompany this report.

## Finding

**Qwen3-ASR-1.7B is the preferred recognition-stage experiment candidate.** It is a different checkpoint from Arion's Qwen3-1.7B text rewriter. Its audio encoder and decoder consume audio together with a context-bearing system message. This makes an audio-plus-correction-memory experiment possible; it does not establish Doubao model or quality parity.

The [Qwen technical report, section 2.2](https://arxiv.org/html/2601.21337v1) describes context-bias training and deliberately suppresses general instruction following. Context tokens serve as recognition background. Consequently, a list of correct spellings is supported by the intended usage, while understanding `wrong → correct` pairs, or a literal `USER_MODIFY` wrapper, remains an empirical question. More forceful editing instructions cannot be assumed to work.

## Verified input and runtime paths

| Candidate path | Verified recognition input | Local feasibility and remaining gate |
| --- | --- | --- |
| Official Qwen3-ASR-1.7B / Transformers | `transcribe(audio=..., context=...)`; `_build_messages` places context in the system message and audio in the user message | Source-reference route. Official examples use CUDA; CPU/MPS correctness, memory and speed on this Mac have not been measured. Official package streaming is vLLM-only in the inspected source. |
| Qwen3-ASR-1.7B / MLX Audio | `generate(audio, system_prompt=..., hotwords=...)`; `_build_prompt` puts the system text before the audio tokens | Preferred Apple Silicon experimental runtime because arbitrary correction-pair text is expressible without changing the runtime. Community 8-bit conversion requires parity checks against the source checkpoint. Not a shared Windows runtime. |
| Qwen3-ASR / sherpa-onnx v1.13.8 | Config `Hotwords`, or offline stream `SetOption("hotwords", ...)`, becomes text in the system segment before audio | Existing Go dependency already exposes recognition-stage bias. CSV input is split, trimmed and joined with spaces: this is a hotword API, not an exact arbitrary-context contract. The local provisioned Qwen ASR is 0.6B, not 1.7B. A compatible 1.7B ONNX export still needs acquisition, graph/metadata and output-parity verification. |
| Fun-ASR-Nano-2512 | Official toolkit example accepts `hotwords` during audio recognition | An additional hotword-control candidate, documented at 800M parameters; it does not satisfy the requested 1.7B condition or establish correction-pair interpretation. Not the first experiment. |

Primary sources: [official Qwen inference source](https://github.com/QwenLM/Qwen3-ASR/blob/7c6daf77a2421100f5fb066495372c00129d39ff/qwen_asr/inference/qwen3_asr.py), [MLX Qwen implementation](https://github.com/Blaizzy/mlx-audio/blob/dad2087f790dfe96b07d5aac1405db4b578b411f/mlx_audio/stt/models/qwen3_asr/qwen3_asr.py), [fixed sherpa implementation](https://github.com/k2-fsa/sherpa-onnx/blob/v1.13.8/sherpa-onnx/csrc/offline-recognizer-qwen3-asr-impl.cc), [sherpa export documentation](https://k2-fsa.github.io/sherpa/onnx/qwen3-asr/export.html), [FunASR example](https://github.com/modelscope/FunASR/blob/main/examples/industrial_data_pretraining/fun_asr_nano/README.md). The local installed Go v1.13.8 module was also checked for `OfflineQwen3ASRModelConfig.Hotwords` and `OfflineStream.SetOption`.

The inspected MLX `stream_transcribe` loads audio and then splits it into chunks before emitting generated tokens. That API is not evidence of causal microphone streaming. Start with complete-utterance acoustic finalization and label it offline; test elapsed-audio streaming separately rather than treating streamed text tokens as streamed audio recognition.

## Model acquisition candidates

| Model | Pinned repository revision | Weight bytes reported upstream |
| --- | --- | ---: |
| [Qwen/Qwen3-ASR-1.7B](https://huggingface.co/Qwen/Qwen3-ASR-1.7B) | `7278e1e70fe206f11671096ffdd38061171dd6e5` | 4,698,521,512 across two safetensors shards |
| [mlx-community/Qwen3-ASR-1.7B-8bit](https://huggingface.co/mlx-community/Qwen3-ASR-1.7B-8bit) | `a8379a2e2f9e313c9292cdf1af4055ab56d50d55` | 2,463,307,541 |

Both model cards declare Apache-2.0. Sizes exclude other assets and are not resident-memory estimates. The ledger retains file metadata including upstream LFS SHA-256 values. Downloaded weights, tokenizer and runtime bytes must be independently hashed before inference. Existing text-model weights cannot be reused as the ASR checkpoint.

## What matching Doubao means

[Retained competitor evidence](edit-learning.md) establishes local correction pairs, reverse-pair suppression, default 30-day expiry, and selection of today's pairs, then high recall counts, then use counts within a total and prompt-length limit. Selected pairs reach its on-device speech model's `USER_MODIFY` slot. It does not recover the checkpoint, exact prompt serialization, active total/length limits, per-bucket quotas, every tie-break or the server's correction extraction algorithm. An experiment can match these observed mechanisms with explicitly declared parameters; it cannot honestly claim a complete reproduction of the competitor.

The earlier [edit-learning benchmark](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/edit-learning-v1/README.md`) measured post-ASR replacement and did not re-measure LLM use of recalled terms. The local Qwen ASR diagnostic explicitly disabled memory and hotwords. Neither is evidence for or against this recognition-stage experiment.

## Experiment design and subsequent execution

The [executed v1 diagnostic](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v1/README.md`) subsequently compared five recognition contexts on 32 synthetic audio cases using the MLX 8-bit checkpoint. It is an exploratory within-checkpoint comparison; source BF16 parity and the remaining gates below were not completed. In particular, the runtime preflight established loading and recognition on a separate warmup, not conversion parity. The primary recurring-term score did not improve with selected correction pairs. This does not establish a recovered Doubao implementation.

1. **P0: establish the audio model.** Provision the pinned source and MLX checkpoints separately, record runtime versions and hashes, and run known audio with empty context. Check source/conversion output parity on a small development set before interpreting memory effects. A failed backend or parity probe remains a failed observation; do not silently switch models or precision.
2. **P0: freeze memory and test audio.** Build memory only from approved prior-session corrections. Test recurring names/terms in different utterances and sessions, with correct ordinary-word controls, unrelated terms, confusable names, reversed pairs, expired entries, silence and noise. Historical public fixtures can serve as regression controls, not unseen test data. Developer-authored synthetic fixtures must be labelled as such; independent human recordings/review are separate evidence.
3. **P0: compare recognition inputs on the same audio and checkpoint.** Use empty context; correct destination terms only; full correction pairs; and Doubao-style selected pairs. Keep identical selected entries for the destination-only and pair-format comparison. Compare the same pair serialization with and without the literal slot wrapper to isolate formatting from selection. Freeze selection parameters and memory once per session, before audio decoding; do not use the test transcript or reference to select entries.
4. **P0: isolate selection policy.** Compare the declared Doubao-style selection against a declared bounded alternative with the same pool and budget. Keep expiry, reverse suppression, deduplication, selection order and length truncation observable. Unknown competitor values are experimental choices, not recovered defaults. First freeze memory across paired runs; online recall-count updates belong to a separate sequential-session replay.
5. **P0: measure raw recognition first.** Report language-specific CER/WER/MER, recurring-term exact recall, correct-input corruption, unrelated-term insertion, silence false emissions, prompt tokens, compute RTF, load time and memory. Record all raw outputs, failures and truncations. Keep deterministic replacement and the 1.7B text rewriter off in the primary arms; a current-product comparator is a separate pipeline and cannot isolate the memory mechanism.
6. **P1: native and streaming gates.** Only after a recognition benefit, test actual incremental audio, stop latency, preview revisions and concurrent rewriting. Archive original fixtures, provenance, per-case results, failed attempts and reproduction sources under `benchmarks/` as required by [008](../../specs/008-evaluation.md). Independent annotation and blind review remain required before a product adoption claim.

Recommendation: first evaluate Qwen3-ASR-1.7B with a real system-context path on this Mac, using MLX as the practical experimental backend and the official source checkpoint as the conversion reference. Keep the product's current ASR and vocabulary behavior unchanged until this comparison establishes both benefit and harm.
