# Rewrite reliability and competitor alignment

## Evidence boundary

This investigation addresses colloquial ASR text reaching the target unchanged. It rehashed installed Doubao IME 1.0.2 and the retained extracted Chatterfly 1.0.6.13398 executable; both match the existing ledgers. Matching REA evidence was reused rather than decompiling the same binaries again. [Evidence and aggregate results](evidence/rewrite-alignment.json) distinguish REA IDs from ordinary static observations. No competitor was launched, no endpoint was requested and no private database or recording was read.

| Boundary | Doubao | Chatterfly | Arion |
| --- | --- | --- | --- |
| Recognition versus edited output | Original and organized text, rewrite decision and delivery context (`CR-DB-coordinator`) | `asrStableText` enters a separate post-processor; `enhancedText` reaches final delivery (`CF-asm-llm-orchestrator`, `CF-asm-do-post`) | Raw recognition, corrected source, proposal and delivered text are distinct |
| Result lifecycle | Pending/no-rewrite/will-rewrite; result and transport completion are distinct named states (`CR-DB-stream`, strings/reflection only) | Typed scene/CMode/pause/error packets; provisional deltas and finished text are distinct (`CF-asm-sse-dispatch`, instruction evidence) | Preview validation, final validation and target ownership are separate gates |
| Context | Cursor/dialogue conditioning exists for recognition; rewrite request context remains unresolved | Stable ASR text plus populated app/field/cursor context and configuration enter a remote-capable stage | Local scene selection and approved spelling terms; full context is validation evidence |
| Semantic quality | Unknown: no controlled outputs or server prompts recovered | Unknown: no controlled outputs or server prompts recovered | Actual local text outputs measured below; no competitor quality parity claim |

See the bounded [Doubao comparison](competitor-rewrite.md) and [Chatterfly reconstruction](chatterfly.md). Similar client stage boundaries do not establish equivalent rewriting models or quality.

## Confirmed Arion defects and changes

1. **Useful edits were rejected.** The validator counted every filler/repetition character toward its 70% content-retention check. Removing speech padding could fail that check and cause the source to be inserted. Deletion checks now discount a narrow set of clause-leading discourse markers and repeated demonstratives. They do not change the delivered text themselves. Quoted spans remain content for this allowance, and an all-padding utterance retains the original checks. Fact deletion, new content/numbers and negation checks remain active.
2. **Loading looked like disabled polishing.** The bootstrap policy previously required worker readiness at session start. An enabled preference could therefore skip the stage with an empty outcome. The policy now preserves the user's choice; a missing worker produces `unavailable` with exactly one source-text delivery. This records a failure; it does not imply that a visible UI explanation was added.
3. **Stronger prompts could lose uncertainty.** Exploratory prompts removed a hedge or damaged a question/self-correction. They were discarded; the template shipped for this investigation was `live-v2`. [ADR 018](../decisions/018-staged-gguf-rewriting.md) later replaced that template. A new presence-based uncertainty proxy blocks losing recognized hedges, allows synonymous/stuttered hedges, and scopes explicit corrections to certainty to the immediately superseded clause. This is a lexical proxy, not a semantic guarantee.

The baseline reproduced both unchanged filler-heavy proposals and good edits rejected as `content_dropped`. No diagnostics were present at the standard installed-app log path, so a particular user utterance's runtime branch remains unverified.

Regression tests cover cleanup versus dropped facts, all-padding input, uncertainty synonyms/stutters/corrections, unrelated hedges, quoted padding, polite questions and exactly-once source delivery with a missing worker. Core race tests, core/desktop tests and vet, and repository documentation/layer checks pass on macOS. The installed app has not been replaced or restarted by this investigation.

## Matched local diagnostic

Both sides used the same Qwen3 1.7B INT4 model, `live-v2`, greedy non-thinking decoding, four CPU threads, rule-corrected input and the production isolated worker on Apple M4. Each fixture includes the live-prefix schedule followed by the final request. Baseline binaries used the original prompt/validator through a build overlay; adopted binaries used the modified validator. Private manifests retain model/runtime/tokenizer, corpus and raw-row hashes.

| Text set | Cases | Baseline delivered proxy passes | Adopted delivered proxy passes | Improved / regressed against baseline |
| --- | --- | --- | --- | --- |
| New development challenges | 16 | 7 | 9 | 2 / 0 |
| Historical regression challenges, generated cases only | 29 | 26 | 27 | 1 / 0 |

These are exact-string constraint checks, not semantic accuracy percentages. The new set includes dense Mandarin/English scaffolding, repetition, conditional statements, references, quoted wording, uncertainty, corrections and technical identifiers. Five preservation controls pass on both sides. Historical self-correction checks remain 8/8. One historical verbatim row is excluded: the text harness applies rule correction before its verbatim bypass, so that row cannot establish verbatim acceptance. Product verbatim behavior is not inferred from it.

The adopted new set still has seven proxy failures. Six proposals pass fidelity validation while missing a cleanup constraint; another switches English into Chinese and is rejected. This confirms remaining generation limitations. Raising acceptance or globally stripping words such as `那个`, `就是`, `可能` would not resolve them safely. Three exploratory prompt variants were rejected because they worsened at least one existing case; their development gains are not presented as an adopted result.

This is a single-author synthetic text diagnostic, with no natural speech, independent blind review, audio recognition, native preview rendering or target insertion. Existing-app activity and power were not isolated, so timings are diagnostic only. It does not establish release acceptance or measured parity with either competitor.

## Multilingual disfluency follow-up

The user's examples separate hesitation (`呃`), an explicit repair (`啊不对`) and context-dependent padding (`那个`). Removing a repair marker must also apply the correction; deleting it alone leaves both contradictory versions. A demonstrative such as `那个按钮`, a comparison such as `I like`, and quoted wording remain content.

A further 21 frozen developer-authored text probes cover Mandarin/English and exploratory Spanish, French and Japanese. This probe uses exact required/forbidden/protected strings only; no unsupported-language CER/WER is computed. It runs the same production worker and prompt, with the same model and thread settings as the comparison above. [Aggregate evidence](evidence/multilingual-disfluency.json) includes hashes; full fixtures, outputs and model/runtime manifests remain private.

| Language | Cases | Before rule fix | After rule fix |
| --- | --- | --- | --- |
| Mandarin | 6 | 4 | 5 |
| English | 6 | 2 | 3 |
| Spanish, exploratory text only | 3 | 1 | 1 |
| French, exploratory text only | 3 | 1 | 1 |
| Japanese, exploratory text only | 3 | 1 | 1 |

