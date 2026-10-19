# Flexible semantic reassessment

This derivative study reviews the original correction-chain-v1 delivered outputs.
Read the [result and limitations](../../results/correction-chain-semantic-v1/README.md)
and `rubric.json` before interpreting the annotations. `author_ratings.py` records
explicit non-blinded agent judgments; it is not a semantic prediction model.
`report.mjs` validates output identities and aggregates the frozen ratings.

```sh
pnpm chain:semantic-report .workspace/semantic-replay-new.json
pnpm chain:semantic-test
pnpm chain:semantic-verify
node benchmarks/reproduction/correction-chain-v1/archive.mjs verify
```

The report destination must be new. The derivative manifest pins source, results,
acquisition metadata and original study dependencies. Weights and private download
headers remain in `.workspace/`. Superseded annotation drafts are retained under
`benchmarks/runs/correction-chain-semantic-v1/drafts/` and are not scored.
