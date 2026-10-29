# Doubao and Chatterfly: hesitation and spoken repair

## Scope and method

This follow-up asks how the shipped clients handle hesitation, repetition and spoken self-correction, and what can be established about other languages. Targets remain Doubao IME 1.0.2 (1) and Chatterfly 1.0.6.13398. Original and analysis-slice hashes match the retained artifacts; Chatterfly's extracted arm64 slice and Doubao's arm64 slice were independently compared with their original universal binaries.

The new [Doubao ledger](evidence/doubao-disfluency.json) and [Chatterfly ledger](evidence/chatterfly-disfluency.json) contain addresses, hashes, bounded searches and limitations. DBD/CFD are ordinary local static observations, not invented REA Evidence IDs. Chatterfly reuses prior REA artifact identity and adds byte-verified LLVM instructions. A new Doubao REA/Ghidra function query returned `provider_timeout` after 240,000 ms; it produced no pseudocode and was not retried. That target was closed successfully. The new Doubao findings therefore use 3,240 independently byte-verified instructions across six bounded ranges, not alleged decompiler output.

No target application was launched, microphone used, user database read, private endpoint requested or traffic intercepted. Server prompts, weights and live outputs remain outside this evidence.

## Doubao: decoder cleanup plus text organization

There are two distinct mechanisms; the evidence does not merge them into one universal default pipeline.

**Packaged decoder cleanup.** The ASR decoder's configuration parser reads `enable_disfluency` into a Boolean field at object offset `0x1e9` (`DBD-decoder-config`, `0x10056d51c–0x10056d57c`). A missing value sets true in this parser; this does not establish the app's active configuration. Result processing calls a cleanup helper when that flag or the adjacent inverse text normalization (ITN) flag is enabled (`DBD-decoder-call`, `0x10057293c–0x10057296c`). The helper checks the disfluency flag again before entering the following marker branch (`DBD-decoder-tags`):

| Structured input pattern, statically inferred | Marker-branch result |
| --- | --- |
| `〈text|a〉` or `〈text|A〉` | Entire marked span replaced with empty text |
| `〈text|other〉` | Keep text before the first pipe; remove the marker and suffix |
| `〈text〉` | Keep the inner text |
| Opening marker with no closing marker | Stop the scan; no inferred completion |

The parser matches the UTF-8 opening/closing angle brackets, finds the **first** pipe and checks that the **whole suffix** is exactly one character `a` or `A`. These branches leave an empty replacement; other branches retain inner text. The imported `std::string::replace` receives the range including both brackets (`0x100575000–0x100575028`). Bytes, indirect symbol signatures and branches were independently checked. The same helper contains separate ITN logic, so this table describes only its marker branch.

This establishes an annotation-based span-cleanup mechanism in the packaged speech decoder. It does **not** establish who creates the annotations, whether `a/A` specifically means a filler, whether a particular utterance uses them, or the results for `呃`, `那个` or `啊不对`. A checkpoint trained to emit such annotations is a possible interpretation, not a recovered fact. This format is a static reconstruction, not captured speech output.

A separate shipped preference key, `audio.filler_word_removal_mode`, and reflection name `fillerWordRemovalMode` identify another configuration surface (`DBD-filler-setting`). Its connection to `enable_disfluency`, active value and default were not recovered. A packaged `SpeechDisfluency` SDK class is likewise not proof that every SDK path is used.

**Text organization.** The request builder at `0x100beaaa4` selects `/api/v2/ai/text_organization` and constructs five application-specific body entries: `query`, numeric `scene=6`, `stream=true`, `space_at_cn_en_nb` and `space_at_newline` (`DBD-organize-body`). The numeric scene is not assigned an invented writing-style label. These entries contain neither a visible filler dictionary nor a language field; shared networking may augment or encrypt the request, and `contextID` transport through another layer remains unknown.

