# Entity rule and actual Vision results

Do not adopt the new rule. All 100 generated outputs and independent cleanup were
personally inspected. Both rules produce 21/26 faithful development outputs,
repair 3/5 required spellings and preserve only 5/8 correct-source controls.
Two correct-source controls become OCR excerpts; another correct name is replaced
with a wrong same-sounding name. Strict validation rejects the two excerpts but
accepts the wrong name. Prompt wording alone has not solved the observed problem.

Actual Apple M4 Vision on 12 fixed text images:

| OCR mode | Character error proxy | OCR p50 | Critical spellings |
| --- | --- | --- | --- |
| accurate (current product choice) | 2.05% | 58 ms | All intended names/code spellings retained; punctuation differs |
| fast | 56.16% | 8 ms | Much Chinese missing or corrupted; English code retained |

These are exact developer-authored image diagnostics, not natural screenshot or
ASR accuracy measurements. Do not switch production to fast for the timing gain.

Each prompt arm has 11/12 faithful outputs for each OCR mode, with 4/5 known gold
spelling targets repaired. That apparent equality is not evidence that bad OCR
is harmless: 回归测试 is an ordinary spelling correction, and the exact
程雨南→程语楠 alias is present in an example. Even with empty OCR the model emits
程语楠. The result is coincidentally correct here, but cannot establish OCR benefit
or name generalization. The corresponding correct-name/wrong-OCR control shows
why such substitutions can harm another message. The new rule also introduces
Traditional Chinese in one output. No production activation occurred.

A unique same-sounding OCR candidate is not proof of truth: the same observed
source/OCR pair can arise from wrong ASR/correct OCR or correct ASR/wrong OCR.
`identifiability.json` records this as an explicit counterfactual, not a fabricated
accuracy experiment. Better context/source reliability evidence is required to
resolve that ambiguity; the entire requested OCR correction capability remains
in scope.

[Complete raw/strict/cleanup review](../../runs/rewrite-entity-evidence-v1/report/review.md),
[metrics](../../runs/rewrite-entity-evidence-v1/report/summary.json),
[actual OCR observations](../../runs/rewrite-entity-evidence-v1/ocr/observations.jsonl),
[reproduction](../../reproduction/rewrite-entity-evidence-v1/README.md).
