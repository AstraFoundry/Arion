# Scene rewrite alignment experiment

Synthetic text development study; automatic counts below are constraint/layout proxies, not semantic accuracy. No production change was adopted.

Calibration: 7/12 faithful pairs accepted; 0/12 harmful pairs incorrectly accepted.

## Development

| Arm | Accepted | Delivered facts + layout | Accepted constraint failures | Rejected fact-passing outputs | Final p95 ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| production/strict | 23/24 | 12/24 | 0 | 0 | 1306 |
| production/envelope | 23/24 | 12/24 | 0 | 0 | 1306 |
| production/semantic | 3/24 | 7/24 | 0 | 20 | 1707 |
| scene/strict | 8/24 | 9/24 | 0 | 7 | 3590 |
| scene/envelope | 19/24 | 15/24 | 4 | 0 | 3590 |
| scene/semantic | 5/24 | 10/24 | 0 | 10 | 3590 |

## Frozen test

| Arm | Accepted | Delivered facts + layout | Accepted constraint failures | Rejected fact-passing outputs | Final p95 ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| production/strict | 22/24 | 10/24 | 0 | 2 | 2180 |
| production/envelope | 24/24 | 12/24 | 0 | 0 | 2180 |
| production/semantic | 5/24 | 10/24 | 0 | 19 | 2782 |
| scene/strict | 8/24 | 8/24 | 0 | 10 | 4455 |
| scene/envelope | 23/24 | 18/24 | 5 | 0 | 4455 |
| scene/semantic | 8/24 | 10/24 | 4 | 14 | 4812 |

## Decision

The development-only decision is no-promotion. The local verifier failed its faithful-rewrite gate. Relaxing to envelope-only validation is a diagnostic arm, not a safe adoption path. Test was not used for retuning.

All generated and rejected outputs, timings, calibration, provenance and original failed observations are retained. Native speech, ASR contention, screen context, cold-load distributions and independent human review are open release gates.

The fact patterns can miss added claims, actor/condition scope or answered questions. Inspect cases.csv and complete blind review before making semantic claims. Merely inserting headings or satisfying required words is insufficient.

## Frozen diagnostic gates

| Gate | Result |
| --- | --- |
| completeGenerationCoverage | PASS |
| calibrationRejectsEveryHarmfulPair | PASS |
| calibrationFaithfulAcceptanceAtLeast95Percent | FAIL |
| devAtLeastFourAdditionalDeliveredConstraintLayoutPasses | FAIL |
| testAtLeastFourAdditionalDeliveredConstraintLayoutPasses | FAIL |
| devNoAdditionalAcceptedConstraintFailures | PASS |
| testNoAdditionalAcceptedConstraintFailures | FAIL |
| devWarmFinalP95Under1500Ms | FAIL |
| testWarmFinalP95Under1500Ms | FAIL |