Two failures originated before the LLM: deterministic filler removal deleted the explicitly quoted `呃` and `um`. `arion-rules-v2` now protects quoted spans using the same quote parser as the rewrite deletion allowance. The ambiguous Han character `额` requires both token boundaries, preserving nouns such as `余额` and `名额`; unambiguous connected hesitation handling stays unchanged. The probe improves 2 cases with no regression. Domain tests cover quoted/unclosed wording, apostrophes versus single quotes, surrounding unquoted fillers, ordinary nouns and preserved demonstratives/repair text.

Remaining failures distinguish generation from validation. The Mandarin `啊不对` correction passes, but the `那个` question is correctly rewritten and then rejected by the deletion proxy. English `no wait`, `I mean` and parenthetical `like` remain unchanged in these 1.7B outputs. Some Spanish/French/Japanese corrections are generated correctly but rejected because the repair/deletion proxies are tailored to Mandarin and English. The lexical uncertainty/negation proxies also do not establish protection for those other languages. No foreign-language word lists or recognition-language claims were added.

An exploratory 4B run on the same **pre-fix** inputs passes 13/21 versus 9/21 for 1.7B, including the three English cleanup failures. It still loses the pre-processing quotes and encounters other-language validator failures. This small text probe is insufficient to adopt a larger default. Prompt and default model remain unchanged. The historical generated-case regression check remains 27/29 after the rule fix, with no changed final outputs versus the previous adopted run.

Per-language release evaluation must pair hesitation/repetition/repair cases with meaning-bearing uses of the same expressions, negation, uncertainty and quoted controls. A table of additional stop words cannot establish contextual cleanup or multilingual fidelity. Foreign-language text probes here are diagnostic only: no corresponding audio recognition, native delivery or independent fluent-rater review was run.

## Question status and next gates

| Question | Status | Remaining evidence |
| --- | --- | --- |
| Do competitors separate recognition, rewrite and delivery? | Answered within the recorded static boundaries | Runtime defaults and exact server behavior remain unknown |
| Can Arion reject legitimate cleanup and then insert the source? | Reproduced and repaired for the declared padding cases | Natural-speech coverage and bounded false-rejection analysis |
| Can enabled polishing be skipped while loading? | Reproduced in policy tests and repaired | Packaged-app loading scenario and visible failure presentation |
| Is broad oral-to-written rewriting reliable? | Unresolved; failures remain | Freeze real-ASR examples, compare prompt/model candidates without regressions, then independent blind meaning/style review |
| Has actual competitor output parity been demonstrated? | Unresolved | Same consented speech fixtures and controlled target capture on all products |

P0: keep raw ASR, proposal, verdict and delivered text distinguishable in evaluation; collect consented natural dictation cases covering both colloquial cleanup and meaning preservation. Compare candidate prompts/models against a frozen baseline, including self-corrections, questions and technical text. P0: independently review meaning and style before adopting a new default. P1: run packaged offline microphone-to-editor scenarios, loading/failure cases and actual competitor fixture comparisons. These gates extend [009](../../specs/009-contextual-rewrite.md), not the static competitor evidence.

## Doubao-aligned implementation follow-up

Arion now separates a local speech deletion stage (`domain.CleanSpeech`, `speech-spans-v1`) from model organization. The rule producer emits typed spans in the unchanged source; the renderer checks ordered UTF-8 ranges and quoted/code protection, and declines deletions that would join separate Latin words. It handles declared hesitation/scaffolding, limited stutters and repairs with a repeated frame or matching typed value. Ambiguous words remain input to the unchanged `live-v2` / Qwen3 1.7B organizer. This follows Doubao's observed stage boundary, without claiming its unrecovered annotation model or measured output parity.

Organization rejection, timeout and worker unavailability retain completed local cleanup and their explicit failure outcome. Raw recognition, corrected source, annotations, proposal and delivered text stay distinct. The text benchmark also bypasses correction for verbatim controls. `arion-rules-v3` preserves repeated first-person `I` for the separate repetition stage, instead of joining it into `II`, and protects quoted spelled letters from acronym joining. Real audio review exposed this pre-existing merge and a blind spot in the original exact-string annotations; the original test set is treated as regression after that review, and a new set is frozen before final inference.

An experimental generative span annotator was rejected for omissions and erroneous date deletion. The shipped span producer is deterministic, adds no model request and does not change model weights or the organizer prompt. [The implementation benchmark](../../benchmarks/results/disfluency-alignment-v2/README.md) retains matched build settings, model/runtime/source/row hashes, initial test exposure, fresh text tests, real ASR/TTS runs, failure cases and native KV-reuse parity. These diagnostics do not replace independent meaning review, natural speech or native target insertion.

## Installed-worker rejection diagnostic

[The installed-worker diagnostic](../../benchmarks/reproduction/rewrite-rejection-v1/README.md)
retains 24 authored inputs, actual generations and production validator results.
Twelve simple probes passed. Twelve follow-up probes produced eight accepted
outputs, two truncated answers to the dictated question and two enumerations
that lost requirements and negation. The same runaway answer's output allowance
rose from 128 to 278 tokens with history: the current budget counts context in
the variable user turn rather than only dictated text. No production policy was
changed. These development observations explain reproducible failure mechanisms,
not the exact candidates behind historical support events, which omit text.
Current-worker `markup` and `length` failures were not reproduced in this set.

## Scene structure versus literal fidelity

[The scene audit](../../benchmarks/reproduction/scene-rewrite-audit-v1/README.md)
separates four installed-worker generations from five authored validator probes.
Numbered work plans and email paragraphs pass, so formatting differences alone
are not rejected. Work/prompt scenes, however, currently select the generic chat
layout unless explicit spoken enumeration selects list; no dedicated work/prompt
semantic organization instruction reaches the model. A coding enumeration also
remained inline despite a list request.

The scene-independent lexical validator rejects diagnostic paraphrases and
headings as unsupported content, `five` to `5` as a new number, and `先别发` to
`暂缓发布` as dropped negation in the retained probes. The desired product
behavior permits reordering, grouping and rewording while preserving facts and
intent. This needs both appropriate scene instructions and evaluation of
meaning-preserving transformations; lexical overlap alone cannot establish it.
Chatterfly's inspected client success path does not contain these ratio gates,
but its service-side prompts and validation remain unknown.

## Latest small-model ambiguity diagnostics

[2B contrast](../../benchmarks/reproduction/rewrite-2b-contrast-v1/README.md) retains successful owner grouping and email paragraphs but both packages still guess a function under same-pronunciation ambiguity (15/16 faithful each). [Explicit pronunciation evidence](../../benchmarks/reproduction/rewrite-pronunciation-evidence-v1/README.md) improves faithful outputs20→21/24 and known-answer repairs6→7/7. All48 raw bodies were inspected without content validation. Three ambiguous cases still choose an unsupported identifier even when the input marks multiple candidates. No package passed its frozen development gate; new evaluation inputs remain ungenerated.

