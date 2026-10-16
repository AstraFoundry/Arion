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
3. **Stronger prompts could lose uncertainty.** Exploratory prompts removed a hedge or damaged a question/self-correction. They were discarded; the shipped template remains `live-v2`. A new presence-based uncertainty proxy blocks losing recognized hedges, allows synonymous/stuttered hedges, and scopes explicit corrections to certainty to the immediately superseded clause. This is a lexical proxy, not a semantic guarantee.

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
