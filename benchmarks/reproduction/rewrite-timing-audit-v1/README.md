# Native/external clock consistency snapshot

This retrospective snapshot verifies396 completed native observations from the
recorded source list. Nine have native prompt_ms+predicted_ms greater than the
external Python generationMs by more than max(25ms,5%). Original bytes and all
clocks remain unchanged. Current in-progress task-wrapper inference is outside
this frozen source list. This is not a new generation or a diagnosed runtime bug.

Exact b11734 source shows ggml_time_us using CLOCK_MONOTONIC and server stages
using differences of that clock. Python reports mach_absolute_time for monotonic.
An independently compiled arm64 helper records four awake250ms intervals: rates
agree while absolute clock domains differ. It does not reproduce a historical
sleep event or observe inference-thread QoS. The bounded power-history query is
incomplete and supplies no matched sleep evidence. No thermal, scheduling or
clock-cause attribution follows from these observations.

Use both same-domain external intervals and separately recorded uptime/wall time
in the next timed experiment; do not overwrite archived fields. No existing
text-stage p95 proves stop-to-insertion latency, and inconsistent stage/request
pairs cannot approve a release SLO. The helper is calibration only; its binary
stays private. Source, compilation flags (-arch arm64 -O2), output and exact-build
upstream sources/license are retained. No power or QoS setting was changed.

`pnpm rewrite:timing-audit:verify` recomputes the fixed snapshot offline without
model/runtime acquisition or inference. Native source URLs/hashes are in
source-provenance.json. The snapshot includes raw timing identities, not an
independent semantic review or a correction to prior benchmark scores.
