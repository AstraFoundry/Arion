# Rewrite stage preservation: Doubao and Chatterfly

## Scope and artifact identity

This focused REA follow-up asks whether the shipped clients preserve a correct contextual spelling from recognition or provisional rewriting through final organization. It reuses matching retained Evidence rather than repeating native analyses that previously timed out. Static observations establish branches, not executed behavior or server policies. No competitor was launched, microphone used, personal data read or endpoint requested.

Installed Doubao is **1.0.2 (1)**; installed Chatterfly is **1.0.8.13459 (13459)**. Earlier Chatterfly context/orchestrator evidence concerns the separately retained **1.0.6.13398** payload and must not be silently attributed to 1.0.8.

| Artifact | SHA-256 revalidated against original bytes |
| --- | --- |
| Doubao installed main | `4b3836dcb8a3e06b01b4c95390c757b71f623c1f7c4141c059e49284f7f6442d` |
| Doubao arm64 | `266f49f6bcb62e43c5170620d4b86e2e4c4ef873bcce624ef1a90b9c46244983` |
| Chatterfly installed main | `595e502f74e41e1b28904affa9093974f6cbc197a3756a74a007ef562f9ee767` |
| Chatterfly 1.0.8 arm64 | `3406894676fd6b9028795858907d1821f54977aa0f9ec5716d42eba12848fe21` |
| Chatterfly 1.0.6 retained main | `1063fbb5b0e18dc323ec4a14487456795b359b9330cfeee14b78454a00791e50` |
| Chatterfly 1.0.6 retained arm64 | `2ea312f6324accd6194e6c612608870321bc62c286aa2cdbccf8288ae19a2433` |

All six Doubao ranges and nine Chatterfly 1.0.8 ranges in the [validator comparison ledger](evidence/rewrite-validator-comparison.json) were rehashed and match. The 1.0.6 slice was also compared directly with its original universal slice. Local verification is retained in `.workspace/research/rewrite-stage-revalidation.json`. These new checks are ordinary shell observations, not new REA Evidence IDs.

REA MCP tools were absent from this session. Pinned `rea-agents@6.4.0 doctor --client codex --json` confirmed aligned registration, but no configured Ghidra/IDA provider; the report is retained in `.workspace/research/rewrite-stage-doctor.json`. Existing byte-matching Evidence remains usable. No client configuration or engine was changed.

## Findings and evidence boundaries

| Question | Observation | Inference and limit |
| --- | --- | --- |
| Does Doubao organize text separately? | `DBD-organize-body` constructs `query`, numeric `scene=6`, `stream=true` and two spacing options for text organization. `DBD-completion-contract` checks request identity, decision/phase and nonempty completed content. | Recognition and organization have separate responsibilities. The numeric scene has no recovered semantic label; server prompts and entity protection are unknown. |
| Does Doubao have stronger final ownership? | Actual decompilation in `ev_7813c9cf5cfd974f88db927e1d3ec7e857a490b646f2b4d8968d44ded8cdb348` establishes generation/session/commit-sequence checks before delayed insertion; `CR-DB-coordinator` retains original/organized text and separate delivery actions. | The inspected delayed-write path checks captured session/generation/sequence. It does not prove semantic preservation of a chosen word. The coordinator surface itself is weaker string/reflection evidence. |
| What enters Chatterfly rewriting? | In 1.0.6, `CF-asm-llm-orchestrator` cancels old postprocessing and supplies `asrStableText`, context and configuration to a separate processor. `CF-asm-payload` serializes ASR text, request/session identities and optional context. | A stable source revision is a clear stage input. No repeated OCR selector followed by a second formatter is established. Exact partial/pause/final scheduling remains unresolved. |
| Can the final result differ from the streamed text? | In 1.0.8, `CVA-CF108-cmode`, `0x1004264d0–0x1004268e4`, appends deltas and calls `onDelta`; the finish path takes response text when supplied, otherwise copies the delta buffer. Anchors: `0x100426688–0x1004266e8`. | A final payload can supersede accumulated deltas within the same postprocessing response. This is not evidence of a second client LLM pass or a correctness guarantee. |
| Which result gets committed? | `CVA-CF108-result` stores response `data.text` as `enhancedText`; `CVA-CF108-commit` selects delivery/result-panel branches using mode/focus/error state. Original ASR remains separately available. | Final text and original text are distinct artifacts. None of the nine inspected methods establishes a protected-spelling contract. This bounded negative finding says nothing definitive about uninspected code or service validation. |
| Is OCR proven to feed voice rewriting? | 1.0.6 context serialization supports screen content, but its inspected builder/setter search did not establish population. In 1.0.8 the symbol inventory includes OCR-window commands and `setScreenContent:`. | An OCR feature elsewhere in the IME is not a voice-context call path. No OCR-to-voice-postprocessing linkage is established for either version; do not generalize the 1.0.6 bounded result to 1.0.8. |

