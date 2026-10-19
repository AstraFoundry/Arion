# Typing-time candidate re-ranking

How Doubao and Chatterfly reorder Pinyin candidates while the user types, and what Arion adopted. Static reads of installed bundles; no runtime traffic, model weights or ranking behavior was observed.

## Doubao (DoubaoIme 1.0.2, macOS)

`OimeEngine.framework` (arm64 strings) names a per-candidate feature sort, **`oime_engine::pinyin::Engine::CanditateSortByFeature`**, with `input_sort_cand_has_feature`, `commit_sort_cand_has_feature` and `enable_model_rank` switches. Its scorer is an ONNX MLP: `mlp_model.cpp` loads `mlp_no_gpt2.onnx`, `mlp_gpt2.onnx`, `mlp_first_screen.onnx` and `mlp_for_cloud.onnx` through the bundled `libonnxruntime.1.16.2`. Features visible by name include user-dictionary frequency (`freq`, `ud_freq_sum`, `ud_word_learned_count`), **user bigrams by context** (`Associate::GetUserBigramCandByContext(History, …)`) and host text around the cursor (`Engine::SetContext`, `left_context`, `right_context`, `tbcReplaceContext`). Separate `llm_*_resort` / `llm_reorder` names suggest an optional language-model resort. Which features each model consumes, their weights and whether the LLM path runs offline are unresolved.

## Chatterfly (1.0.7, macOS)

The preferences app (`SGPreferences.nib`, decoded with REA, evidence `ev_348a48ab…`) lists **上下文调频** (context frequency adjustment) beside 动态调频 (dynamic frequency), 记录新词 and 云计算候选 under its smart-input options. Its ranking code was not traced.

## Arion

Arion implements the context-bigram part locally and without a model ([ADR 017](../decisions/017-offline-pinyin-keyboard.md)): Rime's own user dictionary already provides frequency learning; the context ranker adds encrypted (previous word, word) counts and reorders the engine's first 27 candidates, never promoting a candidate shorter than the engine's first choice. Unlike Doubao it does not read host text; context is Arion's own previous commit in the same client. The setting is named 上下文调频, as in Chatterfly. Measured effect and latency: [context ranking results](../../benchmarks/results/imk-context-v1/README.md).
