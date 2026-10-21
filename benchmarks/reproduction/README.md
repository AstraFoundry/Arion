# Experiment reproduction sources

These are byte-preserving snapshots of scripts, prompt definitions, baseline Go files, generated worker configurations and evaluators used in the archived experiments. Original report configurations are under `report-configs/`; current report configurations under `../configs/` resolve archived inputs.

The exact evaluator source before adding archive support is in `evaluation-source/`. Selected product/driver source is in `selected-source/`. SHA-256 identities and original locations are in the [archive map](../migration/archive.json). A snapshot may contain old relative imports or absolute machine paths; it is retained to explain an experiment, not presented as an automatically runnable entry point.

Use current commands documented in [evaluation tooling](../../tools/evaluation/README.md). For scoring existing observations, use `pnpm bench:archive:report --out <new-directory>`; no model/runtime is needed. New inference requires separately provisioned artifacts and a new output path. Do not edit these snapshots to retrofit a new environment or silently overwrite the original execution evidence.

The new [recognition-stage memory study](user-modify-asr-v1/README.md) has its own audio builder, inference runner and model-free report command. Its identities are in the [study archive](../results/user-modify-asr-v1/archive.json); it is not part of the historical migration report command.

The [recognition memory optimization](user-modify-asr-v2/README.md) adds first-pass retrieval, original-audio re-decoding, entity-aware scoring and development-only policy selection. Use `pnpm memory:asr` for new inference and `pnpm memory:asr-report` for model-free replay; its [separate archive](../results/user-modify-asr-v2/archive.json) preserves both splits and their actual calls.

[Closed-loop memory](user-modify-asr-v3/README.md) uses actual production extraction/encrypted persistence before later audio recognition. The [ordinary-word context probe](context-memory-v1/README.md) separately evaluates text rewriting and reconstructed screen semantics. Both have model-free report commands and frozen production-source snapshots; keys, runtime binaries and model weights are separately provisioned.

[Context ranking](imk-context-v1/README.md) replays seeded homophone streams through the production typing owner and candidate bar with context ranking off and on, and times keys and candidate-bar updates.

[Offline Pinyin engine baseline](imk-quality-v1/README.md) compiles the production native engine/encrypted synthetic-key backend, freezes 100 original reference-isolated fixtures, runs paired cold/warm repetitions and provides model-free replay with corruption/leakage/incomplete-run rejection. No installed input-source or Keychain operation occurs.

[Latest small-model rewriting](rewrite-small-models-v1/README.md) compares current-generation text-only Qwen3.5 bundles <=2B with complete unfiltered output review and a model-free replay. Historical [model screen](rewrite-model-screen-v1/README.md), [serializer](rewrite-4b-packaging-v1/README.md) and [entity](rewrite-4b-entity-v1/README.md) diagnostics are retained; the user excluded further4B and older-Qwen inference.

Latest0.8B follow-ups retain the [four-scene prompt package](rewrite-small-prompt-v1/README.md), [default-batch cache comparison](rewrite-small-cache-v1/README.md), [failed logical-batch1 startup](rewrite-small-cache-parity-v1/README.md) and [physical-microbatch1 parity screen](rewrite-small-cache-parity-v2/README.md). All112 generated bodies, including16 warmups, were personally inspected; the failed startup contributes zero generations. New prompt evaluation remains unexecuted after its failed development gate. No production adoption.

- [2B contrast diagnostic](rewrite-2b-contrast-v1/README.md):32 reviewed outputs; failed ambiguity gate.
- [Pronunciation input evidence](rewrite-pronunciation-evidence-v1/README.md):48 reviewed outputs; repairs improve but ambiguity still fails.

- [Source-plan package](rewrite-source-plan-v1/README.md):56 reviewed generations; valid JSON but worse repairs and natural scene quality. Not adopted.
- [Other small-model acquisition](rewrite-other-small-models-v1/README.md):revision-pinned Granite1.46B metadata and tensor verification only; not inference evidence.

- [rewrite-local-protection-v1](rewrite-local-protection-v1/README.md):84 raw generations personally inspected. Qwen direct→protected25→28/28 faithful and usable; both9/9 repairs.25 unencoded request/body pairs are identical. Residual conversational particles are a minor recorded style issue, not a perfect-fluency claim. Granite19/28 faithful,18/28 usable,7/9 repairs; actual empty EOS and hostile OCR instruction following retained. The main known-development gate passes for Qwen protected, but the separately frozen8-case stress fails. No qualified selection, fresh24 generation or production adoption.

