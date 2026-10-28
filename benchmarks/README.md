# Benchmarks

- [Original Rime LLM reranker reproduction](results/rime-llm-reranker-v1/README.md): 100 real-candidate exploratory fixtures, 219 local model requests including warmups/controls, wall-time measurements, original parser/cache defects and model-free replay.

- [Research archive migration](migration/README.md): copied corpora, per-case results, exact audio, failed/cancelled attempts and frozen source snapshots, with byte identities and model-free report replay.
- [Corpora](corpora/README.md), [raw runs](runs/README.md), [reproduction sources](reproduction/README.md) and [separately provisioned artifacts](artifacts/README.md).

- [Factorial experiment](configs/desktop-factorial.json): 8 feature treatments × 3 seeds × 2 cache states = 48 blocks per corpus.
- [Streaming experiment](configs/desktop-streaming.json): 5 chunk sizes × 3 endpoint thresholds × 3 seeds = 45 warm blocks per corpus.
- [Pinned Hugging Face catalog](datasets.json): dataset metadata only; permissions and import limits recorded.
- [Runner and exact driver/corpus protocol](../tools/evaluation/README.md).
- [Paper-grounded methodology](../docs/research/benchmark-methodology.md).
- [Streaming dataset/import plan](../docs/research/streaming-datasets.md).
- [Contextual rewrite experiment](configs/desktop-rewrite.json): separate rules/LLM, context, approved-memory and scene treatments, FP reference, cold/warm and solo/concurrent-ASR strata. Model/prompt IDs are explicit unprovisioned experiment placeholders.
- [ONNX rewrite candidate catalog](catalogs/rewrite-model-candidates.json): pinned research metadata; no model is approved or measured by this inventory.
- [Rewrite evaluation and human review](../docs/research/rewrite-evaluation.md): token/native-preview/lifecycle observations, preservation proxies and blind intent/style ratings.

- [Phonetic evidence ablation](results/phonetic-asr-1.7b-v2/README.md): all 154 test cases across six Qwen3-1.7B correction conditions, offline acoustic control, confidence routing and output failures. Qwen3-4B was cancelled by the user.
- [Scene switching](results/scene-switching-v1/README.md): scene classifier coverage on [labelled app and site contexts](configs/scene-contexts.json), and rewriting by automatic scene versus fixed chat on developer-authored scene sets.
- [Edit learning and recall](results/edit-learning-v1/README.md): extraction on [labelled edits](configs/edit-learning-cases.json), held-out and in-sample effect of memory learned from dev corrections on [recorded test recognition](configs/edit-learning-asr.json), recall/store/tracker latency, concurrency and storage faults (`make memory-bench`).
- [Recognition-stage correction memory](results/user-modify-asr-v1/README.md): 160 actual Qwen3-ASR-1.7B recognitions comparing empty context, selected destination words, selected correction pairs, an experimental USER_MODIFY wrapper and unfiltered pairs; raw outputs, regressions and synthetic-corpus limitations retained.
- [Recognition memory optimization](results/user-modify-asr-v2/README.md): paper-guided first-pass term retrieval and original-audio re-decoding, development-only policy selection, a frozen new synthetic recurrence test and natural-read regression; actual-call counts and wrong-entity metrics retained.
- [Closed-loop correction memory](results/user-modify-asr-v3/README.md): actual earlier audio, authored user corrections, production extraction/encrypted writes/reload, frozen memory and new later audio with entity-disjoint selection and test.
- [Ordinary-word context probe](results/context-memory-v1/README.md): reconstructed screen semantics, actual 1.7B text inference and product validation; separates retrieved pinyin examples from lexical mappings and keeps the reported Codex case in development.
- [Rewrite context ablation](results/context-ablation-v1/README.md): cursor, history and screen (Vision OCR) context for Qwen3 1.7B and 4B on rendered windows, including negatives and injection.
- [Native recognition validation](configs/desktop-models.json): the explicitly adopted native backend on real-time fixture replay. The prior model-comparison report remains historical evidence in [`results/models-dev-2`](results/models-dev-2/summary.json); its old backend selection is retired under ADR 020. Plans are reproducible without weights/audio; actual runs require licensed local fixtures and a reviewed live adapter. Recorded benchmark audio, transcripts, per-case evidence and reports are archived here; weights, environments, personal recordings and new scratch execution remain in `.workspace/`. Public ASR corpora test recognition; approved-memory relevance, dictionary candidates, safe IME insertion and semantic corrections need additional consented product fixtures.

- [Context ranking and typing latency](results/imk-context-v1/README.md): context ranking lifts later-occurrence top-1 from 32% to 89% on 50 authored context-dependent homophone pairs with no pair regressing; letter-key p95 0.81 ms and candidate-bar p95 0.16 ms after removing per-word window ordering.
- [Offline Pinyin quality and snapshot latency](results/imk-quality-v1/README.md): 100 original synthetic fixtures, 600 actual cold/warm native processes per attempt, reference-free driver inputs and independently replayed 71% intended-commit baseline; original annotation-error attempt retained. Engine-only timing does not establish IMK event/render latency.

- [Qwen3-ASR 1.7B and accessibility](results/qwen17-accessibility-v1/README.md) — native paired ASR comparison, actual window reads and historical final-only integration; its model-selection decision is superseded by ADR 020.

- [Qwen ASR size and quantization diagnostic](results/qwen-asr-speed-v1/README.md) — same-development-corpus native 0.6B/1.7B INT8 and separate MLX 8-bit/4-bit compute/accuracy evidence.

- [Native Qwen3-ASR migration acceptance](results/native-asr-v1/README.md) — native Swift model loading, offline signed-package inference, live coordinator replay and explicit remaining platform/performance gates.

- [General LLM keyboard alignment](results/llm-ime-alignment-v1/README.md) — actual local completion over frozen Rime candidates, no-context abstention, model latency, cancellation and full alignment/evaluation plan.

- [Vocabulary grouping and canonical recall](results/vocabulary-alignment-v1/README.md): production correction/recall comparison, actual local GGUF stage acceptance, immutable failed attempts and report replay.

- [Production ASR hotwords](results/asr-hotwords-v1/README.md): 16 paired development fixtures through the actual native finalizer, canonical spelling and negative controls, sensitive context reset, retained failures and model-free replay.
