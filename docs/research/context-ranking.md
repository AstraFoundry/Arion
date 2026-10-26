# Typing-time candidate re-ranking

Installed-artifact investigation of Doubao and Chatterfly, with a separate description of Arion. Both clients contain local candidate decoding/ranking and conditional model/cloud enhancement. These static paths do not establish active runtime policy, measured quality or latency.

## Inspection and provenance

The `reverse-engineer-anything` workflow was used against installed macOS bundles. REA 6.4.0 `inspect-macho` generated fresh identity/metadata evidence; native function decompilation was unavailable because no current decompiler provider was configured. Focused Apple/LLVM disassembly was checked against the current arm64 bytes. These ordinary static call anchors are distinguished from REA Evidence IDs.

- DoubaoIme 1.0.2 build 1: engine SHA-256 `d1cebd7c9a9d888acc2a6d2f4aad372cb3b31570ff68a7e631cba930517f5db8`, REA `ev_c97d8d9800d3ba5fab17bbaf881ffef6b49bb167925284e9475eda82ded512d0`. [Focused ledger](evidence/doubao-ranking-followup.json) and [earlier matching engine investigation](evidence/keyboard-ime.json).
- Chatterfly 1.0.8.13459: executable SHA-256 `595e502f74e41e1b28904affa9093974f6cbc197a3756a74a007ef562f9ee767`, REA `ev_d610f14c9a0052e5d4a20192869665ae142b6c207c3438bc09e1b0c9e92860ef`. [Current-version ledger](evidence/chatterfly-ranking-followup.json), 26 selected functions/windows and 5,846 verified instructions. Earlier 1.0.6/1.0.7 addresses and runtime probes are not observations of this version.

No applications were launched or attached, and no user text, private application storage or live requests were inspected. Complete REA records and proprietary instruction listings remain in ignored `.workspace/research/typing-rank-rea-v1/`; sanitized ledgers retain hashes, call addresses and limitations.

## Doubao: local feature ranking and conditional LLM fusion

The matching engine investigation establishes native Pinyin lattice/dictionary conversion, including user/system/extension dictionaries and sentence conversion. User-context bigrams and surrounding-text interfaces also exist. The focused call chain is:

```text
Engine::UpdateResult
  -> ExtractCandFeature
  -> CanditateSortByFeature
       -> MLPModel::CalcScoreByFeature
            -> CombinedFeature
            -> CalcScoreByFeatureCombined
                 -> FeatureBucket / FeatureStandard
  -> MergeCloud
       -> conditional old/new merge strategy
       -> new-session MergeInner
            -> MergeCloudConvert / MergeCloudLLM[_SP]
            -> MakeLLMCandidate
            -> PromoteByDecisionTree / other promotion rules
```

`UpdateResult` calls local feature sorting at `0x1bc334` and cloud merging at `0x1bc464`. Candidate sorting calls the MLP scorer at `0x1c3e74`. The bundled ONNX runtime and loader literals name `mlp_no_gpt2.onnx`, `mlp_gpt2.onnx`, `mlp_first_screen.onnx` and `mlp_for_cloud.onnx`. These establish model-loading paths, not a decoded inventory of active weights or a public GPT checkpoint.

Cloud/LLM enhancement has a separate client request chain carrying raw input, preceding/following text, request serial, keyboard scheme and `shouldSendLLMRequest`. In the new-session fusion branch, `MergeCloudLLM` calls `MakeLLMCandidate` at `0x13a19c` and `PromoteByDecisionTree` at `0x13b6b4`. The promotion routine builds features that reference both LLM items and local candidate features. Multiple explicit promotion and long/short-change adjustment routines are also called. This supports selective candidate fusion after local ranking; it does not establish that the server returns a permutation of the existing candidates.

A bounded resource follow-up found no GPT-2-named file in the installed bundle, but confirmed `mlp_gpt2.onnx` and `EnableGpt2` literals in the engine. Two hashed resources (`d9cce8a55257`, 167,497 bytes; `3aa2ea70d0b2`, 176,585 bytes) contain readable `onnx.quantize` metadata and integer-matmul/linear-layer names. The former references `convert_no_gpt2.py`. These are not wholly encrypted opaque payloads; no graph parser validated them or mapped either to `mlp_gpt2.onnx`. A complete GPT-2 checkpoint remains unconfirmed.

Unknown: active old/new strategy, model weights, exact feature mapping, remote checkpoint/prompt, triggering frequency, actual candidate movement and latency. The symbol `old_900M` cannot establish a deployed 900M model.