Sources: [Doubao disfluency and request ledger](evidence/doubao-disfluency.json), [actual Doubao decompilation](evidence/doubao-decompiled.json), [organization surfaces](evidence/competitor-rewrite.json), [Chatterfly 1.0.6 instruction ledger](evidence/chatterfly-rewrite.json), [Chatterfly 1.0.8 instruction ledger](evidence/rewrite-validator-comparison.json).

The Chatterfly 1.0.6 package identity comes from actual REA Evidence `ev_e86f7a8a483ab5d40506a8aadc00d715761d84afe030d451ac85c0ba46f38d70`, archive extraction `ev_d7b75b5f00fcbb8469fe6c48e8050a0c2699c71754343e3c858052dbc139272d`, and Mach-O inventory `ev_357f2213ffae36932ca3c1c489824db2b8487a33c049894dba6808dca0e0d91e`. `CF-*`, `CVA-*`, `CR-*` and `DBD-*` identify ordinary static observations, not fabricated REA outputs.

## Arion design implications and experiment obligations

These are Arion proposals, not recovered competitor algorithms:

1. Retain immutable recognition, selected spelling edits, formatter input/output, preview revision and committed text. Record the first stage that damages a correct term; fluent final text is insufficient evidence.
2. Give each stage one authority. Context selects an evidenced spelling; organization handles prose and layout. Carry accepted spelling decisions as structured edits with source revision and exact spans, rather than trusting a later prompt to infer which words were settled.
3. Apply a spelling-preservation check only to selected, validated edits. Unrelated OCR words must not become mandatory output. Cancellation, changed source and changed context invalidate decisions that cannot be safely realigned.
4. Treat final payload and streaming preview as separate observable artifacts even when produced by one request. Reuse a preview only when the complete effective input matches; identical visible ASR alone is insufficient.
5. Reproduce a real model regression before optimizing: correct contextual selection or preview, then an incorrect final organization result. A forced stub demonstrates control flow, not inference failure. Freeze all inputs and stage outputs before changing production behavior.
6. Compare paired baseline, prompt-only preservation, structured selected-edit preservation, and preservation plus formatter authority restriction. Include no-context, unrelated/corrupted OCR, repeated terms, genuine spoken correction and changed-final-ASR controls. Measure selected-term corruption, correction success, semantic fidelity, useful formatting and latency separately. Report failed trials and do not tune against held-out cases.

The artifact research answers client stage boundaries and final-result selection for the inspected versions. It does **not** establish competitor OCR-assisted voice correction, entity locking, exact server prompts/models, actual preview-to-final regression rates or equivalent local inference quality. Controlled Arion inference and native product acceptance remain separate work.

## Exact-name source diagnostic: Chatterfly versus chad fly

A temporary Go program called the real `spelling.Load().Prepare` and `domain.Correct`; no LLM or audio inference ran. Source/results are retained in `.workspace/research/rewrite-stage-probe/`. With OCR context `Chatterfly`, sources containing `Chatterfly`, `chad fly` or `chatter fly` all produce **zero spelling hypotheses** in prompt, work and chat scenes. The screen-identifier producer requires at least two case/digit-separated parts; `Chatterfly` has one, so it never enters the candidate set. This also prevents the existing collapsed-letter equality branch from repairing `chatter fly` to this spelling.