The CMU producer supplies text-derived English hypotheses, not acoustic evidence. It omits unknown dictionary candidates and does not cover Chinese names or underscore identifiers. It remains experimental. Identical-request negative controls show serial timing drift, so raw time differences do not isolate prompt cost. Minimum supported target is Apple Silicon16GB; current M4/32GB observations are not minimum-device acceptance. Production prompt, validator and insertion behavior remain unchanged.

## Source-plan package diagnostic

[All56 source-plan comparison outputs](../../benchmarks/reproduction/rewrite-source-plan-v1/README.md) were inspected, including raw JSON and constructed bodies. Required unit keys and pronunciation-bound anchors retain three ambiguous English spans but do not guarantee semantic completeness or natural organization. Direct body→plan gives24→19/28 faithful,20→6/28 scene-ready and8→0/9 required repairs. All28 plans are valid and untruncated, so protocol validity must not be reported as rewrite quality. This package changes prompting, demonstrations, segmentation, grammar and rendering together; no general claim against structured generation follows. It failed the frozen gate; fresh24 remain ungenerated and production is unchanged.

The next bounded direction should preserve direct natural generation and test lighter local protection of unresolved terms. [A non-Qwen small-model acquisition](../../benchmarks/reproduction/rewrite-other-small-models-v1/README.md) verifies Granite1.46B weights/provenance. Its subsequent compatibility and quality observations are reported in the local-protection/model screen below. No4B or older Qwen inference is planned.

## Local term protection and new stress cases

[The local-protection/model screen](../../benchmarks/reproduction/rewrite-local-protection-v1/README.md) adds84 fully inspected generations. Updated shared scene examples structure multiple requirements and remove an initial conversational filler. Within Qwen, opaque encoding is used only for explicit same-pronunciation ties;25 unencoded requests and bodies are identical. Qwen direct→protected gives25→28/28 faithful/usable and9/9 repairs in both. Residual conversational particles remain a minor style caveat. Granite1.46B gives19/28 faithful,18/28 usable and7/9 repairs; it drops deadlines, emits one empty body and follows hostile OCR once. Native/exported chat templates match upstream; empty content is genuine EOS. This is not a universal claim about all Granite precision/prompt packages.

[The8-case stress supplement](../../benchmarks/reproduction/rewrite-local-protection-stress-v1/README.md), frozen before its first inference, adds16 fully inspected bodies. Protected output is faithful4/8 and usable3/8, with2 explicit protocol errors. Multiple ambiguity, literal quotations and misleading unique OCR still fail; repeated mentions produce duplicate tasks. The2 rejected raw bodies themselves contain errors, so no observed false rejection is claimed. A missing marker alone would not establish semantic failure, and a valid marker does not establish semantic correctness. No package qualifies; fresh24 remain ungenerated and production is unchanged.

Next work must cover all36 exposed main/stress cases before broader claims. Preserve free natural generation, test literal/semantic precedence and multi-term behavior, and consider a smaller current-input task wrapper before adding more global rules. Do not hide failures by source fallback, narrowing the corpus or removing OCR. Minimum16GB device, natural ASR, independent ratings, native safety and real stop-to-insertion latency remain separate gates.

## Task-boundary and source alignment diagnostics

- [rewrite-current-task-v1](../../benchmarks/reproduction/rewrite-current-task-v1/README.md)
- [rewrite-system-task-probe-v1](../../benchmarks/reproduction/rewrite-system-task-probe-v1/README.md)
- [rewrite-cleanup-preflight-v1](../../benchmarks/reproduction/rewrite-cleanup-preflight-v1/README.md)
- [rewrite-natural-source-audit-v1](../../benchmarks/reproduction/rewrite-natural-source-audit-v1/README.md)
- [rewrite-timing-audit-v1](../../benchmarks/reproduction/rewrite-timing-audit-v1/README.md)

Current-task108 and system-probe12 native outputs are fully inspected. User-tail rules can be copied as editable text; trusted system placement fixes four regressions but returns only to8/12 faithful and7/12 scene-ready on the same hard subset. No candidate qualifies, and the mandatory36-case pool and fresh24 are not reduced. Removing rejection alone does not repair wrong native content. Original request messages and restored bodies are both visible in the living HTML.

The actual production-cleanup source replay changes one36-case input; prior raw-direct experiments remain unchanged and correctly scoped. Natural-ASR preparation verifies120 historical public read-speech sources, not new candidate inference or independent held-out data. Timing audit flags9/396 historical observations without attributing a sleep/runtime cause or approving latency. Minimum target remains Apple Silicon16GB; the M4/32GB host does not substitute for it. Original-language/mixed-input and native-chain acceptance remain pending.


## Cleanup-aligned private spans

[The full36×2 regression](../../benchmarks/reproduction/rewrite-private-spans-v1/README.md) uses actual production CleanSpeech observations in both model-input arms, retaining raw source and deletion traces independently. Private literal/unresolved-span maps, shared repeated tokens and reduced derived-spelling visibility improve faithful outputs31→34/36 under the frozen conservative naming rule, scene-ready30→34/36 and required repairs10→11/11. Every72 raw native and restored output was inspected; no output rejection or source fallback occurs.

The candidate adds unrelated screen function text to an ordinary cash-balance message, a definite new regression. On another case it preserves the explicit cash-not-cache definition but changes the identifier to readCache. That is separately marked as normalization-sensitive under the original rubric, not misreported as a definition reversal. The later confirmed policy permits same-letter readCash normalization, but this readCache replacement remains unsupported. No release qualification follows. This is a complete packaging treatment: always-added token policy and reference key casing change alongside private maps; causal isolation of an individual technique is not claimed. Fresh24 remain ungenerated; production remains unchanged.

[The120-pair natural-source text audit](../../benchmarks/reproduction/rewrite-natural-text-review-v1/README.md) finds ASR omissions of names, numeric digits, negation and long suffixes, and apparent artifacts in published transcripts. Source-faithful editing and evidence-supported speech recovery require separate judgments; do not require hidden-reference reconstruction or inject reference-derived OCR as if it were a real screen. All120 actual production-cleanup outputs remain unchanged. No new ASR/LLM generation, independent ratings, original-language preservation or minimum16GB/native-chain acceptance is implied.


## Input objects and higher-precision2B artifact

[All72 input-format outputs](../../benchmarks/reproduction/rewrite-data-object-v1/README.md) were inspected. Lowercase reference keys remain34/36 faithful/usable with11/11 repairs, but retain unrelated OCR content and reverse a definition previously retained. Equal aggregate scores therefore hide a material severity change. The uniform JSON-input package is31/36 faithful/usable and9/11 repairs: one OCR addition disappears, two recipient repairs and distinct multi-term/conditional targets regress. Eight unchanged no-context requests and outputs remain byte-identical. Neither package qualifies; fresh24 are ungenerated.

