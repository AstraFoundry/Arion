# General LLM IME alignment observations

Repository-authored synthetic text and frozen real Rime candidate snapshots only. No personal typing, credentials or model weights are archived.

- `pilot/`: initial ungated run, 202 actual model calls including two warmups; request JSONL, environment/source identities, status, raw chat requests/responses and explicit errors.
- `gated/`: default no-context policy, 202 input requests but only 78 model calls including two warmups. Abstentions are explicit decisions, not inference.
- `schedule.json`: reference-free 142-snapshot schedule derived from the native producer fixtures.
- `schedule-observations.jsonl`, `schedule-test.log`: preliminary real-time Go/model replay.
- `schedule-observations-v2.jsonl`, `schedule-test-v2.log`: replay after adding the final promotion-coverage check at the application gate.

The [reports](../../results/llm-ime-alignment-v1/README.md) retain errors, cancellations and coverage limitations. Complete/failing observation statuses are separate from quality acceptance. Source/hardware provenance is retained; local binaries, weights and runtime environments remain in ignored `.workspace` or provisioned model storage. `pnpm ime:align verify` checks archived hashes and exact replay.
