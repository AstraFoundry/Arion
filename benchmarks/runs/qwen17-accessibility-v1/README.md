# Qwen3-ASR 1.7B and accessibility evidence

See the [measured report](../../results/qwen17-accessibility-v1/README.md) and [reproduction sources](../../reproduction/qwen17-accessibility-v1/README.md).

- `qwen17-asr-dev-v1`: complete corrected paired run, 236 observations and four warmups.
- `pre-envelope`: original complete paired run, retained with protocol metadata leaks; not an adopted result.
- `failed-hello`: original failed artifact-path attestation, with exact error and no passing inference claim.
- `window`: 36 actual paired window reads, 24 fixed-source rewrites, native visibility/stale/limit checks and hash-bound assistant judgments.
- `integration`: 24 actual ASR observations, four warmups and 24 stopped-source/context rewrites; three separate native parser checks are retained alongside them. Final-only replay omits earlier stable recognition and live preview.
- `checks`: successful checks and their retained earlier failures. Provision failures remain explicit summaries.

All raw audio is authored/public benchmark material. Private models, runtime binaries and environments are excluded. Per-output reviews are assistant inspection, not independent human ratings. The result archive inventory binds original bytes and source snapshots.
