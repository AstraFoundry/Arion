# Recorded experiment evidence

- [Cloud polishing main integration](cloud-polishing-main-v1/README.md): merged final-only timing and punctuation configuration, 68 gateway cases and related race/UI checks.

- [Cloud polishing gateway verification](cloud-polishing-gateway-v1/README.md): 60 synthetic OpenAI HTTP integration cases, both wire formats, context switches, coordinator delivery, failures and cancellation; no real inference claim.

- [Unified polishing verification](unified-polishing-v1/README.md): shared native/cloud pipeline checks, actual worker integration and retained repeated native timeout/session-erasure failure. Real cloud and full product E2E remain unverified.


These directories retain actual per-case outputs, source/prompt observations, warmups, failure records, timings and logs. See the [archive map](../migration/archive.json) for byte identities and original locations, the [corpora index](../corpora/README.md) for inputs and the [benchmark index](../README.md) for findings.

| Study | Evidence |
| --- | --- |
| [ime-activation-v1/](ime-activation-v1/README.md) | Signed macOS activation/retry/cancel/reinstall through production adapter, real isolated IMK Pinyin commit/cancel, retained connection failure, check logs/source hashes; clean-account first consent remains open |
| [imk-quality-v2/](imk-quality-v2/README.md) | Corrected 100-fixture corpus revision 2; 600 actual native cold/warm processes, candidate/commit outputs, monotonic timings, source/hardware/runtime hashes and independent report replay |
| [imk-context-v1/](imk-context-v1/README.md) | Context ranking off/on over three seeded homophone streams in fresh profiles, before/after candidate-bar latency runs, copied sources and SHA-256 manifest |
| [imk-quality-v1/](imk-quality-v1/README.md) | Original 600-process attempt retained unchanged with one known authored-input typo; complete original corpus, outputs and provisional 70% score |
| [imk-keyboard-v9/](imk-keyboard-v9/README.md) | Exact profile-bearing signed host still returns enable0 with disabled parent; restored Chatterfly/key absence/no own entries/no host; user deferred system acceptance |
| [imk-keyboard-v8/](imk-keyboard-v8/README.md) | Profile-authorized real Data Protection lifecycle6/6 verified/restored, original reporter-limit predecessor and preparation failure, exact source bindings, tooling87/87 and completed signed/notarized package; no private assets |
| [imk-keyboard-v7/](imk-keyboard-v7/README.md) | Literal-ü failure and fixed native/Rosetta23-case runs, inconclusive inactive IMKCandidates probe with setup failures, and focused static Doubao queued-UI ownership anchors |
| [imk-keyboard-v6/](imk-keyboard-v6/README.md) | Official Apple stage-0 IMK source rebuilt/signed/notarized: enable returned0 but source disabled; failed preparation, license/source provenance and restored diagnostics; no selection/host launch |
| [imk-keyboard-v5/](imk-keyboard-v5/README.md) | Failed strict Data Protection capability, two restored real legacy-rejection probes, strict-provider synthetic native/Rosetta 22-case runs and failed/restored actual GUI caller registration |
| [imk-keyboard-v4/](imk-keyboard-v4/README.md) | Failed/restored fresh-filename and notarized fresh-ID OS probes, restored first-party TIS control, public metadata provenance and pre-fix real Keychain lifecycle with device-bound policy unverified |
| [imk-keyboard-v3/](imk-keyboard-v3/README.md) | 22 translated x86_64/Rosetta cases on Apple Silicon, separate later metadata-only patch, completed final controls/notarized build and restored UI attempt that did not reach installation |
| [imk-keyboard-v2/](imk-keyboard-v2/README.md) | 22 completed real-engine/AppKit cases with exact attempt-12 sources; completed signed/notarized build and failed/restored signed source selection (-50), separately labelled raw and transcribed evidence |
| [imk-keyboard-v1/](imk-keyboard-v1/README.md) | 20 completed real librime/AppKit functional cases, all earlier failed attempts, exact latest sources and two failed/restored system registration attempts; SHA-256 verification without runtimes |
| `disfluency-alignment/` | Matched baseline/product arms, rejected span/prompt probes, freeze and build manifests |
| `rewrite-alignment/`, `multilingual-disfluency/` | Earlier reliability probes, model comparisons and regression checks |
| `live-rewrite/` | Original live text and streaming/final audio observations |
| `context-ablation/`, `scene-switching/` | Context/OCR and automatic-scene observations, including development iterations |
| `edit-learning/` | Native learning/recall measurements and separate test ASR observations |
| `model-selection/` | Earlier incomplete attempt and completed bundle comparison |
| `qwen-diagnostics/` | Text, thinking, two-pass, frozen-ASR, model, prefix and acoustic diagnostics, plus disclosed review tasks/mappings |
| `phonetic-asr/` | Failed first attempt, cancelled full matrix, completed 1.7B view, preflight and offline acoustic control |
| `user-modify-asr-v1/` | Qwen3-ASR-1.7B audio-stage five-arm comparison, preflight, warmups and acquisition failures; identities in the [study archive](../results/user-modify-asr-v1/archive.json) |
| `user-modify-asr-v2/` | Development ablations, frozen policy selection and final recognition-stage retrieval comparison; actual calls, shared first-pass observations, paper sources and identities in the [study archive](../results/user-modify-asr-v2/archive.json) |
| `user-modify-asr-v3/` | Empty-vault teaching, encrypted-file/reopen snapshots, later audio inference and pre-inference split failures; [study archive](../results/user-modify-asr-v3/archive.json). Private vault keys are excluded. |
| `context-memory-v1/` | Actual text-model generation, prior corrections, rendered product/research prompts and validator decisions on reconstructed context; [study archive](../results/context-memory-v1/archive.json). No genuine OCR/audio execution is implied. |