The completed-result handler checks request identity, phase, rewrite/no-rewrite state and nonempty content, and manages the result timeout (`DBD-completion-contract`). It does not contain a lexical coverage ratio or filler-specific rejection test in this inspected range. Earlier evidence establishes separate original/organized delivery actions and timeout/failure/no-rewrite paths ([organization research](competitor-rewrite.md#doubao-organization-has-its-own-stream-and-decision-lifecycle)). This is a server-capable semantic organization stage distinct from decoder marker cleanup; exact server editing instructions remain unknown.

## Chatterfly: stable ASR text enters the post-processor

The traced client path is stable ASR text plus context/configuration, then a separate post-processing request, typed SSE deltas/completion, `enhancedText` and final delivery. The earlier [instruction reconstruction](chatterfly.md#native-instruction-and-call-path-findings) established that path. The new findings clarify three policies:

- **Short input is not demonstrated to bypass rewriting by default.** The ASR response's `short_utterance` flag is copied into session state. The post-processor flag is that value **AND** the debug toggle `speech.shortUtteranceSkipLLM`; the toggle's shipped title is “短音频不走大模型请求” and its registered default is false. Both post-processor implementations serialize `audit_only` (`CFD-short-audit`). The active override, server short-input threshold and meaning of `audit_only` remain unknown.
- **A completed rewrite is a distinct result.** SSE uses the server's final string, or accumulated deltas when the final string is absent. Successful response text becomes `enhancedText` and reaches `commitVoiceText`. The inspected success path has no character-retention percentage, length-ratio check or client filler dictionary (`CFD-result-policy`). Two specific security-result codes select the ASR-text path; generic errors show LLM error UI. This is not a claim that the whole app or service has no validation.
- **Original and rewrite remain available.** The result panel stores both and offers “切换原文 / 切换改写”. That callback changes the panel text, not an already committed input-field value (`CFD-asm-result-switch`).

Packaged product copy describes automatic correction, concise wording and organizing spoken developer requests. It does not reveal the rule deciding whether `那个` is padding or a demonstrative, or how contradictory clauses are repaired. No specific filler/self-correction prompt was recovered; those semantics remain a server-side unknown.

## Installed rewrite-validator comparison

The [validator comparison ledger](evidence/rewrite-validator-comparison.json)
revalidates installed Doubao **1.0.2 (1)** against all six retained instruction
ranges. Installed Chatterfly is now **1.0.8.13459 (13459)**, distinct from the
1.0.6 artifact above. Nine freshly decoded, byte-verified methods cover its
CMode response, result selection, final commit, input validation and edit metrics
(1,951 verified instructions). New CVA IDs are ordinary LLVM observations, not
REA decompilation results. No app was launched, user content read or network
traffic captured. Full local instructions remain in
`.workspace/research/rewrite-validator-audit/`.

| Client path | Recovered checks | Boundary |
| --- | --- | --- |
| Doubao completed-result handler | Request identity, state/decision, completed/no-rewrite handling, nonempty content and timeout lifecycle | No lexical-ratio or filler-word gate in this bounded handler; other handlers and service policies remain unknown |
| Chatterfly 1.0.8 CMode and result/commit methods | Delta/final result handling, error/security/command branches, focus/delivery state, enhanced text and original/result-panel separation | No source-versus-candidate lexical coverage, novelty, numeral-spelling or negation-count gate in these bounded methods |
| Chatterfly `validateAsrText` | Nonempty ASR input or pause detection enabled; otherwise reports input error | Pre-request input validation, not generated-output fidelity validation |
| Chatterfly edit metrics | Edit distance used by edit-beacon reporting; edited-text validity trims whitespace and tests nonempty | This inspected reporting path does not veto the initial rewrite |

Arion differs: the shared validator applies character-multiset coverage, lexical
novelty, length ratios, literal digit-run evidence and recognized negation counts
without a scene policy. [Seven reproducible probes](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/rewrite-validator-comparison-v1/README.md`)
show rejected paraphrases and accepted changes to assignees or release-day
restrictions. For scene-aware rewriting, these are overly restrictive lexical
gates and incomplete meaning checks. This finding supports replacing their role
as unconditional polishing gates, not asserting that competitors have no
server-side validation or that operational/protocol protections should disappear.

## Language boundary

Chatterfly's live-ASR parameter builder explicitly sets `zh-cmn-Hans-CN` (`CFD-asm-asr-params`, `0x1003c202c–0x1003c2034`). That does not rule out server language identification or mixed speech, but it supplies no proof of general multilingual cleanup. Doubao's tag parser operates on spans without checking their linguistic content; that is a property of the parser, not proof that its producer labels every language correctly. Neither client analysis establishes the quality of English, Japanese, French or Spanish disfluency handling.

## Implications for Arion

The concrete lesson is to distinguish **hesitation/repair identification**, **text transformation**, and **result delivery**. Doubao's packaged decoder can interpret span annotations; Chatterfly delegates text editing to its post-processing service and preserves the original beside the result. Neither inspected result handler supports treating a global character-retention percentage as a recovered competitor fidelity rule.

For Arion, safely adopting annotation-based editing would require an evaluated local producer and a defined annotation contract. A parser alone cannot identify fillers or infer a speaker's correction. Delegating to a model still requires measured fact/negation/uncertainty preservation, lifecycle guards and explicit failure handling. Competitor transport evidence is not authorization to add cloud inference or to remove Arion's protections without replacement evidence.

| Investigation question | Status | Remaining evidence |
| --- | --- | --- |
| Does Doubao contain a concrete disfluency cleanup mechanism? | Answered statically for the decoder branch | Active configuration, annotation origin and controlled speech outputs |
| Does Doubao have a separate organization request and result lifecycle? | Answered for the bounded builder/handlers | Shared request augmentation, server algorithm and execution defaults |
| Does Chatterfly process stable text with original/enhanced result separation? | Answered statically | Controlled request/result timing and quality |
| Is Chatterfly's short-utterance skip unconditional? | No; gated by server flag and a debug toggle registered false | Active debug value and server audit behavior |
| Exactly how do they decide whether to remove `呃/那个/啊不对`? | Partially answered by client mechanisms | Producer/server rules, training or controlled outputs |
| Have multilingual cleanup and competitor parity been measured? | Unresolved | Same consented multilingual fixtures, actual product execution and independent fluent-rater review |