[All36 Q8 artifact outputs](../../benchmarks/reproduction/rewrite-2b-precision-v1/README.md) use the identical36 requests/encodings as reused Q4 private baseline. Actual tensor count remains1,881,825,088, with2,012,012,800 weight bytes. Q8 gives32/36 faithful/usable and9/11 repairs versus34/36 and11/11. It removes the unrelated OCR declaration but changes conditional targets, misses recipient/buildCache corrections and reverses cash/not-cache meaning. All36 prompt-token counts match, but numeric base-weight/converter lineage is not independently proven; do not claim isolated causal quantization effects. Maximum sampled workerRSS is4,418,600,960 bytes, not peak or whole-app/ASR memory. M4/32GB does not establish minimum16GB acceptance; serial/clock limitations preclude causal speed/SLO claims.

All108 new bodies, maps and restored text are personally inspected without output rejection or source fallback. Reused baseline36 is referenced once, never added to new-generation totals. Production is unchanged. These failures motivate testing whether bounded source-relative spelling edits and separate natural formatting reduce context substitution; that would be an explicit research architecture with disclosed full-OCR flow, not an adopted product decision. Preserve all36 regressions, necessary repairs, natural structure and remaining language/native/independent/minimum-device gates.

## User spelling alignment and bounded-edit study

The human now explicitly permits read cash→readCash while rejecting unsupported readCache. The [separate aligned review](../../benchmarks/reproduction/rewrite-spelling-alignment-v1/README.md) changes16 of25 historical failures, leaving9 incorrect bodies. Qwen3.5-2B's earlier24-case screen is24/24 faithful under this user policy, rather than23/24. Earlier blanket claims that choosing readCash proves ambiguity failure were too strict. Original reviews and selections remain available; this does not prove broader production readiness.

[Bounded-edit research](../../benchmarks/reproduction/rewrite-bounded-edit-v1/README.md) adds72 actual native2B-Q4 outputs:36 spelling decisions with full OCR and36 source-only natural formatting bodies. All raw outputs, hypotheses and mappings inspected. Selections33/36 conform with8/11 necessary repairs; final bodies28/36 faithful,27/36 scene-ready,6/11 repairs. Grammar cannot prevent formatting from dropping corrected recipients or changing a restriction/negative question. Owner grouping and leave-email anchors succeed. No output rejection/source fallback; fresh24 unused, production unchanged.

The full pypinyin character-reading table is a fixed general resource, not a test-name vocabulary or acoustic proof. A separate zero-inference producer audit finds missing English/mixed identifier subspans inside longer English runs. Next research must combine the user-aligned spelling policy with balanced positive repair examples, source-name preservation during formatting and broader language coverage. Minimum AppleSilicon16GB, independent natural quality and native/concurrent latency acceptance remain open.

## Visible-source refinement and provider capability

[Bounded-edit v2](../../benchmarks/reproduction/rewrite-bounded-edit-v2/README.md) uses the confirmed source-letter normalization rule and English subspans, balances positive selector examples and exposes complete corrected source identities to formatting. All72 outputs reviewed:36/36 intermediate choices conform, all11 local spelling repairs survive; final34/36 faithful,33/36 scene-ready. Corrected recipients, repeated-task restriction, named definition and negative question recover. One formatter narrows checking a function to its records; another drops explicit not-forty-seconds emphasis while retaining exact four seconds. These are distinct criteria, not incorrect canonical spelling or a changed numeric value.

[Formatter v3](../../benchmarks/reproduction/rewrite-formatter-v3/README.md) reuses actual selector rows and changes only the formatting prompt recipe:36 new bodies, original edited sources36/36 identical. Scope narrowing disappears, oral polish improves; frozen explicit-emphasis rubric35/36 faithful/ready, all11 local repairs and all11 repaired messages faithful. The one omitted-emphasis case is separately flagged pending user preference about semantically redundant contrast. Original scores/selections remain unchanged; fresh24 unused. Logical quantity preservation must not be confused with a new permission or missing independent negation.

CPU two-call v2 p95≈11.15s is not release latency. A [native Metal capability probe](../../benchmarks/reproduction/rewrite-metal-preflight-v1/README.md) requests the same2B weights and same owner message; one real response≈1542ms preserves grouping but differs in wording from CPU. Separate load-only verbose logs observe25/25 GPU offload. Initial quiet process lacks per-request placement telemetry; later load evidence does not retroactively supply it. One formatting request is not p95/full pipeline, parity, concurrent ASR/native or minimum16GB acceptance. Full provider quality/performance remains necessary.

[Recent small-model metadata](../../benchmarks/artifacts/rewrite-model-refresh-v1/README.md) pins official LiquidAI1.2B model/export cards and custom LFM1.0 terms. No weights acquired or inference; published parameters/languages are author claims, not measured tensor counts, Arion quality or distribution approval. Production and all independent/natural/language/native/minimum-device gates remain unchanged.

## Native language routing and natural-source review

[Metal identical-request regression](../../benchmarks/reproduction/rewrite-metal-regression-v1/README.md) retains35/36 faithful/usable bodies and11 repairs;34 bodies match CPU bytes. Every actual worker records25/25 offloaded layers. This establishes provider coverage for these requests, not full-pipeline or minimum16GB performance.

[The first natural formatter experiment](../../benchmarks/reproduction/rewrite-natural-format-v1/README.md) personally reviews120 historical public ASR text sources:109 faithful,8 incorrect,3 uncertain,72 usable,116 original-language outputs. An [English policy and script-profile recipe](../../benchmarks/reproduction/rewrite-language-profile-v1/README.md) improves this to117 faithful,2 incorrect,1 uncertain,73 usable and118 language-preserving outputs. It introduces a Chinese-to-English translation despite fixing earlier failures.

[Source-script policy and demonstration routing](../../benchmarks/reproduction/rewrite-language-profile-v2/README.md) produces119 verified faithful outputs,0 definite failures,1 uncertain and75 usable; all120 retain source language. Single-script inputs receive matching-language trusted instructions and examples; mixed inputs retain the full source. OCR and original source are not cropped. The same36 development probes retain all meaning and11 repairs, but one explicit numeric self-correction remains spoken rather than polished. These are three recipes on the same120 sources, not360 independent subjects. There is no new ASR, screen capture, hidden-reference reconstruction or independent human assessment. Copied garble/fragment is not automatically usable; af→afraid remains uncertain. No output validator, rejection or source fallback hides these bodies. Production model, prompt, provider and validator are unchanged.

## Explicit copular speech repair

The production deletion producer is now `speech-spans-v2`. The prior typed-value matcher failed when an explicit Chinese repair cue was followed by a copula: “四十秒，不对，是四秒”. The scoped change accepts the complete replacement clause with the same unit, deleting the earlier value/cue/copula using original UTF-8 byte spans. Negated or uncertain replacements, unrelated predicates, mismatched units, quotations and untyped entities are excluded. Explicit “四秒，不是四十秒” remains intact.

