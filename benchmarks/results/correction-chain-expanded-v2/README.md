# Expanded ASR -> correction -> intent rewriting study

## Decision

Do not adopt the tested BART4CSC/FLANEC top-1 correction combination as the default
stage before Qwen3-1.7B rewriting. On the new spoken set it reduces usable outputs
from 48/96 to 40/96, and preserved intent from 57/96 to 46/96. This is evidence
against these exact checkpoints, prompts and input adaptation, not against every
ASR-specific corrector or FastCorrect, which was not run.

Offline acoustic recognition followed by rewriting is the best measured path in
this study: 58/96 usable and 68/96 faithful outputs. Prioritize acoustic finalization
and reliable resolution of spoken repairs before adding a separate text corrector.
The direct rewrite itself is not consistently safe: it introduces two semantic
losses and leaves several spoken repairs unresolved. No production default changes
are made by this research.

## Scope and controls

A new frozen set has **96 spoken scripts**, 32 each Mandarin, English and mixed,
plus four separate silence durations. All six arms ran real local inference,
producing **600 test outputs**, plus retained warmups. Previous 26 cases are not
pooled into these estimates. Model/native artifact hashes, acoustic artifacts,
correction manifests and effective treatment configuration exactly match the
original study. Scripts, audio and the flexible rubric were frozen before inference;
no test outcomes were used to tune a prompt or model.

This is a single-author synthetic challenge set with two test voices per language,
175-word-per-minute synthesis and separate development voices. Some Chinese and
English long-form themes are translations, not independent semantic populations.
Voice selection alternates by script, not a fully crossed voice experiment. Audio
transcripts are the synthesis scripts, not independently reviewed human transcripts.
There are no natural-user recordings or independent human semantic ratings.

The six arms include actual production speech-span cleanup and delivery validation
for rewriting. Evaluate delivered text, not an attractive but rejected raw proposal.
Exact numbering, greetings, thanks and reference words are not mandatory. Fidelity
and organization remain separate: unresolved repairs can preserve final intent but
still require cleanup. Fluent wrong amounts or missing conditions are not usable.
Uncertain intent fails the usable threshold. Strict-clean results below require
organization 2 instead of the main adequate-or-better threshold 1.

## Primary spoken results

| Pipeline | Preserved | Degraded | Uncertain | Usable | Strict clean |
| --- | ---: | ---: | ---: | ---: | ---: |
| Streaming ASR | 59/96 | 35 | 2 | 34/96 | 30/96 |
| Streaming ASR -> correction | 48/96 | 46 | 2 | 29/96 | 23/96 |
| Streaming ASR -> rewrite | 57/96 | 37 | 2 | 48/96 | 43/96 |
| Streaming ASR -> correction -> rewrite | 46/96 | 47 | 3 | 40/96 | 33/96 |
| Offline ASR | 69/96 | 18 | 9 | 40/96 | 37/96 |
| Offline ASR -> rewrite | 68/96 | 19 | 9 | 58/96 | 52/96 |

All six arms correctly emit nothing for all four silence cases. Silence is excluded
from the primary denominator. All 600 outputs are available; no correction or
rewrite request is marked generation-truncated. Semantic omissions still occur
when a model ends normally.

### Usable outputs by language

| Pipeline | Mandarin /32 | English /32 | Mixed /32 |
| --- | ---: | ---: | ---: |
| Streaming ASR -> rewrite | 13 | 25 | 10 |
| Streaming ASR -> correction -> rewrite | 13 | 17 | 10 |
| Offline ASR -> rewrite | 18 | 25 | 15 |

Adding correction yields one usable gain and nine usable losses, a net eight
losses (-8.3 percentage points). All usable losses are English. Fidelity has two
gains and thirteen losses, net eleven losses (-11.5 points), including one Mandarin
case becoming ambiguous. Neither Mandarin nor mixed shows a usable gain from BART.

Offline -> rewrite versus streaming -> rewrite yields eleven usable gains and one
loss, net ten (+10.4 points). Gains are mainly Mandarin and mixed technical speech.
These paired counts describe this fixed corpus; no population significance is claimed.