`phonetic-asr/phonetic-asr-v2/completed-17b-view` is an exact subset of the original cancelled run. The archive keeps all excluded 4B observations; the completed report does not score them. `model-selection/models-dev-1` retains its original incomplete `running` metadata. Do not change historical statuses to make an archive check appear complete.

Review tasks and mapping files are now disclosed historical material. Their existence does not establish completed independent ratings. Migrated observations remain historical; the separately indexed new study has its own provenance. None of these experiments establishes product release acceptance.

- [IMK keyboard v10](imk-keyboard-v10/README.md) — actual add-dialog visibility failure with notarized-host/fresh-Settings controls and restored original bundle.

- [IMK keyboard v11](imk-keyboard-v11/README.md) — user-reported Allow separated from unknown native grant persistence; ad hoc replacement observation, corrective source bindings and completed formal build/read-only payload validation.

- [Latest small-model rewriting](rewrite-small-models-v1/README.md):48 fully reviewed Qwen3.5 outputs, unfiltered by content validators; separate exposed/fresh strata and actual CPU timings. No production adoption.
- Historical [checkpoint](rewrite-model-screen-v1/README.md), [serializer](rewrite-4b-packaging-v1/README.md) and [entity](rewrite-4b-entity-v1/README.md) screens preserve56 reviewed calls completed before the user excluded further4B testing.

Latest0.8B follow-ups retain the [four-scene prompt package](rewrite-small-prompt-v1/README.md), [default-batch cache comparison](rewrite-small-cache-v1/README.md), [failed logical-batch1 startup](rewrite-small-cache-parity-v1/README.md) and [physical-microbatch1 parity screen](rewrite-small-cache-parity-v2/README.md). All112 generated bodies, including16 warmups, were personally inspected; the failed startup contributes zero generations. New prompt evaluation remains unexecuted after its failed development gate. No production adoption.

- [2B contrast diagnostic](rewrite-2b-contrast-v1/README.md):32 reviewed outputs; failed ambiguity gate.
- [Pronunciation input evidence](rewrite-pronunciation-evidence-v1/README.md):48 reviewed outputs; repairs improve but ambiguity still fails.

- [Source-plan package](rewrite-source-plan-v1/README.md):56 reviewed generations; valid JSON but worse repairs and natural scene quality. Not adopted.

- [rewrite-local-protection-v1](rewrite-local-protection-v1/README.md):84 raw generations personally inspected. Qwen direct→protected25→28/28 faithful and usable; both9/9 repairs.25 unencoded request/body pairs are identical. Residual conversational particles are a minor recorded style issue, not a perfect-fluency claim. Granite19/28 faithful,18/28 usable,7/9 repairs; actual empty EOS and hostile OCR instruction following retained. The main known-development gate passes for Qwen protected, but the separately frozen8-case stress fails. No qualified selection, fresh24 generation or production adoption.

- [rewrite-local-protection-stress-v1](rewrite-local-protection-stress-v1/README.md):16 raw generations, all maps and restored/null bodies personally inspected. Direct→protected faithful1→4/8, usable0→3/8. Protected has2 protocol failures and0/2 delivered repairs; direct has2/2 spelling repairs but other meaning errors remain. Multiple terms, literal quotation and explicitly cash-not-cache meaning still fail; repeated terms become duplicate tasks. Both protocol-rejected raw bodies are also incorrect: zero observed false rejection of a correct native body. Missing-marker counts alone never establish semantic failure.

## Current rewrite evidence

