# Vocabulary alignment reproduction

This development acceptance set exercises the production `domain.Correct`, immutable vocabulary index and combined canonical recall budget. It is assistant-authored synthetic text, not held-out speech. Inputs contain approved pairs and recognized/prior text only; `expected.json` stays in the evaluator. The 2,000-entry case is a controlled scaling probe. The native labels were reviewed during development after the initial eight-case run; they are not independent or blind ratings.

## Deterministic recall

```sh
pnpm vocabulary:bench:test
pnpm vocabulary:bench:run --out .workspace/vocabulary-new-run
pnpm vocabulary:bench:verify benchmarks/runs/vocabulary-alignment-v1/recall-04
pnpm vocabulary:bench:report --run benchmarks/runs/vocabulary-alignment-v1/recall-04 --out .workspace/vocabulary-replayed-report
```

The runner snapshots source, fixtures, environment, Git state and original baseline code. Its baseline overlays the pre-change index and the historical application recall adapter; no model is substituted. Each case retains 100 warmed nanosecond observations. Construction, process startup and compilation are excluded. Baseline runs before candidate; differences are descriptive, not significance claims. Existing directories are rejected; completed and failed attempts remain hash-verifiable. `Makefile` exposes equivalent `vocabulary-bench-*` targets.

## Native staged polishing

`native-cases.json` is generated from actual candidate correction/recall observations, never evaluator outputs. It keeps the original prior recognition. Both harness phases use observed corrected text; they are not two independent examples or ASR passes. The `selector-memory` case exercises a real spelling-selector call using canonical memory; other cases primarily verify formatting preservation and no insertion of irrelevant terms.

```sh
GOCACHE=/private/tmp/arion-vocabulary-go-cache node scripts/rewrite-stage-command.mjs run --arm candidate --cases benchmarks/runs/vocabulary-alignment-v1/recall-04/native-cases.json --out .workspace/vocabulary-native-new --runtime .workspace/rewrite-small-models-v1/runtime/llama-b11734 --model .workspace/rewrite-small-models-v1
python3 benchmarks/reproduction/vocabulary-alignment-v1/native-report.py benchmarks/runs/vocabulary-alignment-v1/native-03
node scripts/rewrite-stage-command.mjs verify benchmarks/runs/vocabulary-alignment-v1/native-03
```

The runtime/model paths refer to locally provisioned artifacts; nothing is downloaded by this workflow. The native runner verifies the model against the catalog and records runtime hashes, exact wire requests/responses, token counts, source snapshots and failures. Native execution needs Metal and private Unix-socket access. Report replay needs neither model nor runtime. The native evaluator verifies run hashes, completeness, exact allowed outputs, canonical selector context and formatter source isolation.

See the [results](../../results/vocabulary-alignment-v1/README.md) and [implementation comparison](../../../docs/research/type4me-vocabulary.md).