As a positive control, OCR `ChatterFly` produces a `chatter fly → ChatterFly` hypothesis in prompt/work. It does not repair `chad fly`: pinned dictionary readings are `chatter = CH AE T ER`, `chad = CH AE D`, `fly = F L AY`, with no `chatterfly` entry. Exact pronunciation equality cannot bridge this acoustic difference. These are text-derived pronunciations, not measurements of the spoken audio.

Approved vocabulary replacements are a separate mechanism. With explicit entries `chad fly → Chatterfly` and `chatter fly → Chatterfly`, `domain.Correct` repairs lower/title/uppercase `chad fly` and spaced `chatter fly`; `chadfly` remains unchanged. `Vocabulary.Terms` also recalls a relevant entry from canonical `Chatterfly`. This demonstrates capability with supplied entries, not the presence of such entries in the user's memory.

The installed worker must be distinguished from the checked-out producer. `/Applications/Arion.app/Contents/Resources/llm/arion-llm`, SHA-256 `c30d031339337153100bc29f6a126a235fb06c4ae9f302f7e487bd3810e51d0a`, contains `selector-screen-v1-baseline+natural-layout-v1` and no `natural-layout-v3`; at that initial diagnostic revision, checked-out `prompt.SelectedVersion` was `selector-screen-v1-baseline+natural-layout-v3`. Known literal searches establish this artifact difference, not the full installed source revision. The static version check did not execute the worker; subsequent installed-worker executions are retained separately. The final candidate source is `selector-contextual-v5+natural-layout-v3`.

Current `packages/core/application/live.go` also re-decodes the whole audio after stop when segment finalization is disabled, then delivers that new result. Thus acoustic finalization is another plausible first-divergence point for this exact name. The Go spelling probe does not establish whether the user's observed `Chatterfly → chad fly` first occurred there or during formatting; stage-recorded real audio/model reproduction is required.

## Rejected experiment: fixed names with model-written gaps

A local experiment split source text into fixed names/literals and editable gaps, asked the formatter for a JSON object of gap text, then interleaved fixed bytes locally. It was rejected and removed from production. The remaining source retains ordinary formatting; contextual nomination and preview reuse must be evaluated independently.

Two actual native generation attempts are retained under `.workspace/rewrite-stage-preservation/`:

- `development-full-1.jsonl`, SHA-256 `840d8c8db839d1b48c6a45b1e12d33b0ae871c3f6105d2772855b2c806831218`: eight pairs / 16 phases. Assistant inspection found duplicated fixed names, literals or instructions in 14 phases; the other two retained upstream wrong names. All 16 failed intended-output review. This is assistant review, not independent human evaluation; substring-presence metrics alone misleadingly counted several duplicated entities as present. The hash-bound review and rejection are retained as `development-full-1.review.json` and `development-full-1.rejection.json`.
- `development-full-2.jsonl`, SHA-256 `71a9c134518ea71a176a6ba7ca46662e483f57c9760df7e219778c4729610728`: a gap-format example and an explicit duplicate-fixed-text protocol rejection were added. Fifteen of 16 phases failed with `repeated fixed spelling in layout gaps`. Catching corruption changed it into a generation failure; it did not establish usable output. The source snapshot and hashes are retained in `attempt-2-source/sha256.json`, including `packages/core/domain/rewrite_layout.go`, `apps/desktop/infrastructure/llm/prompt/selected.go` and `apps/desktop/infrastructure/llm/gguf/engine.go`.

Independent source review also identified three reasons this experiment was unsuitable for adoption:

1. The layout builder collected fixed Latin tokens from every context value, including `EarlierSource`. This bypassed the new producer's independent-evidence boundary: ordinary shared English words could become fixed merely because they appeared in earlier recognition. Selection evidence and protection authority must remain separate; incidental screen words must not all become immutable.
2. It fixed every context-matching occurrence without respecting spoken correction. A source such as `用 Chatterfly，不对，改用 Typeless` with screen evidence for Chatterfly necessarily retained a Chatterfly anchor. The formatter could not resolve the speaker's correction by removing that earlier mention. Preserving source spelling is different from requiring every earlier mention to survive intentional cleanup.
3. The formatter retained existing plain-text assistant demonstrations and then appended one Chinese JSON-gap demonstration. This mixed output contracts and provided no representative English, numeric-list or multi-anchor structured examples. It plausibly contributed to the duplication behavior; the causal contribution was not isolated by this experiment.