[Actual Go evidence](../../benchmarks/reproduction/rewrite-cleanup-copula-v1/README.md) replays156 inputs with source snapshots, red-before-fix tests, passing domain/full-core tests and fuzz logs. Exactly two source bodies change; all120 natural sources remain unchanged. This is zero LLM/ASR generation. Existing frozen inference inputs are not modified; the new cleanup has not yet been evaluated through the complete native two-call pipeline. Independent quality, real OCR/ASR, all scenes/languages, native lifecycle, concurrent end-to-end latency and AppleSilicon16GB gates remain open.


## Actual conditional Metal chain

[The complete chain diagnostic](../../benchmarks/reproduction/rewrite-metal-chain-v1/README.md) executes current speech-spans-v2 through an actual persistent Go JSONL worker before each case.18 nonempty candidate sets invoke new actual Metal spelling decisions;138 empty sets skip that model call uniformly. All156 cases invoke language-profile-v2 formatting, yielding174 new native model responses. These are actual stage observations, not reused CPU selections or invented empty model responses. All174 outputs, source/OCR, cleanup spans and source-relative edits were personally reviewed.

Development36/36 are faithful and usable with11/11 required repairs; all18 actual selector decisions conform. Natural120 remain119 verified faithful,1 uncertain and75 usable, with all120 retaining original language.155/156 final bodies are byte-identical to the previous formatter-only study. The one improvement is complete removal of the explicit copular numeric withdrawal; four seconds, concurrency and other-configuration prohibition remain. This establishes composition coverage on exposed inputs, not broad new quality improvement or independent validation. Copied garble and af→afraid uncertainty remain visible.

Continuous per-case monotonic cleanup-IPC-to-body latency has developmentp50≈1.87s,p95≈2.92s and naturalp50≈1.51s,p95≈2.41s. Startup is separate; these are neither explicit cold/warm strata nor ASR/native stop-to-insertion measurements.174 native stage-timing sums do not exceed their outer request measurements under the checked1ms tolerance, but that does not settle historical flags or validate a latency SLO. Maximum sampled native workerRSS≈2.10GB is not peak or full-application memory. Current hostM4/32GB does not prove AppleSilicon16GB performance. No production model/provider/prompt/validator change, fresh24 generation or release approval follows. Next priorities are actual stage-overhead reduction with quality/parity controls, broader source/context and scene coverage, independent review, and native/minimum-device acceptance.

Stage timings identify repeated prompt processing as the next performance target: on development data, actual selection request median≈1348ms, prefill≈1226ms and decode≈102ms (18 calls); formatting median≈1169ms, prefill≈933ms and decode≈263ms (36 calls). These are separate medians, not an additive per-case decomposition. Prefix/session reuse requires a new pinned treatment with measured output parity, context isolation and full quality coverage; it is not enabled by this diagnostic.


## Prefix reuse on the complete chain

[Cache-enabled full-chain evidence](../../benchmarks/reproduction/rewrite-metal-prefix-v1/README.md) changes only cache_prompt in the actual174 source-bound requests; all174 normalized request payloads and all174 raw model-content strings match the uncached controls. All new outputs were personally reread. Native counters report164 cache hits and90,542 reused tokens. Development36 remain faithful/usable with11 repairs; natural120 remain119 verified,1 uncertain and75 usable. This is tested parity for one forward order, not reverse-order, cross-session or broad new semantic success. Exact b11734 code uses common-prefix reuse and truncation after divergence; code inspection and absence of prior-source additions are not a dedicated privacy acceptance test.

Observed continuous per-case developmentp50≈694ms,p95≈2420ms,p99≈2776ms; naturalp50≈312ms,p95≈685ms,p99≈764ms. All initial cache misses remain included. The uncached timing reference is historical serial, so these observations do not establish an isolated causal speedup. Maximum sampled workerRSS≈2.10GB is not peak/full-application/minimum16GB proof. First scene/route/selector requests remain slow; explicit prefix priming, its startup cost, output parity across orders and canary isolation require a separate frozen experiment. No production cache/provider/prompt/validator change, fresh24 consumption, native/ASR contention, independent ratings or release approval follows.


## Explicit general priming and input order

[The two-order priming experiment](../../benchmarks/reproduction/rewrite-metal-primed-v1/README.md) executes380 actual model responses:348 measured and32 startup warmups. Each scene worker primes a selector and all three script routes on declared synthetic inputs, including actual Go cleanup. No max_tokens=0 shortcut is claimed. All380 outputs were personally reviewed; each arm's174 measured raw strings match the uncached controls, and all174 matched stages agree across orders. Development36 retain meaning/usefulness and11 repairs; natural120 remain119 verified,1 uncertain and75 useful. These are repeated exposed sources, not independent subjects.

All174 measured calls per arm report native cache hits. Twelve positive synthetic markers per arm survive their warmup outputs and no marker occurs in156 measured bodies; this is a limited isolation control, not general privacy/native lifecycle proof. Forward developmentp95≈836ms,p99≈946ms; reversep95≈1470ms,p99≈1485ms. Naturalp95≈790/1058ms. The reverse arm is materially slower; fixed serial order and static host metadata do not identify its cause or establish stable latency. Measurements are cleanup-IPC-to-body after readiness, excluding ASR/native insertion.

Per scene worker, model startup plus priming is≈4.77–8.15s, with priming alone≈4.10–7.03s; this cost is explicitly retained, not hidden in uncounted warmups. Worker resets are diagnostic boundaries, not one persistent application workload. Maximum sampled native workerRSS≈2.15GB is not peak/full-application/minimum16GB evidence. Broader quality/context/scenes, actual audio/OCR/native lifecycle, concurrency, independent ratings and actual AppleSilicon16GB remain required. No production activation or fresh24 inference follows.


## Full historical source coverage

The [401-tuple regression](../../benchmarks/reproduction/rewrite-history-chain-v1/README.md) exercises all distinct source/full-context/requested-scene tuples from the prior report, excluding warmups and intermediate decisions. All465 actual responses were reviewed:64 spelling choices and401 unfiltered final bodies. Synthetic281 yields278 confirmed faithful,2 retained-criterion failures,1 uncertain and260 useful; historical natural120 yields119 confirmed,1 uncertain and75 useful. These exposed sources do not establish held-out quality.

The shared start-time scope narrows incorrectly in one structured work result. One English result drops an explicit not-twenty-seconds contrast while retaining the correct two-second value; emphasis scoring is disclosed separately from numeric correctness. Cancellation-request framing remains uncertain after documented second review; original labels and amendment are preserved.

Seven reviewed normalization opportunities remain: four cross-script Latin targets and one different-tone Chinese word produce no spelling hypothesis; two unique Chinese spelling candidates reach selection but are kept. This separates producer coverage from prompt behavior.51 supported spelling-edit cases survive correctly. Same-letter readCash normalization is permitted; unsupported readCache replacement is not. These are evaluator-selected opportunities, not a general correction recall denominator. No production prompt/provider/validator adoption or Apple Silicon16GB acceptance follows.


