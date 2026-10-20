# Historical alignment retrospective

This adds assistant inspection of all 96 original alignment-v1 generations and
113 original auditor outputs (24 calibration, 89 generated candidates). It makes
no new model calls and does not change the original selection, observations or
proxy scores. Every derived row retains the original row hash/path and unchanged
response; `report.py` validates these against the frozen original corpus and raw
files. Post-hoc manual ratings are not independent blind review.

The current contract review also considers language preservation and natural
organization. The old meaning-audit prompt did not explicitly require language
preservation; disagreements must not be relabeled as calibrated semantic false
positive/negative rates. Invented review-before-delivery conditions and fabricated
error explanations are direct examples of harmful semantic approvals.

`python3 benchmarks/reproduction/rewrite-retrospective-v1/test_report.py` tests
full coverage and rejection of altered historical output even after rehashing.
`pnpm rewrite:retrospective:verify` checks all archived identities. `report.py`
can validate via `checked('dev')` and `checked('test')` without writing files;
its main report writer refuses overwrite. `review.py` is the archived manual
annotation/transformation source, not an automated semantic judge.

The standalone iteration HTML contains all 96 rewrites and a separate expandable
viewer of all 113 audits. Its totals do not call these new experiments or new
inference. Original [proxy results](../../results/scene-rewrite-alignment-v1/README.md)
and [retrospective review](../../runs/rewrite-retrospective-v1/report/review.md)
remain separately inspectable.
