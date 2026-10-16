# Benchmarks

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
- [Model comparison](configs/desktop-models.json): every catalog bundle on identical fixtures and settings; its aggregate report lives in [`results/models-dev-2`](results/models-dev-2/summary.json) and its summaries in `models/catalog.json`, which select the default bundle. Plans are reproducible without weights/audio; actual runs require licensed local fixtures and a reviewed live adapter. Recorded benchmark audio, transcripts, per-case evidence and reports are archived here; weights, environments, personal recordings and new scratch execution remain in `.workspace/`. Public ASR corpora test recognition; approved-memory relevance, dictionary candidates, safe IME insertion and semantic corrections need additional consented product fixtures.
