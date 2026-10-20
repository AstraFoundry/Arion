# Fixed-model input packaging study

The same local 1.7B model, aligned templates, demonstrations and absolute 384-token
budget compare XML, JSON and labeled source/reference sections. All sanitized
context is retained. Only the serializer changes, including demonstration inputs.
The common native worker overlay leaves production sources untouched.

Build: `pnpm rewrite:packaging:build --out .workspace/rewrite-packaging-v1/build`.
The frozen build requires the recorded runtime/model and production source hashes.
Do not overwrite the existing build or refreeze published artifacts.
Run: `pnpm rewrite:packaging:run --phase dev --out <new-directory>`, inspect all
18 × 3 outputs and cleanup, then write the hash-bound selection. Evaluation:
`pnpm rewrite:packaging:run --phase evaluation --selection <dev>/selection.json --out <new-directory>`.
The runner deliberately requires its pinned common build location. Rebuilding on
a changed toolchain requires a new study identity rather than silently reusing it.

Model-free replay: `pnpm rewrite:packaging:report --dev benchmarks/runs/rewrite-packaging-v1/dev --evaluation benchmarks/runs/rewrite-packaging-v1/evaluation --out <new-directory>`.
`pnpm rewrite:packaging:verify` checks the archive; `pnpm rewrite:packaging:test`
checks the report integrity guards. Make provides equivalent packaging targets.
References are evaluator-only. Every raw output and cleanup was inspected by the
assistant, not independent raters. Scene assignment is explicit fixture input.
[Results](../../results/rewrite-packaging-v1/README.md) distinguish raw quality from
strict delivery. This text-only macOS experiment does not establish actual ASR,
OCR, native lifecycle, offline-network or release performance acceptance.
