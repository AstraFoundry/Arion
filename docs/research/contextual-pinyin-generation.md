# Contextual Pinyin candidate generation

## Finding

Generating a new whole-composition candidate is a different operation from ranking existing engine IDs. The current Doubao binary contains that operation; msime also implements a separate generated-candidate path. Arion's existing same-range candidate ranking cannot supply a missing sentence merely by waiting longer. The local feasibility study below did not reach the requested approximate 10–19% error tolerance.

This investigation does not enable new candidate generation in the installed app. It adds the local research probe described in [the driver contract](../../tools/driver/README.md). Context acquisition, native candidate insertion, commit, cancellation and focus ownership remain separate implementation/acceptance work.

## Commercial artifact evidence

The installed artifacts were inspected statically, without launching them, reading personal input/preferences or capturing account traffic. REA signature records establish artifact identity. Its deep providers were unavailable on this host (Ghidra unconfigured, Hopper absent, IDA unconfigured), so focused code observations use Apple `nm` and `llvm-objdump`; they are not REA pseudocode or runtime observations.

| Artifact | Exact executable SHA-256 | REA signature Evidence |
| --- | --- | --- |
| Chatterfly 1.0.10.13541 | `1add2da41cc93dc71fcf13649d74d039aadd28fe455c5da304daba43e5530f31` | `ev_d556fafdf838ee25b6b4e19a02025470d30f1fe4409cffb3a5b4eb881cee20ac` |
| Doubao IME 1.0.3 | `823b311215c1a16c2b3f12d88061d19a0d22856f5915eaf8c18cd5bcf57b8f89` | `ev_452865f38bbe2ed9c87cdaac40809cec85dfa6f331f9d314f91453ca0776616f` |

Doubao's nested OimeEngine SHA-256 is `d671cf611eb915600e8abfd4783578c5690e34d813898ad8aeddc564397c9bb0`. Its arm64 `MakeLLMCandidatesWithZhuyin` at unslid VM address `0x172b80` allocates a new candidate, copies LLM UTF16 text, compares it with local candidates, retains an unmatched-text path, then calls `SysDict::Zhuyin`, `ApplyLLMZhuyinResult`, `IsMatchFilterSeqs` and `AppendCandidate`. This is strong static evidence of generated text plus phonetic integration, not index-only reordering. Exact phonetic tolerance and enabled runtime policy remain unknown.

Its `RequestBody::buildLLMRequest` at `0x6fa90` serializes input, left/right context, keyboard/input type and rank version. The network adapter actually calls the engine request function. Admission checks include input characters, short syllables and experiment groups. No screen-text member was established in the inspected builder, and no server model, prompt, trained weights or exact debounce was recovered. A numeric return of 200 was not established to represent milliseconds.

Chatterfly's `requestBodyForInputData:cloudSnapshot:` at `0x100283c40` reads former/latter context, local candidates, user selection and input. The inspected controller sends a merged cloud request and has a pinyin-mismatch cancellation path. Whether this keyboard cloud path uses an LLM or generates text beyond dictionary cloud completion remains unresolved. Speech LLM functions are not proof of keyboard generation. No keyboard screen-text acquisition was established.

Complete ordinary-tool outputs, exact commands and REA signature records remain locally in the private evaluator's `.workspace/contextual-generation-native/`; no competitor implementation or weights are product dependencies. Older reports concern older builds and are not substituted for current execution evidence.

## Open-source comparison

The inspected msime commit is `0e2824c8a2bff6a010e7a40525ff348d0a1a231b`, not a claim about uninspected latest HEAD.