- [rewrite-local-protection-stress-v1](rewrite-local-protection-stress-v1/README.md):16 raw generations, all maps and restored/null bodies personally inspected. Direct→protected faithful1→4/8, usable0→3/8. Protected has2 protocol failures and0/2 delivered repairs; direct has2/2 spelling repairs but other meaning errors remain. Multiple terms, literal quotation and explicitly cash-not-cache meaning still fail; repeated terms become duplicate tasks. Both protocol-rejected raw bodies are also incorrect: zero observed false rejection of a correct native body. Missing-marker counts alone never establish semantic failure.

## Current rewrite evidence

- [rewrite-current-task-v1](rewrite-current-task-v1/README.md)
- [rewrite-system-task-probe-v1](rewrite-system-task-probe-v1/README.md)
- [rewrite-cleanup-preflight-v1](rewrite-cleanup-preflight-v1/README.md)
- [rewrite-natural-source-audit-v1](rewrite-natural-source-audit-v1/README.md)
- [rewrite-timing-audit-v1](rewrite-timing-audit-v1/README.md)

Current-task108 and system-probe12 native outputs are fully inspected. User-tail rules can be copied as editable text; trusted system placement fixes four regressions but returns only to8/12 faithful and7/12 scene-ready on the same hard subset. No candidate qualifies, and the mandatory36-case pool and fresh24 are not reduced. Removing rejection alone does not repair wrong native content. Original request messages and restored bodies are both visible in the living HTML.

The actual production-cleanup source replay changes one36-case input; prior raw-direct experiments remain unchanged and correctly scoped. Natural-ASR preparation verifies120 historical public read-speech sources, not new candidate inference or independent held-out data. Timing audit flags9/396 historical observations without attributing a sleep/runtime cause or approving latency. Minimum target remains Apple Silicon16GB; the M4/32GB host does not substitute for it. Original-language/mixed-input and native-chain acceptance remain pending.

## Cleanup-aligned rewrite diagnostics

- [rewrite-private-spans-v1](rewrite-private-spans-v1/README.md)
- [rewrite-natural-text-review-v1](rewrite-natural-text-review-v1/README.md)

## Input-object and2B artifact diagnostics

- [rewrite-data-object-v1](rewrite-data-object-v1/README.md)
- [rewrite-2b-precision-v1](rewrite-2b-precision-v1/README.md)

- [Bounded spelling and isolated formatting](rewrite-bounded-edit-v1/README.md):72 actual stage observations, full source-bound replay and unqualified36-body review.
- [User-confirmed spelling alignment](rewrite-spelling-alignment-v1/README.md):25 retrospective complete-body reviews,16 reclassified without inference or original-score changes.

- [Bounded-edit v2](rewrite-bounded-edit-v2/README.md):72 native source-letter/correction/visible-formatting observations.
- [Formatter v3](rewrite-formatter-v3/README.md):36 new formatter outputs, explicit reuse of native selector results, no duplicate generation count.
- [Metal capability probe](rewrite-metal-preflight-v1/README.md):one identical-request generation and separate zero-generation placement observation.

- [rewrite-metal-regression-v1](./rewrite-metal-regression-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-natural-format-v1](./rewrite-natural-format-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-language-profile-v1](./rewrite-language-profile-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-language-profile-v2](./rewrite-language-profile-v2/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-cleanup-copula-v1](./rewrite-cleanup-copula-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-metal-chain-v1](./rewrite-metal-chain-v1/README.md): continuous actual Go cleanup, conditional Metal spelling selection and source-only formatting; all outputs reviewed separately from release acceptance.

- [rewrite-metal-prefix-v1](./rewrite-metal-prefix-v1/README.md): complete actual174-response cache-on parity/quality and source-bound timing, with initial misses retained.

- [rewrite-metal-primed-v1](./rewrite-metal-primed-v1/README.md):380 actual measured/startup outputs, both source orders, explicit readiness costs and limited canary controls.

- Full historical401-source cleanup/selection/formatting regression: [reproduction](rewrite-history-chain-v1/README.md).465 actual responses; synthetic281/natural120 separated; no output filtering or release adoption.

- Full401-source formatter prompt revision: [reproduction](rewrite-history-format-v2/README.md).401 actual final bodies reviewed; prior cleanup/64 choices fixed. Improvements and new background-fact/repetition regressions retained.

- Work-scope v3/v4: [v3](rewrite-work-scope-v3/README.md), [v4](rewrite-work-scope-v4/README.md).126 plus110 actual responses, all reviewed; background/status, unchanged details, owner/date layout and16 synthetic controls.

- Tone-insensitive Chinese spelling and complete401-history/24-control chain: [reproduction](rewrite-tone-chain-v1/README.md).538 actual responses reviewed; one correction gained and one harmful substitution introduced, rejected.

- [Semantic selector policy](rewrite-selector-semantic-v1/README.md):609 actual responses, role/context ablations, positive repair regressions and explicit raw metadata limitation.

