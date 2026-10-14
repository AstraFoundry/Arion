# Benchmarks

- [Factorial experiment](configs/desktop-factorial.json): 8 feature treatments × 3 seeds × 2 cache states = 48 blocks per corpus.
- [Streaming experiment](configs/desktop-streaming.json): 5 chunk sizes × 3 endpoint thresholds × 3 seeds = 45 warm blocks per corpus.
- [Pinned Hugging Face catalog](datasets.json): dataset metadata only; permissions and import limits recorded.
- [Runner and exact driver/corpus protocol](../tools/evaluation/README.md).
- [Paper-grounded methodology](../docs/research/benchmark-methodology.md).
- [Streaming dataset/import plan](../docs/research/streaming-datasets.md).
- [Contextual rewrite experiment](configs/desktop-rewrite.json): separate rules/LLM, context, approved-memory and scene treatments, FP reference, cold/warm and solo/concurrent-ASR strata. Model/prompt IDs are explicit unprovisioned experiment placeholders.
- [ONNX rewrite candidate catalog](catalogs/rewrite-model-candidates.json): pinned research metadata; no model is approved or measured by this inventory.
- [Rewrite evaluation and human review](../docs/research/rewrite-evaluation.md): token/native-preview/lifecycle observations, preservation proxies and blind intent/style ratings.

- [Scene switching](results/scene-switching-v1/README.md): scene classifier coverage on [labelled app and site contexts](configs/scene-contexts.json), and rewriting by automatic scene versus fixed chat on developer-authored scene sets.
- [Model comparison](configs/desktop-models.json): every catalog bundle on identical fixtures and settings; its aggregate report lives in [`results/models-dev-2`](results/models-dev-2/summary.json) and its summaries in `models/catalog.json`, which select the default bundle. Plans are reproducible without weights/audio; actual runs require licensed local fixtures and a reviewed live adapter. Audio, transcripts, weights and generated reports stay in `.workspace/`. Public ASR corpora test recognition; approved-memory relevance, dictionary candidates, safe IME insertion and semantic corrections need additional consented product fixtures.