## Chatterfly: local LM scores, feature ranking and conditional generation

Current-version `GetConvertData` calls local candidate construction (`FillWord`, `SmartFuzzy`), a conditional generation entry, `SortCandsByLtr` and conditional cloud-request/fusion paths. The dedicated scoring chain includes:

```text
FeatureSortHandlerLm::SortByFeatures
  -> LMInferWithCloudCache
  -> score inference
  -> GetEntryScoreImpl
       -> LocalLanguageModel::IsAttach
       -> LocalLanguageModel::GetScore

LoadLocalLanguageModel: lanmodel.xnet
  -> AttachModel -> AttachModelXNet
  -> InferenceEngineXNetLMIncremental
```

`GetEntryScoreImpl` reaches `GetScore` at `0x100ac338c`. A separate `model.lm.xnet` loader and `GetSortFeatureScoreLm` path convert features to float arrays and call `t_sortFeatureModel::GetScore` at `0x100c3a210`. This is stronger evidence for input-specific scoring than the earlier preferences-only observation of 上下文调频. `LMInferWithCloudCache` also incorporates cached cloud candidates; its name alone does not prove a network request during scoring.

Generation is separate: `StartGenerateModelInferenceAsync` checks enablement and `IsNeedGenerateCand`, then reaches `RunGenerateCand`, which constructs a `std::thread` at `0x100ad35e0`. A loader references encoder/decoder XNet and SentencePiece assets. Local LM mode predicates accept numeric modes 4/5; generation enablement is set for 5/6 and has an additional suppression guard. Active modes were not inspected.

The current app bundle contains none of the six referenced XNet/SentencePiece asset names or `.xnet`, `.gguf`, `.safetensors`, `.onnx` files. This bounded result does not exclude external provisioning or opaque packaging. LM architecture, parameter count, active weights and cloud model remain unknown.

## Implications for Arion

The evidence favors evaluating a hybrid architecture: preserve immediate local candidate decoding/ranking, and treat model-generated candidates or richer contextual reranking as an additional conditional path. Neither competitor's static client supports the claim that every key waits for a general chat LLM.

A dedicated LM that scores candidates is different from a generic query/document retrieval reranker. The observed MLP/XNet components are not evidence that a public BGE/Jina-style retrieval model will improve Pinyin ranking. Candidate scoring and generation should be measured separately on the same real engine snapshots, including top-1 rescues/breakages, p50/p95 latency, cancellation and stale-result handling. Reusing Arion's LLM transport is an implementation option; these competitor artifacts do not establish whether subscription/cloud or local chat models meet the typing budget.

## Arion

Arion implements the context-bigram part locally and without a model ([ADR 017](../decisions/017-offline-pinyin-keyboard.md)): Rime's own user dictionary already provides frequency learning; the context ranker adds encrypted (previous word, word) counts and reorders the engine's first 27 candidates, never promoting a candidate shorter than the engine's first choice. Unlike Doubao it does not read host text; context is Arion's own previous commit in the same client. The setting is named 上下文调频, as in Chatterfly. Measured effect and latency: [context ranking results](../../benchmarks/results/imk-context-v1/README.md).

## Open-source LLM filter reproduction

The unchanged `FuJiK0/rime-llm-reranker` filter was executed over real Rime snapshots with local Qwen3.5-0.8B BF16/oMLX. On 100 authored exploratory fixtures, top-1 regressed from 56% to 45% (3 rescues, 14 breakages); 89 actual HTTP calls had 288.85 ms p50 / 336.44 ms p95 latency. Its near-1 ms CPU timer does not measure waiting for inference. Original UTF-8 parsing and positional-cache defects were reproduced. This does not change Arion's production ranker. See the [report, original outputs and reproducible corpus](../../benchmarks/results/rime-llm-reranker-v1/README.md); this is an engine/filter-boundary pilot, not installed IMK/Squirrel acceptance or a general LLM comparison.

## General LLM alignment follow-up

The [general LLM alignment](llm-ime-alignment.md) implements a separate Go ID-only selector over the existing local worker/provider completion interface and a latest-request application scheduler. The [actual 2B pilot](../../benchmarks/results/llm-ime-alignment-v1/README.md) reaches 68/100 versus producer 56/100 with eight breakages; 76 scored model calls take approximately 404 ms p50 / 515 ms p95. The installed native keyboard does not invoke this experimental path. Native integration, provider consent and release gates remain explicit.