## Formatter-only prompt revision across all historical sources

The [full401-source comparison](../../benchmarks/reproduction/rewrite-history-format-v2/README.md) keeps actual prior cleanup and64 spelling decisions fixed and regenerates all401 final bodies. All new outputs were personally inspected in complete source/body scene dumps. Confirmed faithful bodies increase397→399 and useful bodies335→342 on these exposed historical inputs. Synthetic281 yields280 confirmed,1 failed and267 useful; natural120 remains119 confirmed,1 uncertain and75 useful. There are59 changed raw bodies, including6 natural punctuation-only changes. These are paired prompt observations, not independent subjects or new full-chain timing.

General shared-time examples repair the known scope narrowing, a direct imperative email example preserves cancellation framing, and an English contrast example retains the explicit rejected numerical value. English weekday withdrawal improves; its Chinese counterpart remains unpolished. New work bullets omit API-restoration and ongoing-backfill background in one case, and retain an oral repeated phrase in another. Some individual deadlines move away from their owner bullets while remaining readable. These regressions prevent adopting the complete recipe simply because total scores improve. The next work-scene revision must preserve background facts and distinguish shared conditions from individually attached deadlines. Missed spelling hypotheses/selection choices are unchanged and cannot be credited as repaired by formatter prompting.


## Work-scope completeness and deadline layout

Two [work-scene revisions](../../benchmarks/reproduction/rewrite-work-scope-v4/README.md) add236 actual formatting responses, all personally reviewed against full source/body pairs. Version3 runs94 historical work candidates and16 new controls in both baseline/candidate arms (126 responses). It repairs omitted API/backfill status but newly drops an unchanged-owner requirement: historical fidelity remains93/94, useful85. Both16-control arms retain meaning and utility. This unchanged aggregate score conceals a different failure, so the recipe is not adopted.

Version4 shortens the work policy, explicitly preserves unchanged details and reorders shared/individual-date demonstrations. All94 historical candidates and the same16 exposed controls are regenerated (110 responses). Historical meaning is retained94/94 and useful organization87/94; all16 controls remain faithful/useful. Owner/date grouping is restored in four historical cases including the user-preferred Chinese example, while three other cases keep dates separately, including an English grouping regression. The repeated-code oral restart is now removed; shared start time and validation gate remain shared. Seven unresolved-term bodies still fail useful-quality expectations; no new spelling-selection repair is credited.

The other307 scene requests and encodings exactly match prior actual requests in a zero-generation audit. They are not newly executed, and these work-only observations do not establish a new full401-order or full-chain result. Native request timings exclude reused historical cleanup/selection; the16 synthetic controls execute no cleanup/selector. Production provider/prompt/validator, fresh24, ASR/OCR, independent ratings and minimum16GB/native acceptance remain unchanged.


## Tone-insensitive hypotheses do not establish intended spelling

The [complete chain](../../benchmarks/reproduction/rewrite-tone-chain-v1/README.md) actually executes candidate401 historical sources plus24 new authored controls and separately executes24 baseline controls:538 responses, all reviewed,449 Go cleanup attempts. Chinese phonemes may now match across tones; English policies, selector prompts/schema and paired-control formatter stay fixed. All64 old actual selector requests/decisions and401 historical final bodies match prior reviewed evidence. Only one historical case gains a candidate, and the model still keeps验受报告. Candidate availability and effective correction are distinct.

The eight reviewed typo controls improve2/8→3/8 repairs, adding质保书, while24-control source fidelity drops24→23 and usable output remains16 in each arm. The valid nursery培植 term is replaced with software配置 despite the source domain and exclusion. Wrong-domain OCR and an instruction occur together, so the run does not isolate their individual causal roles. Reject this coverage change until spelling selection handles source semantics and context relevance more reliably; do not hide its wrong output with a validator.

This run also completes the current formatter across all401 history sources,400 confirmed/1 conservative ASR-source uncertainty/344 useful. Historical synthetic281 text-chainp95≈2652ms and natural120p95≈1081ms exclude startup and are not native/concurrent/min16 acceptance. No priming was performed; the older small priming study remains separate.

A separate evaluator-only check finds af→afraid exactly matches the archived published transcript after case/punctuation normalization. This is evidence of reference restoration, not a contradiction of speech, but it does not silently change the conservative source-fidelity label or replace independent intended-meaning/style ratings. Keep those evaluation targets separate rather than using source overlap alone as a correctness judge.

The new timing review checks538 engine/outer observations and retains both stages for the twenty slowest historical synthetic cases. Several largest formatter prefill durations are2.2–2.7s with94–96 cached tokens; other slow pipelines contain a long selector before a short formatter. This motivates studying prefix reuse across stage/script switches with controlled requests, while preserving startup/miss costs and privacy boundaries. These observations do not identify a cache change causally or satisfy native/concurrent/min16 timing.


## Semantic-role policy development evidence

The [selector experiment](../../benchmarks/reproduction/rewrite-selector-semantic-v1/README.md) reduces harmful role substitutions6→2/32 but loses positive repairs3→1/8 on prior controls and nine historical name corrections. Three question/term repairs improve; total historical usefulness344→338/401. Reject this prompt. Same-letter readCash normalization remains allowed. Full609 actual responses are reviewed without semantic rejection. Eight fixed-source groups are not32 independent utterances; context-dependent producer suppression is explicitly separated from selector behavior. All101 candidate selector rows retain an incorrect old diagnostic template label/hash; complete actual messages bind to the frozen new prompt in a sidecar. This is diagnostic development evidence, not clean release provenance. A versioned executor requires explicit frozen prompt ownership for future policies. Actual16GB/native/concurrent/independent acceptance remains open.


## Entity-positive semantic policy

The [next selector](../../benchmarks/reproduction/rewrite-selector-semantic-v2/README.md) restores all nine entity repairs lost in v1 and improves old typo recovery1→7/8. Historical usefulness338→346/401, with400 confirmed/one source uncertainty. Three wrong-domain instruction substitutions and four related-context role misses remain, so the candidate is rejected before fresh evaluation. All558 current responses are reviewed; no semantic output rejection is used. The actual v3 executor now records explicit frozen prompt ownership and actual message hashes, verified by replay. The living report displays intermediate `choiceConforms` independently of recorded source fidelity; missed repairs are no longer shown as conforming merely because the original wording was retained. Original raw labels and legacy choice-scoring bases remain visible.


## Candidate sentence preview development evidence