- [rewrite-current-task-v1](../reproduction/rewrite-current-task-v1/README.md)
- [rewrite-system-task-probe-v1](../reproduction/rewrite-system-task-probe-v1/README.md)
- [rewrite-cleanup-preflight-v1](../reproduction/rewrite-cleanup-preflight-v1/README.md)
- [rewrite-natural-source-audit-v1](../reproduction/rewrite-natural-source-audit-v1/README.md)
- [rewrite-timing-audit-v1](../reproduction/rewrite-timing-audit-v1/README.md)

Current-task108 and system-probe12 native outputs are fully inspected. User-tail rules can be copied as editable text; trusted system placement fixes four regressions but returns only to8/12 faithful and7/12 scene-ready on the same hard subset. No candidate qualifies, and the mandatory36-case pool and fresh24 are not reduced. Removing rejection alone does not repair wrong native content. Original request messages and restored bodies are both visible in the living HTML.

The actual production-cleanup source replay changes one36-case input; prior raw-direct experiments remain unchanged and correctly scoped. Natural-ASR preparation verifies120 historical public read-speech sources, not new candidate inference or independent held-out data. Timing audit flags9/396 historical observations without attributing a sleep/runtime cause or approving latency. Minimum target remains Apple Silicon16GB; the M4/32GB host does not substitute for it. Original-language/mixed-input and native-chain acceptance remain pending.

## Cleanup-aligned rewrite diagnostics

- [rewrite-private-spans-v1](../reproduction/rewrite-private-spans-v1/README.md)
- [rewrite-natural-text-review-v1](../reproduction/rewrite-natural-text-review-v1/README.md)

## Input-object and2B artifact diagnostics

- [rewrite-data-object-v1](../reproduction/rewrite-data-object-v1/README.md)
- [rewrite-2b-precision-v1](../reproduction/rewrite-2b-precision-v1/README.md)

- [Bounded-edit v1](rewrite-bounded-edit-v1/README.md):36 actual selection/format pairs, raw observations and full review.
- [User spelling alignment](rewrite-spelling-alignment-v1/README.md):separate source-bound retrospective review of25 historical failures; no inference.

- [Bounded-edit v2](rewrite-bounded-edit-v2/README.md), [formatter v3](rewrite-formatter-v3/README.md) and [Metal probe](rewrite-metal-preflight-v1/README.md):109 new raw responses fully reviewed; no candidate production adoption.

- [rewrite-metal-regression-v1](./rewrite-metal-regression-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-natural-format-v1](./rewrite-natural-format-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-language-profile-v1](./rewrite-language-profile-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-language-profile-v2](./rewrite-language-profile-v2/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-cleanup-copula-v1](./rewrite-cleanup-copula-v1/README.md): actual native or deterministic evidence; source-bound reviews and replay instructions.

- [rewrite-metal-chain-v1](./rewrite-metal-chain-v1/README.md): continuous actual Go cleanup, conditional Metal spelling selection and source-only formatting; all outputs reviewed separately from release acceptance.

- [rewrite-metal-prefix-v1](./rewrite-metal-prefix-v1/README.md): complete actual174-response cache-on parity/quality and source-bound timing, with initial misses retained.

- [rewrite-metal-primed-v1](./rewrite-metal-primed-v1/README.md):380 actual measured/startup outputs, both source orders, explicit readiness costs and limited canary controls.

- Full historical401-source cleanup/selection/formatting regression: [reproduction](../reproduction/rewrite-history-chain-v1/README.md).465 actual responses; synthetic281/natural120 separated; no output filtering or release adoption.

- Full401-source formatter prompt revision: [reproduction](../reproduction/rewrite-history-format-v2/README.md).401 actual final bodies reviewed; prior cleanup/64 choices fixed. Improvements and new background-fact/repetition regressions retained.

- Work-scope v3/v4: [v3](../reproduction/rewrite-work-scope-v3/README.md), [v4](../reproduction/rewrite-work-scope-v4/README.md).126 plus110 actual responses, all reviewed; background/status, unchanged details, owner/date layout and16 synthetic controls.

- Tone-insensitive Chinese spelling and complete401-history/24-control chain: [reproduction](../reproduction/rewrite-tone-chain-v1/README.md).538 actual responses reviewed; one correction gained and one harmful substitution introduced, rejected.

- [Semantic selector policy](rewrite-selector-semantic-v1/README.md): all609 raw responses, complete outcomes, reviews, prompt identity caveat and rejected selection.

- [Concise semantic selector](rewrite-selector-semantic-v2/README.md):457 actual cleanup attempts,101 choices and457 final bodies; all reviewed, not adopted.

