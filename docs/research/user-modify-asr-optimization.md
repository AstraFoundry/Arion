# Optimizing recognition-stage vocabulary recall

The [v1 experiment](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v1/README.md`) exposed inconsistent correction-pair recall. The [v2 measured comparison](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v2/README.md`) implements a compact spelling context retrieved from the first-pass transcript, then re-decodes the original audio with the same Qwen3-ASR-1.7B checkpoint. It performs no transcript replacement and no model training.

## What the papers explain

| Primary source | Finding relevant to the failure | Boundary of this implementation |
| --- | --- | --- |
| [Qwen3-ASR technical report, §2.2](https://arxiv.org/html/2601.21337v1) | The model is trained for ASR rather than general instruction following, while learning to use context as background. | An arrow does not define a supported replacement operation. This is a supported architectural distinction, not an attention-level explanation of individual errors. |
| [Gong et al., Interspeech 2024, §§3.1, 5.1](https://www.isca-archive.org/interspeech_2024/gong24b_interspeech.pdf) | Bias prompts and new bias tags are trained with LoRA and negative/empty contexts. Recognition can degrade with larger lists. | Adding an untrained tag to Qwen3-ASR is not their trained-tag method. Their list-size threshold is specific to their Qwen-Audio setup. |
| [RECAST, Findings EMNLP 2025](https://aclanthology.org/2025.findings-emnlp.203/) | Trains decoder-state/keyword alignment to retrieve a compact shortlist for downstream contextual audio recognition. | We adopt shortlist selection, using an untrained text/pinyin approximation instead of the paper's trained acoustic retriever. Its published gains do not transfer. |
| [Kong et al., arXiv:2512.21828v1, §§2.2, 4.2](https://arxiv.org/html/2512.21828v1) | Separates hotword retrieval from ASR adaptation. More retrieved terms do not monotonically improve recognition; negative contexts and task rewards address false positives. | Top-two retrieval and skipping already-correct spellings are inference-only adaptations. We do not implement GLCLAP, its training-time vocabulary filtering, GRPO or joint beam search. |
| [Harding et al., Interspeech 2023, §2.1.6](https://www.isca-archive.org/interspeech_2023/harding23_interspeech.html) | Selective activation of trained contextual adapters addresses over-biasing on ordinary words. | Our explicit lexical routing has no trained gate and cannot inherit that method's guarantees. |
| [Huber and Waibel, arXiv:2608.05759v1, §V](https://arxiv.org/html/2608.05759v1) | Compares speech LLM context prompting, including Qwen3-ASR, against dedicated biasing methods; large distractor lists and filtering cost matter. | We separately measure term hits, unrelated entity insertion, general-speech changes and total two-pass compute. This is not a reproduction of their checkpoints or corpus. |

## Local diagnosis

Three mechanisms are consistent with v1's observations:

1. **Both spellings compete.** The pair `麦狗 → MyGo` supplies Chinese and Latin alternatives. The model sometimes returns `麦狗`; it is not executing an application-level replacement rule. This competition explanation is a hypothesis supported by the contrast, not a measured attention attribution.
2. **Global counts miss current relevance.** The eight-entry selection omitted GitHub/Codex regardless of what was spoken, yet supplied unrelated high-frequency terms. The v1 `Codex → Kubernetes` error shows why aggregate MER can reward a wrong Latin entity over a multi-character Chinese error.
3. **Slot names are model-specific.** Inspection of the pinned tokenizer's 62 registered added tokens found no `USER_MODIFY`. Our v1 wrapper is ordinary text, not a registered Qwen control token or the recovered serialization of Doubao's prompt slot. Its failure cannot establish that Doubao's own slot is ineffective.

The community 8-bit conversion has not been compared against source BF16, so quantization/export effects remain an alternative contributor. Holding checkpoint and runtime fixed across v2 arms isolates the tested inference-policy changes, not every possible underlying cause.

## Implemented optimization

`benchmarks/reproduction/user-modify-asr-v2/policy.py` keeps approved wrong forms as retrieval keys. It excludes expired and reversed records, searches the first-pass transcript for phonetic spans, ranks at most two eligible entries by normalized edit distance and then usage/recall counts, and skips a target already present with equivalent Latin spacing/case.

The winning development variant sends only the destination spellings in a short natural-language context. It never sends the wrong forms, the first-pass transcript or references as that context. If there is no eligible match, the explicitly recorded `no_retrieval` route keeps the actual first-pass recognition. Otherwise, the same original audio is decoded again with the short spelling context. Every delivered result links to the actual producing model call.

Pinyin is derived from recognized text and adds no independent acoustic evidence. The original waveform contributes fresh evidence only through the second recognition. The policy cannot retrieve a term whose pronunciation has been lost beyond the approved aliases, and a retrieved term may still not be recognized correctly.

## Evidence and remaining work

The exposed v1 corpus became development data. New utterances with different TTS voices and separate natural-read regression controls were frozen before tuning. The selected policy and its source hashes were frozen before the single final test. [The v2 report](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v2/README.md`) contains all scores, raw outputs, source identities and replay instructions.

Further prompt tuning requires another test set. Training a pronunciation-aware/acoustic retriever or a context-aware adapter requires its own data and training experiment. Natural recurring names across real user sessions, independent annotations, BF16 parity, causal streaming and native insertion remain open gates. Production vocabulary behavior and the default recognition bundle are unchanged.