UTF-8 span shifting used byte lengths consistently for the producer's ordered nonoverlapping hypotheses; no concrete offset failure was found in that bounded review. However, generation-free preparation budgeted the plain formatter instead of the larger structured request, so it could not certify structured-request fit. These code observations are independent of the measured duplicate-output failures. There is no claim that fixed-span editing is universally impossible; this model/prompt/protocol implementation failed its actual local trials and is not an adopted protection mechanism.

## Input qualification and forced ambiguity

The `selector-contextual-v5+natural-layout-v3` source keeps the original selector messages/examples and ordinary formatter. Its domain producer excludes English contextual candidates while the current sentence discusses literal words/spelling or explicit spacing/written-form intent; prior candidates are additionally excluded when that sentence characterizes earlier recognition as mistaken. This is conservative lexical input qualification, not semantic output filtering. A legitimate metalinguistic request to repair spelling can also be excluded; already-applied approved vocabulary is independent. Source bytes remain unchanged by qualification. Ordinary product references, function names and numbers remain eligible, and unrelated sentences do not disable a candidate.

When every nominated hypothesis is ambiguous, the existing selector schema permits only `keep`. The worker now constructs that sole legal decision and applies the normal domain coverage/span checks without a selector inference request, then sends the unchanged source to ordinary formatting. Mixed hypotheses still require inference. This removes a redundant model request; it supplies no semantic confidence and does not repair the formatter's remaining errors. Unit transport tests establish selector-call exclusion and exact formatter input, while malformed spans, duplicate IDs and inconsistent ambiguity fail explicitly. Actual generation adoption remains governed by the paired experiment report.

## Adopted change and actual regression results

The final source uses `selector-contextual-v5+natural-layout-v3`: original selector examples and ordinary formatter, expanded contextual compound retrieval, bounded previous-recognition evidence, current-source intent qualification, approved-memory continuity and exact-request preview reuse. Prompt-only strengthening and contrast demonstrations were rejected after harmful preservation results; they are not shipped. Full-ambiguity selection is mechanically kept through the existing span validator with no selector generation.

Actual native replay of an authored Chatterfly utterance, fixed-image Vision and the installed worker reproduced seven correct rewritten previews followed by a wrong final `chatter fly`. The first observed divergence was whole-audio recognition, followed by lost relevant vocabulary recall; the formatter preserved its wrong input. The original user's audio is unavailable. Literal `chad fly` is a separately labeled constructed text-stage counterfactual, not an exact historical acoustic reproduction. Approved memory in the native demonstration was deliberately exposed development memory created after observing recognition, not pre-existing user data or held-out evaluation memory.

Assistant review of the final paired 24-case source-intent set found baseline 39/48 versus candidate 48/48 correct phases: nine independently supported repair finals recovered, and all 15 preservation finals retained. The first 16 controls had already exposed rejected variants; the remaining eight were frozen before the final guard implementation. They are authored development controls, not independent speakers or a blind human rating. Baseline rows combine two original runs with explicit row/hash lineage; latency is not an isolated treatment estimate.

The eight-pair development ablation produced 12/16 correct phases for baseline, 14/16 without prior recognition, 10/16 without screen context and 15/16 for the full candidate. The remaining full-candidate failure is an upstream preview recognition miss. Actual coordinator replay corrected all three Chatterfly final deliveries, including the screen-off/approved-memory-only case. A completion-matched cache on/off replay used identical source and requests except `Live`: both completed the full preview, identical raw/final text was delivered, and caching removed exactly one final model call. The first shorter-silence pilot had a completion-timing confound and remains separately retained.

Two numeric native controls retained distinct decimal values and quantities. A technical-name control still introduced a wrong negation after correcting `getUserProfile`, and an ambiguous contact remained unresolved. Those failures prohibit treating this change as a general semantic-fidelity pass. The installed app was not replaced. See the [complete stage-preservation archive and reports](../../benchmarks/reproduction/rewrite-stage-preservation-v1/README.md) for exact sources, native requests, assistant judgments, rejected trials and checksums.