- Its [generated-candidate scheduler](https://github.com/metasequoiaime/msime/blob/0e2824c8a2bff6a010e7a40525ff348d0a1a231b/platforms/macos/src/input/InputController.mm#L2955-L3048) uses a 650 ms debounce and checks session/client/query/epoch before accepting results. This is separate from the 150 ms settled reranker.
- Its [prompt/request](https://github.com/metasequoiaime/msime/blob/0e2824c8a2bff6a010e7a40525ff348d0a1a231b/crates/client-providers/src/ai.rs#L42-L76) produces new candidates from segmented Pinyin and preceding context. The [eligibility gate](https://github.com/metasequoiaime/msime/blob/0e2824c8a2bff6a010e7a40525ff348d0a1a231b/crates/engine/src/session/online.rs#L225-L237) requires every segment to be complete Pinyin; mixed initials require additional work. Screen reading was not established in this path.
- Generated suggestions have their own source and require confirmation; the first AI suggestion occupies the last slot of the first page. Inspected parsing/insertion bounds and deduplicates text, but phonetic correspondence is a prompt instruction rather than a hard parser check. Workspace licensing is GPL-3.0-only; this study uses architecture evidence, not copied code.

At pinned `tcvdog/ime-llm` commit `808011896f7d2e5ddf5af768cab8337b98129ef6`, [the backend](https://github.com/tcvdog/ime-llm/blob/808011896f7d2e5ddf5af768cab8337b98129ef6/ime-core/llm_backend.py) defines sentence generation, but repository-wide search found no caller; the inspected engine invokes ranking. Its GUI timer invokes inference synchronously. A defined helper or README claim is insufficient evidence of a working asynchronous generation path.

## Local feasibility experiment

The probe uses product commit `94280cf0b4ac089b883d6456c68ccd3f6d6af096` and the existing local Qwen3.5-2B Q4_K_M worker; no cloud inference or personal screen contents were used. Model SHA-256 `aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223` matches the catalog. Private harness source and pinned product revision are committed separately. Original records and reproduction artifacts remain in `.workspace/contextual-ime-generation-v1/` in the private evaluator.

Four development and 28 evaluation cases were frozen before inference. Evaluation includes 25 positive completions and three intended abstentions across full/mixed/initial Pinyin, misleading context, no context, boundaries and caret context. English-instruction run 01 is the first frozen evaluation. A Chinese-instruction run 02 uses the same now-exposed set and is an exploratory comparison, not a new held-out test. The deterministic context-span baseline is also exploratory.

| Method | Correct positive completions | Accepted nonempty suggestions | Wrong among accepted | Abstentions | Inference latency p50 / p95 |
| --- | --- | --- | --- | --- | --- |
| Run 01, strict phonetic gate | 2/25 | 4/28 | 2/4 (50%) | 24/28 | 383 / 498 ms |
| Run 02, Chinese prompt + same gate | 4/25 | 6/28 | 2/6 (33.3%) | 22/28 | 493 / 667 ms |
| Unique phonetic context-span retrieval | 13/25 | 13/28 | 0/13 | 15/28 | No model inference |

The raw run-01 generator returned 25 incorrect nonempty outputs out of 28 cases. The gate suppresses many invalid readings, but low overall false-output frequency must not be confused with accurate useful suggestions: run 01 still gets half its accepted suggestions wrong. Both generative prompts failed the requested mixed-initial and multiword regression examples. The span baseline can succeed only when the exact wanted phrase is already present in context; it cannot compose an absent sentence. Its small accepted sample does not establish a release error bound.

For `mo x b j j d y y`, run 01 returned `None` and run 02 echoed the Pinyin instead of producing 模型被拒绝的原因. For `niu rou zi zhu fang ti`, they returned only 牛肉 and 牛肉自助 respectively. All were rejected by the gate. The private `.workspace/contextual-ime-generation-evidence/REPORT.md` and adjacent evidence archive preserve exact outputs, commands, hashes and metric replay instructions.

Observed latency is model-call latency under ordinary local system use, excluding debounce, Accessibility capture and native UI delivery. Both model runs completed without worker errors or truncation. The strict gate uses per-character dictionary readings and one Han character per supplied token; segmentation alternatives, tones, phrase-level polyphony, misspelling tolerance and English/mixed-script output are not covered. Inputs are already segmented; raw keystroke decoding is not tested. No generated candidate was inserted into a real editor, and stale/focus/selection safety was not evaluated by these model-only calls.

## Next implementation boundary

Keep fast local ranking. A new generation operation must accept text-bearing results even when the engine has only one full-range candidate, validate the result against the entire typed Pinyin span, and preserve the original candidates. The response remains provisional until user selection; edits, focus changes and manual navigation invalidate it.

The most promising next experiment is phonetic candidate-lattice or context-span proposal construction followed by contextual model choice, rather than unconstrained generation by the present small model. A dedicated Pinyin-conditioned model or a larger local model may also help, but neither is selected or benchmarked here. Independently repair the current editor-context contract; synthetic context injection does not resolve its `context_marked_mismatch` or `TARGET_UNSUPPORTED` failures.

The user's approximate 10–19% tolerance should be measured as wrong accepted suggestions, alongside useful completion coverage and exact sentence accuracy. Abstaining almost everywhere is not a successful release strategy. Wrong-target insertion, stale commit and unexpected automatic submission remain correctness failures regardless of language-model tolerance.