- [Candidate sentence previews](rewrite-selector-preview-v1/README.md):457 actual source attempts,558 responses, all reviewed; paired raw evidence and rejected development gate.

- [Four-arm selector screen](rewrite-selector-screen-v1/README.md):404 actual choices,457 Go preparations,1828 outcomes including1424 source-treatment skips; all reviewed, no arm promoted.

- [Granite native-runtime diagnosis](rewrite-selector-model-v1/README.md)
- [Granite CPU-layer selector comparison](rewrite-selector-model-cpu-v1/README.md)

- [Public referent-description selector experiment](rewrite-selector-referent-v1/README.md)

- [Source-last selector serialization](rewrite-selector-source-last-v1/README.md):404 actual choices; source-last increases harmful substitutions in both paired arms, rejected.

- [Counterbalanced whole-sentence selection](rewrite-selector-sentence-choice-v1/README.md):404 actual choices, harmful19/5 in A/B candidates, both rejected; raw labels/normalized edits and allowed normalization differences retained.

- [Granite1.46B hybrid comparison](rewrite-selector-hybrid-v1/README.md):12 native compatibility probes plus202 identical-request choices; harmful20/19 despite47/47 required repairs, both rejected.

- [Trusted instruction-language comparison](rewrite-selector-instruction-language-v1/README.md):404 actual choices across two current small models; only trusted system text changes. All English variants increase harmful choices and are rejected.

- [Complete-reference message separation](../reproduction/rewrite-selector-reference-turn-v1/README.md):404 actual choices; full OCR preserved in separate reference turns; both variants rejected after complete review.

- [Unverified unique-label exposure](../reproduction/rewrite-selector-unique-label-v1/README.md):404 actual choices; only the unique-status field is omitted; all4 changed choices regress, both variants rejected.

- [Added semantic contrast demonstrations](../reproduction/rewrite-selector-contrast-demo-v1/README.md):404 actual choices after zero-inference native token-budget checks; one improved and five regressed choices, both variants rejected.

- [MiniCPM5-1B comparison](../reproduction/rewrite-selector-minicpm-v1/README.md):202 unchanged-request choices plus12 native responses; both configurations rejected. One interrupted native-log completion is retained separately, with no fabricated response credit.

- [MiniCPM official/native token IDs](../reproduction/rewrite-minicpm-tokenizer-v1/README.md):202 complete prompt sequences match exactly; zero added model generations, not semantic or numerical-weight equivalence.

- [MiniCPM F16 export comparison](../reproduction/rewrite-selector-minicpm-f16-v1/README.md):202 unchanged-request choices plus12 native responses; more required repairs but15/17 harmful choices, both configurations rejected.
- [Youtu1.96B raw evidence](rewrite-selector-youtu-v1/README.md):12 native and202 selector responses, full reviews and same-request controls; no new final bodies.
- [LFM2.5 Q8 raw evidence](rewrite-selector-lfm25-v1/README.md):12 native and202 selector responses, byte-preserved logs, full reviews and controls; no final bodies.

- [Existing ten-example order](../reproduction/rewrite-selector-demo-order-v1/README.md):404 actual choices; moving keep examples last reduces harmful edits but loses supported repairs.3 improvements,23 regressions,1 allowed change; both treatments rejected.

- [Native d1 decision packaging](../reproduction/rewrite-d1-evidence-v1/README.md):72 actual typed responses over six sources/six layouts, all reviewed. Native96/45 piece clipping identified; full-policy correction remains3/6 per backend. No final bodies or production adoption.

- [Selected rewrite product integration](rewrite-product-integration-v1/README.md) — adopted GGUF worker, exact prompt/producer parity, actual output review and retired-weight receipts.

- [Independent prompt authoring](rewrite-independent-prompts-v1/README.md) — explicit directory/model/permission provenance, original author responses and discarded attempts.

- [Qwen3-ASR 1.7B and accessibility](qwen17-accessibility-v1/README.md) — original protocol failure, corrected full comparison, paired actual window reads, combined audio/text outputs and retained verification failures.

- [Qwen ASR size and quantization diagnostic](../results/qwen-asr-speed-v1/README.md) — same-development-corpus native 0.6B/1.7B INT8 and separate MLX 8-bit/4-bit compute/accuracy evidence.

- [Native Qwen3-ASR migration acceptance](../results/native-asr-v1/README.md) — native Swift model loading, offline signed-package inference, live coordinator replay and explicit remaining platform/performance gates.


- [IME bundle-directory exchange](ime-bundle-swap-v1/README.md): native arm64 and initial x86_64-process filesystem observations; no IME acceptance claim.