The [preview experiment](../../benchmarks/reproduction/rewrite-selector-preview-v1/README.md) holds semantic-v2 rules/examples/hypotheses/formatter fixed and adds only deterministic full-sentence candidate substitutions. All558 actual outputs are reviewed and selector requests project exactly to the frozen baseline after deleting previews. Old repairs7→8/8 and related role repairs0→3/4 improve, but role harmful substitutions3→10/32 and old controls gain two wrong-object replacements. Total usefulness remains383/457 while unfaithful bodies3→12; reject the candidate. The nursery error changes the forbidden alternative span, so main-noun/forbidden-substring checks alone would miss it. No semantic output rejection is active. After-only framing and target repetition are hypotheses to test, not established causal diagnoses. Balanced source/alternative data must retain all positive and negative cases; no new held-out or native/min16 qualification is claimed.


## Balanced selector representation and stage-only qualification

The [four-arm screen](../../benchmarks/reproduction/rewrite-selector-screen-v1/README.md) actually prepares all457 sources and runs101 candidate-bearing requests per arm:404 model outputs, all reviewed, zero new final bodies. Baseline/after controls exactly match202 old complete-chain requests/choices/edited sources. Conformance is92/88/94/83; harmful choices3/12/4/17. Balanced data improves over after-only exposure but retains four harmful choices and three misses; explicit entity/ordinary-word rules worsen this candidate. No arm passes; no full-body or fresh generation is counted. Same-letter normalization remains optional under the existing human policy, with evaluator targets frozen before inference. Stage-only screening preserves full source/skip coverage and does not narrow final acceptance or prove final delivery quality.

## Granite selector and native-runtime follow-up

The [official1.63B checkpoint study](../../benchmarks/reproduction/rewrite-selector-model-v1/README.md) retains an initial Metal failure with0 completed responses, followed by12 runtime probes with7 completed outputs and5HTTP errors. The initial request is a sixth HTTP error. CPU-layer0 returns valid probe JSON; Metal without schema produces repeated8. f32KV and flash-off do not repair it. No NaN instrumentation or exact kernel cause is established.

The explicit [CPU-layer quality comparison](../../benchmarks/reproduction/rewrite-selector-model-cpu-v1/README.md) then completes202 actual choices, all personally reviewed. Same-request baseline/balanced Qwen controls score92/94 conforming with3/4 harmful choices; Granite scores84/86 with16/13 harmful choices. Thirty choices change:7 improve,23 regress. Provider/tokenizer/checkpoint differ, so this is a configuration comparison, not parameter-count causality. Both configurations violate the same-letter cash boundary and fail adoption; no formatter/fresh expansion is run. All457 preparations/914 outcomes remain explicit, with0 newGo preparations.

The living HTML distinguishes runtime probes, selector choices and final bodies, retaining every actual probe output and all HTTP errors. No runtime error is fabricated into a body or passing inference. Cross-script coverage, complete-chain quality and unchanged native/currentASR+Vision/concurrent/independent/actual16GB acceptance remain open.

## Public referent-description follow-up

The [four-arm description experiment](../../benchmarks/reproduction/rewrite-selector-referent-v1/README.md) completes404 native choices with equal256token budgets and all457 actualGo preparations. Current plain controls match their old raw outputs202/202 after the declared128-to256 budget change. Choice conformance92/92/94/93 and harmful choices3/4/4/5 do not support either new candidate. All212 generated description fields are personally reviewed;44 fail as inaccurate/unsupported/incomplete, including27 cases whose choice is correct. Semantic paraphrases are accepted rather than judged by literal overlap.

Descriptions are never applied as text or passed to the formatter. Actual description-before-choice serialization does not establish hidden reasoning or truthful self-report. No semantic output rejection/source fallback is introduced; no final-body/fresh expansion occurs. The living HTML keeps independent badges/filters for choices and explanations. Cross-script and unchanged complete-chain/native/currentASR+Vision/concurrent/independent/actual16GB requirements remain open.

## Source-last serialization development result

Moving only top-level source to the end of all ten examples and the current input does not reliably constrain OCR. Four actual101-choice arms conform92/93/94/88 with harmful3/5/4/12. Both source-last candidates are rejected; no formatter/fresh expansion follows. All404 responses were personally reviewed;202 reordered requests restore exactly to concurrent controls, whose choices/edited sources reproduce old outputs202/202. No semantic output rejection/source fallback or production change occurs. This remains exposed development evidence, not actual16GB/native/independent acceptance. See [complete evidence](../../benchmarks/reproduction/rewrite-selector-source-last-v1/README.md).

## Counterbalanced whole-sentence choice development result

A combined whole-sentence task/data/A-B output framing degrades against the balanced keep/replace control:81/91 conforming versus94 of101; harmful19/5 versus4. All404 actual choices and edited sources were personally reviewed. Swapping all example/current labels and option order changes24 normalized decisions:20 conformance changes and4 both-correct allowed normalization differences. All101 paired requests otherwise match exactly; no semantic output rejection/source fallback is added. Both orders fail; no formatter/fresh expansion occurs. Each arm retains12 original-only ambiguity schemas, disclosed separately from89 non-forced cases. This is exposed development evidence, not actual16GB/native/independent release acceptance. See [complete evidence](../../benchmarks/reproduction/rewrite-selector-sentence-choice-v1/README.md).

## Eligible hybrid-model development result

Actual1.46B Granite4.0-h-1b runs on nativeb11734 Metal with41/41 GPU layers;12 probes confirm limited JSON/template/cache compatibility, but4 cash-definition choices fail, including schema-free requests. Full202 same-request choices conform81/82 of101 with20/19 harmful, versus Qwen92/94 and3/4 harmful. Both repair47/47 required items; recall alone would hide serious OCR-induced regressions. All214 outputs,19 unique failed tuples and41 paired changes were personally reviewed. Both configurations fail, so no formatter/fresh/production expansion follows. Runtime success is not semantic/release success. See [complete evidence](../../benchmarks/reproduction/rewrite-selector-hybrid-v1/README.md).

## Trusted instruction-language development result

Only translating the existing nine system clauses to English worsens both current eligible configurations while all Chinese data/examples/candidates/schema stay fixed. Qwen conforming92/94 becomes88/87, harmful3/4 becomes10/12; hybrid81/82 becomes80/79, harmful20/19 becomes21/22. All404 actual outputs,24 unique failed tuples and23 changed same-model pairs were personally reviewed. All four translations fail; no formatter/fresh/production expansion follows. This assistant-aligned translation is not proof about English instructions generally, independent generalization or actual16GB acceptance. See [complete evidence](../../benchmarks/reproduction/rewrite-selector-instruction-language-v1/README.md).

## Complete-reference message separation