## Concrete failures

- `expanded-en-00`: the final request is three rooms for Friday. FLANEC keeps
  `reserve five rooms`; the downstream LLM cannot recover the deleted correction.
- `expanded-en-02`: final eleven in the morning becomes nine after correction.
- `expanded-en-28`: the no-partial-shipment condition and missing-parts inquiry
  disappear after FLANEC, despite a normal end-of-sequence and no truncation flag.
- `expanded-en-10`: removing `if` turns a conditional signature check into a
  factual assertion. This is not a formatting difference.
- `expanded-en-06`: direct and offline rewrites incorrectly remove the second
  paragraph; the correction chain preserves its raw spoken repair. This is a
  fidelity gain for correction but remains unready because cleanup is unresolved.
- `expanded-zh-29`: an offline bullet list is well structured but says equipment
  is not manufactured instead of not yet arrived. Formatting cannot establish
  factual fidelity.

FLANEC was trained for N-best correction and receives one genuine hypothesis here.
The observed tendency to discard later clauses is a failure of this measured
adaptation; this run does not reproduce or invalidate its original N-best results.

## Acoustic scores and sensitivity

These compare the non-LLM outputs with the full spoken script, including repairs;
they are not rewriting-quality scores. Mixed MER is tokenization-dependent.

| Stage | Mandarin CER | English WER | Mixed MER |
| --- | ---: | ---: | ---: |
| Streaming ASR | 9.90% | 1.74% | 22.85% |
| After correction | 10.40% | 26.37% | 19.56% |
| Offline ASR | 5.45% | 1.00% | 15.90% |

The low-recognition Mandarin Eddy voice contributes 32 Mandarin/mixed scripts.
Raw streaming intent is preserved on only 2/32 of those; failures cannot all be
attributed to downstream rewriting. A **post-hoc secondary exclusion** of that
voice leaves 64 spoken cases: direct rewriting 47 usable, correction chain 39,
offline rewriting 52. The net correction loss remains eight. Both English voices
also show losses: Samantha 14 -> 8 usable; Daniel 11 -> 9. These are corpus strata,
not randomized voice causal effects.

A stricter clean-organization threshold gives direct 43/96, correction chain 33/96,
offline 52/96. Treating every uncertain result as usable cannot reverse the direct
versus correction conclusion. Subjective spelling boundaries (Git/GT, phonetic
reproduction steps, accounting/counting) are visible in the annotations and do not
explain the clear lost final numbers, conditions and clauses.

Median serial engine estimates are 1.14 s for direct rewrite, 1.24 s for the
correction chain and 0.72 s for offline rewrite. These sum stage compute times
under accelerated playback and warm sequential batches. They exclude actual speech
pacing, model loads and native insertion; no user-visible latency win is established.

## Review, provenance and replay

One Codex agent reviewed all delivered outputs with known arm identities, without
independent adjudication. Every annotation records exact text identity and a
rationale. Identical output bytes in the same case must receive identical grades.
The report does not infer semantic judgments from edit distance or keyword gates.

- [Frozen scripts](../../reproduction/correction-chain-expanded-v2/cases.json)
- [Frozen flexible rubric](../../reproduction/correction-chain-semantic-v1/rubric.json)
- [All 600 annotations](ratings.json)
- [Summary, strata, paired case IDs and reviewed outputs](summary.json)
- [Exact model observations](../../runs/correction-chain-expanded-v2/test-run/run.json)
- [Explicit review decisions](../../runs/correction-chain-expanded-v2/decisions.json)
- [Archive manifest](../../runs/correction-chain-expanded-v2/manifest.json)
- [Execution and replay workflow](../../reproduction/correction-chain-expanded-v2/README.md)

Superseded baseline review notes and initial aggregates remain under `review-drafts`.
Only silence rationales were clarified between initial and final ratings; semantic
and organization grades were unchanged. Original small-study archives are intact.
Weights, runtime environments and FastCorrect assets remain private in `.workspace`.
