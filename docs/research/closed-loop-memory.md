# Closed-loop memory and contextual disambiguation

The earlier [35% to 55% diagnostic](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v2/README.md`) uses authored prior pairs. It does not measure how those pairs were acquired. The [new audio study](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/user-modify-asr-v3/README.md`) begins with an empty encrypted vocabulary, recognizes earlier-session audio, applies authored user corrections through production extraction/persistence, reopens the file and tests different later utterances. The separate [context probe](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/context-memory-v1/README.md`) investigates ordinary-word ambiguity with actual text-model inference.

## Primary research

| Source | Relevant mechanism | Local boundary |
| --- | --- | --- |
| [Yu et al., Interspeech 2004](https://www.isca-archive.org/interspeech_2004/yu04f_interspeech.html) | Separates detecting correction from ordinary editing, diagnosing its cause and adapting recognition; pronunciation learning combines acoustic and letter-to-sound evidence. | Our actor supplies explicit earlier-session corrections. We measure production extraction and storage, without claiming automatic intent detection or learned acoustic pronunciations. |
| [Monteiro, Interspeech 2024](https://www.isca-archive.org/interspeech_2024/monteiro24_interspeech.html) | User feedback can enroll natural keyword audio, addressing limitations of TTS pronunciations in contextual keyword spotting. | The existing product persists pairs, not keyword audio. This study uses authored TTS and does not implement its trained spotter or natural-speech enrollment. |
| [Training-Free Contextual ASR, arXiv:2609.30694v1](https://arxiv.org/html/2609.30694v1) | Localizes likely domain errors, retrieves phonetically related terms and re-recognizes original audio. Empty retrieval skips the second pass. | Qwen3-ASR is ASR-specific; we do not assume it can emit the paper's structured error-localization output. Our queries still derive from first-pass text. |
| [PTC-Bias, arXiv:2609.28727v1](https://arxiv.org/html/2609.28727v1) | Uses trained phoneme posteriors and temporal competition to resolve overlapping near-homophones, with local acoustic verification. | The tested lexical competition policy borrows abstention on competing spans only. It has no phoneme head, posterior confidence or timing alignment; it is not PTC-Bias. It loses to the previous policy on development and is not selected. |
| [HearInContext, arXiv:2609.18680v1](https://arxiv.org/html/2609.18680v1) | Separates implicit dialogue support from explicit word hints using shared audio, competing meanings and unrelated-context controls; separately investigates adaptation of Qwen3-ASR-1.7B. | Our smaller text probe uses authored/reported transcripts and reconstructed screen text, without audio or genuine OCR capture. We do not transfer the paper's accuracy or training gains. |
| [G-SPIN, ACL Industry 2026](https://aclanthology.org/2026.acl-industry.151/) | Separates a phonetic candidate neighborhood from contextual scoring and final semantic selection. | We investigate compact retrieved examples, not its GNN, masked language model or re-ranking checkpoints. |
| [Conservative data filtering, EMNLP Industry 2024](https://aclanthology.org/2024.emnlp-industry.20/) | Correction targets should be linguistically useful and inferable from available evidence; noisy correction pairs can induce overcorrection outside their domain. | User edits supply provenance, not a guarantee that the learned replacement is appropriate in every later context. Preserve unrelated, correct, question, number and negation controls. |

## The reported ordinary-word failure

The user reported `codex能爆吗` corrected to `codex能报吗`, followed by recognition of `codex能播吗` while colleagues discussed reimbursement. The exact reported strings are disclosed development evidence. The probe reconstructs a short cost-related screen excerpt; no original recording, screenshot or runtime trace was supplied.

Code inspection establishes two structural limits independently of that particular run:

1. `domain.Vocabulary.Terms` matches stored source/destination text exactly under its word rules. A different phonetic rendering may retrieve nothing in the current product path. The research pinyin policy can retrieve the reported `播吗` variation from `爆吗` (normalized distance 0.2).
2. `application.withContextTerms` passes only `domain.ContextTerms` to the product rewriter. This extractor keeps sender/quoted names, Latin identifiers and codes; it does not preserve the relationship expressed by an ordinary reimbursement conversation. The full bounded context remains available for validation, but is not the model's product input. Even the separate section template mainly instructs spelling of names and identifiers.

This is not evidence that OCR failed to read the user's actual window. It explains how successfully recognized screen text can lose useful semantics before model input.

`报` and `爆` share Mandarin pronunciation `bao4`; `播` is the nearby `bo1`. Text-derived pinyin can connect possible confusions but cannot choose between identical pronunciations on its own. An acoustic phoneme posterior supplies different evidence from converting an already wrong transcript to pinyin. For ordinary words, the current clause and relevant dialogue must decide the meaning; a prior correction cannot safely become an unconditional global rule.

Production extraction can also retain adjacent characters rather than a complete lexical item. In the probe, `公示需要` corrected to `公式需要` produces `示需 → 式需`; `全免的` corrected to `全棉的` produces `免的 → 棉的`. The changed character is widened by the existing CJK rule. Such fragments may not match later wording. Grouping actual aliases by destination helps only when the destination representation itself is stable.

## Evaluation and adoption boundary

The audio study freezes entity, voice, audio, session and utterance-template separation before development inference. Final-test learning is from its own earlier sessions after policy selection; future references never enter learning. Negative edits, reload, undo, reversal, missed learning, retrieval, recognition errors and total compute are distinct outcomes. Two pre-inference fixture-design failures are retained rather than hidden.

The text probe separates the product path, existing full-context sections, a generic semantic-disambiguation instruction and retrieved pinyin correction examples in one user turn. The selected policy is frozen before new semantic families are tested. These examples are ordinary text, not audio few-shot demonstrations or learned phonemes. All raw generations and validator rejections are retained.

No product recognition bundle, context policy, persistence schema or shipped prompt changes in these experiments. A production change needs evidence on natural recurring user speech, actual field tracking/OCR, unrelated contexts and independent meaning review. Adding single-case rules for Codex/reimbursement or tuning to final-test failures would not satisfy that gate.

The complete-audio final test improves recurring-term hits from 3/72 to 33/72 through actual acquired memory, with one worsened utterance. The two new audio candidates lose on development. The selected text pinyin-example candidate falls from 9/12 development exact matches to 8/16 on new semantic families, below the product path's 10/16. Retaining correct inputs after validator rejection is reported separately from successful model correction. Neither new strategy is promoted.

The remaining architectural hypothesis is to keep approved prior corrections as evidence, retrieve a compact phonetic candidate set and make a local meaning choice using the current utterance and bounded dialogue, before any unconditional ordinary-word replacement. Multiple confirmed aliases may share a destination; unconfirmed phonetic neighbors are candidates, not automatically approved new memory. Acoustic retrieval and constrained local semantic ranking need a fresh experiment and data; they are not implemented by these prompt tests.

## Production follow-up

The user subsequently requested full OCR delivery and correction-boundary fixes. Production now passes full sanitized screen context to preview/final rewrites (`context-v2+live-v2`) and aligns edits to local word boundaries on both sides, skipping ambiguous single characters. See [ADR 013](../decisions/013-rewrite-context-on-device.md), [ADR 014](../decisions/014-edit-learning.md) and [verification](../quality/validation.md). The earlier runs and their frozen source snapshots remain unchanged; their model quality numbers do not measure these production changes.
