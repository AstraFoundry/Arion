# Rewrite-validator comparison probes

Seven authored candidate checks use the production `domain.ValidateRewrite`
through `cmd/arion-context-probe`. They are deterministic validation observations,
not model outputs, blind human judgments or semantic accuracy rates.
[`probes.json`](probes.json) retains original inputs; raw verdicts and validator
identities are in [`results.json`](../../runs/rewrite-validator-comparison-v1/results.json).

Five probes repeat the [scene audit](../scene-rewrite-audit-v1/README.md): layout
alone passes; concise paraphrase, work headings, word-to-digit normalization and
negation paraphrase produce lexical rejection verdicts. Two further probes
demonstrate accepted meaning changes despite identical vocabulary:

- Swapping the developers responsible for development and testing is accepted.
- Swapping today's prohibition and tomorrow's permission to release is accepted.

These checks establish that lexical similarity neither proves semantic fidelity
nor provides an adequate unconditional gate for scene-aware paraphrasing.

The [client comparison ledger](../../../docs/research/evidence/rewrite-validator-comparison.json)
separately records byte-verified installed Doubao 1.0.2 completion handling and
nine installed Chatterfly 1.0.8 methods. No commercial implementation bytes,
private app history, credentials or competitor model outputs are archived here.

To replay, build the production helper from the repository root, then run:

```sh
mkdir -p .workspace/rewrite-validator-comparison
cd apps/desktop
CGO_ENABLED=0 go build -o ../../.workspace/rewrite-validator-comparison/validate ./cmd/arion-context-probe
cd ../..
python3 benchmarks/reproduction/rewrite-validator-comparison-v1/replay.py \
  --validator .workspace/rewrite-validator-comparison/validate
```

The script compares actual verdicts with retained verdicts. A match reproduces
the observed limitation; it does not mark that behavior as desirable.