- [Concise semantic selector](rewrite-selector-semantic-v2/README.md):558 actual responses, restored entity repairs, remaining role failures and corrected intermediate-choice display.

- [Candidate sentence previews](rewrite-selector-preview-v1/README.md):558 actual responses isolate preview data; better positives but nine new harmful substitutions, rejected.

- [Four-arm selector screen](rewrite-selector-screen-v1/README.md):404 actual choices after complete457-source preparation; no new final bodies or release qualification.

- [Granite native-runtime diagnosis](rewrite-selector-model-v1/README.md)
- [Granite CPU-layer selector comparison](rewrite-selector-model-cpu-v1/README.md)

- [Public referent-description selector experiment](rewrite-selector-referent-v1/README.md)

- [Source-last selector serialization](rewrite-selector-source-last-v1/README.md):404 actual choices; source-last increases harmful substitutions in both paired arms, rejected.

- [Counterbalanced whole-sentence selection](rewrite-selector-sentence-choice-v1/README.md):404 actual choices, harmful19/5 in A/B candidates, both rejected; raw labels/normalized edits and allowed normalization differences retained.

- [Granite1.46B hybrid comparison](rewrite-selector-hybrid-v1/README.md):12 native compatibility probes plus202 identical-request choices; harmful20/19 despite47/47 required repairs, both rejected.

- [Trusted instruction-language comparison](rewrite-selector-instruction-language-v1/README.md):404 actual choices across two current small models; only trusted system text changes. All English variants increase harmful choices and are rejected.

- [Complete-reference message separation](rewrite-selector-reference-turn-v1/README.md):404 actual choices; full OCR preserved in separate reference turns; both variants rejected after complete review.

- [Unverified unique-label exposure](rewrite-selector-unique-label-v1/README.md):404 actual choices; only the unique-status field is omitted; all4 changed choices regress, both variants rejected.

- [Added semantic contrast demonstrations](rewrite-selector-contrast-demo-v1/README.md):404 actual choices after zero-inference native token-budget checks; one improved and five regressed choices, both variants rejected.

- [MiniCPM5-1B comparison](rewrite-selector-minicpm-v1/README.md):202 unchanged-request choices plus12 native responses; both configurations rejected. One interrupted native-log completion is retained separately, with no fabricated response credit.

- [MiniCPM official/native token IDs](rewrite-minicpm-tokenizer-v1/README.md):202 complete prompt sequences match exactly; zero added model generations, not semantic or numerical-weight equivalence.

- [MiniCPM F16 export comparison](rewrite-selector-minicpm-f16-v1/README.md):202 unchanged-request choices plus12 native responses; more required repairs but15/17 harmful choices, both configurations rejected.
- [Youtu1.96B native and same-request comparison](rewrite-selector-youtu-v1/README.md):12 probes and202 choices personally reviewed,75/84 conform and3/4 harmful; both arms rejected, custom-license research only.
- [LFM2.5 actual1.17B Q8 comparison](rewrite-selector-lfm25-v1/README.md):12 native and202 selector responses reviewed, both79/101 and22 harmful; rejected.

- [Existing ten-example order](rewrite-selector-demo-order-v1/README.md):404 actual choices; moving keep examples last reduces harmful edits but loses supported repairs.3 improvements,23 regressions,1 allowed change; both treatments rejected.

- [Native d1 decision packaging](rewrite-d1-evidence-v1/README.md):72 actual typed responses over six sources/six layouts, all reviewed. Native96/45 piece clipping identified; full-policy correction remains3/6 per backend. No final bodies or production adoption.

- [Selected rewrite product integration](rewrite-product-integration-v1/README.md) — production port parity fixtures, native opt-in test and offline archive verification.

- [Independent instruction-only prompts](rewrite-independent-prompts-v1/README.md) — Gemini-authored candidate with no supplied benchmark examples; not activated.

The [numeric and hesitation formatter check](rewrite-numeric-filler-v1/README.md) preserves baseline, failed and revised actual-worker text observations for natural-layout-v2.

The [technical spelling follow-up](rewrite-technical-spelling-v1/README.md) checks phonetic English terms and protected Chinese names in the actual formatter.

- [Native Qwen3-ASR 1.7B and system accessibility](qwen17-accessibility-v1/README.md) — pinned files, native window probes, protocol diagnosis, code snapshots and model-free report/verification commands.

- [Qwen ASR size and quantization diagnostic](qwen-asr-speed-v1/README.md) — same-development-corpus native 0.6B/1.7B INT8 and separate MLX 8-bit/4-bit compute/accuracy evidence.

- [Native Qwen3-ASR migration acceptance](../results/native-asr-v1/README.md) — native Swift model loading, offline signed-package inference, live coordinator replay and explicit remaining platform/performance gates.
