# Input packaging results

The labeled package is the next development baseline; it is not approved for
production. All 150 generated outputs and independent cleanup results were
personally inspected. The same model, templates and decoding budget were used.

| Paired scope | XML | Labeled |
| --- | --- | --- |
| Old 24: faithful raw outputs | 17/24 | 22/24 |
| Old 24: faithful and scene ready | 17/24 | 21/24 |
| Old 24: required OCR repairs | 2/4 | 2/4 |
| Old 24: harmful context changes | 4 | 0 |
| New 24: faithful raw outputs | 17/24 | 20/24 |
| New 24: faithful and scene ready | 13/24 | 16/24 |
| New 24: required OCR repairs | 3/7 | 4/7 |
| New 24: harmful context changes | 3 | 1 |
| New 24: faithful strict deliveries | 19/24 | 20/24 |

Development selected labeled before evaluation: fidelity 16/18 versus 15/18 for
XML and JSON. No arm reliably produced owner/task/date bullets. JSON did not
improve the development result and was not promoted to evaluation.

Separating reference data from the final dictated message reduced context
replacement and removed the observed demonstration leakage. Remaining problems:
ambiguous identifiers are still guessed, several names/terms remain uncorrected,
one Simplified Chinese message changed script, and multiple requirements remain
inline. In the new cohort strict validation also rejected one faithful paraphrase.
Removing it entirely would deliver 20/24 faithful proposals, including the harmful
ambiguous correction; neither strict nor ungated output meets release quality.

Request p50/p95: XML 597/871 ms; labeled 744/1163 ms. These shared-host local text
requests use a common experimental absolute token budget. They are not end-to-end
stop-to-insertion latency, and cannot be causally compared with earlier studies
that used proportional output budgets. All data are authored Chinese probes,
not natural recordings/screenshots or independent blind review.

[Every output and strict delivery](../../runs/rewrite-packaging-v1/report/review.md),
[recomputed metrics](../../runs/rewrite-packaging-v1/report/summary.json),
[scene provenance](../../runs/rewrite-packaging-v1/report/scenes.json),
[reproduction](../../reproduction/rewrite-packaging-v1/README.md).

Next: retain labeled boundaries, remove hostile demonstration strings as possible
contamination sources, strengthen faithful owner grouping and unique spelling
alignment with clean examples. New tests must be frozen before generation.