The [reference-turn study](../../benchmarks/reproduction/rewrite-selector-reference-turn-v1/README.md) preserves complete OCR but moves context into preceding tool-role turns in all existing demonstrations and current requests. Native template-only preflight confirms user-turn/tool_response serialization, not privilege or tool execution.404 actual choices conform92/89/94/91, with3/9/4/8 harmful cases and6/3/3/2 missed-repair cases. All outputs,14 unique failed tuples and16 changed pairs were personally reviewed;202 controls reproduce exactly. Both new variants are rejected. Harm with plain unrelated OCR as well as imperative OCR shows command filtering alone would not resolve relevance. No new final bodies or production changes follow; actual16GB/native/independent and complete-chain acceptance remain open.

## Unique-status label exposure

The [single-field study](../../benchmarks/reproduction/rewrite-selector-unique-label-v1/README.md) removes only status:unique from all existing example/current hypotheses. Actual producer/prepared data, completeOCR, every spelling option, kind/ambiguous labels, system rules, answers and schema remain unchanged.404 actual choices conform92/91/94/91, with3/4/4/7 harmful cases and6/6/3/3 missed repairs. All4 changed pairs regress; no required repair improves. All404 outputs and13 unique failed full-source/context tuples were personally reviewed. Controls reproduce202/202 exactly. Reject both variants; this does not remove all candidate-cardinality cues or establish a universal metadata effect. No new final bodies, ASR, Vision or production changes. Original full-chain/native/independent/actual16GB acceptance remains open.

## Added semantic contrast demonstrations

The [four-example prefix study](../../benchmarks/reproduction/rewrite-selector-contrast-demo-v1/README.md) retains the original ten examples/rules and all current source/fullOCR/candidate/schema data, adding novel fabric/project and embankment/jewelry pairs with one mixed replace/keep demonstration. Native template/tokenizer checks make zero model generations; maximum full prompt2482+reserve128 fits4096, and all404 actual inference token counts match.404 actual choices conform92/90/94/92, harmful3/3/4/6 and missed6/8/3/3. All outputs,13 unique failed tuples and6 changed pairs were personally reviewed;1 change improves and5 regress. Both new prefixes are rejected. No semantic output filtering, new final bodies/ASR/Vision or production changes. Exact-source-disjoint authored demos are not independent or held-out evidence. All original full-chain/native/independent/actual16GB acceptance remains open.

## MiniCPM5-1B same-request result

The [actual1.08B comparison](../../benchmarks/reproduction/rewrite-selector-minicpm-v1/README.md) conforms80/77 of101 versus prior same-request Qwen92/94. Harmful11/4 and missed10/20 expose both OCR meaning pollution and insufficient required repair (37/27 of47).55 choice differences include6 improvements,35 regressions and14 both-correct normalization differences. All202 outputs,12 native responses,32 failed tuples,55 changed pairs and1 interrupted native-log completion were personally read. Native JSON/cache/nonthinking/template compatibility passes separately, while all12 probe semantic judgments fail. Pinned lexer single-final-LF removal explains the initial template assertion; failed orchestration and its1 log-only completion remain visible, not complete-case credit. Reject both configurations; no semantic rejection/source fallback, new final body/ASR/Vision or production change. Full-chain/fresh/cross-script/native/concurrency/independent/actual16GB gates remain open.

## MiniCPM F16 export result

The [official F16 comparison](../../benchmarks/reproduction/rewrite-selector-minicpm-f16-v1/README.md) preserves202 complete Q4 requests/preparations/input-token counts and native settings. Both arms conform82/101 versus Q4 baseline80 and balanced77. Required repair rises37/27 to43/45 of47, while harmful changes rise11/4 to15/17.47 changed pairs include25 improvements,18 regressions and4 both-correct normalizations. All202 choices,12 probes,22 failed tuples and47 changed pairs were personally read. Exact tokenizer/architecture metadata, template and tensor name/shape identity do not prove numerical conversion lineage. Eight native positive probes improve but four cash-literal probes remain wrong, including without schema. Both configurations fail; no final-body/fresh/production expansion or semantic filtering/source fallback. Full-chain/currentASR+Vision/cross-script/native/concurrency/independent/actual16GB gates remain open.

## Youtu same-request development comparison

[Official Youtu1.96B Q8](../../benchmarks/reproduction/rewrite-selector-youtu-v1/README.md) produces75/84 conforming choices per101,3/4 harmful and23/13 missed repairs. All202 outputs and12 native probes were personally reviewed;58 changed control pairs contain5 improvements and32 regressions. Incorrect OCR still causes role substitutions while sufficient context often fails to repair ASR spellings. Both configurations are rejected; no new final bodies, independent/actual16GB acceptance or production change. The custom license is authorized for research, with production restrictions unresolved.

## LFM2.5 same-request development comparison

[Official actual1.17B Q8](../../benchmarks/reproduction/rewrite-selector-lfm25-v1/README.md) conforms79/101 in both arms, with22 harmful modifications despite47/47 required repairs. All202 choices and12 probes were personally reviewed, including21 unique failed tuples and46 changed pairs;9 improve and37 regress. Every unique-candidate case is replaced, including explicit comparisons and incompatible OCR meanings. Both configurations are rejected; no new final bodies, fresh/currentASR+Vision, independent/actual16GB acceptance or production changes. Local text-only serializer parity is separate from native/token-ID parity; the official Q8 is distinct from the QAD-Q4_0 artifact.

## Existing demonstration order

The [order-only study](../../benchmarks/reproduction/rewrite-selector-demo-order-v1/README.md) preserves all ten original pairs and changes only their order, moving three keep examples last.404 actual choices conform92/78/94/88, harmful3/2/4/2 and missed6/21/3/11. All outputs,25 unique failed tuples and27 changes were personally reviewed:3 improve,23 regress,1 remains allowed. Both treatments are rejected.202 current controls reproduce prior actual choices;202 inverse permutations restore exact current controls. Native template/token checks generate zero answers and all404 actual input-token counts match. No final-body/ASR/Vision or production changes. The original corruption-test arm-name failure remains; version2 corrects the test target only. User has no16GB device currently and authorized continuing32GB development verification; actual minimum-configuration acceptance remains open.

## Native decision-model packaging and piece limits

The [d1 actual0.38B text-component investigation](../../benchmarks/reproduction/rewrite-d1-evidence-v1/README.md) reviews72 real typed responses, six sources across six layouts/CPU/Metal. Choice counts per six are3/2/3/2/6/3 on both backends; probabilities differ. The pinned native model path clips questions to96tokens and each two-choice description to45, independently of total context.48 responses therefore ran clipped requested instructions. Native tokenizer/detokenizer evidence reproduces all60 earlier usage counts and50 observed prompts. Moving full policy into state avoids clipping in all12 new responses, maximum514tokens, but both backends remain3/6. The compact sentence6/6 observation has clipped trusted rules and exposed small samples; it does not qualify. No full-source expansion, final bodies, newASR/Vision/Go, confidence thresholds, semantic filtering/fallback or production changes. Original complete-chain/native/independent/actual16GB acceptance remains open.
