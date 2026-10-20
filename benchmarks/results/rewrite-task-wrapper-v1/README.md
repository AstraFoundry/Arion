# Scene task wrapper results

Do not promote this version. Explicit per-input scene tasks improve owner bullets
and some developer restrictions, but introduce a semantic regression and latency.
All 132 generations and independent cleanup outputs were personally inspected.

| Paired outcome | Labeled | Extra scene task |
| --- | --- | --- |
| Development faithful | 16/18 | 16/18 |
| Development faithful + scene ready | 12/18 | 14/18 |
| Old regression faithful | 22/24 | 21/24 |
| New test faithful | 18/24 | 18/24 |
| New test faithful + scene ready | 14/24 | 16/24 |
| New test required OCR repairs | 4/7 | 4/7 |
| New test ambiguous-context guesses | 3 | 3 |
| Request p50 / p95 | 838 / 1150 ms | 1265 / 1590 ms |

The owner-task-date anchor and a previously unseen owner case now produce correct
bullets. Email paragraph behavior remains good. However, the old source question
“这个方案还不能发布吗？先别操作。” becomes the assertion “这个方案还不能发布，先别操作。”;
the lexical validator accepts it. All three new ambiguous spellings are still
guessed. Name/platform/ordinary-term repairs remain incomplete. No reliable
improvement in semantic fidelity follows from the layout gain.

This isolates the extra wrapper sentence with the same model, templates, examples,
runtime and common worker budget. Input length differs and may contribute to
prefill time; shared-host timing is descriptive, not end-to-end product latency.
Fresh probes were frozen but unused in the failed clean-policy study, then reused
with explicit provenance before this study's output. All are authored text,
not actual OCR/ASR or independent ratings. No production behavior was changed.

[Complete output review](../../runs/rewrite-task-wrapper-v1/report/review.md),
[metrics](../../runs/rewrite-task-wrapper-v1/report/summary.json),
[reproduction](../../reproduction/rewrite-task-wrapper-v1/README.md).
The labeled package remains the development baseline. Next test must jointly
protect question intent, handle same-sounding OCR ambiguity and reduce input cost.
