# Expanded correction-chain study

A new frozen 100-case test set contains 96 spoken scripts (32 each Mandarin,
English and mixed technical speech) and four silence durations. Original 26 cases
remain a separate regression study and are excluded from primary estimates.
Two test voices per language, one author, paired themes and synthetic audio do not
supply 96 independent speakers. No population confidence or release claim follows.

The six model treatments reuse the original runner and provisioned binaries,
including BART4CSC for Mandarin/mixed, FLANEC for English and Qwen3-1.7B rewrite.
FastCorrect is not an executed treatment. Native rewrite cleanup/validation remains
part of the treatment; raw proposals and delivered outputs are retained separately.
Scripts and the prior flexible semantic rubric are frozen before inference. Exact
keyword gates are not used as semantic judgments in this study. Development uses
separate voices and old development scripts; it does not tune the new test labels.
All execution is local and uses accelerated audio replay and sequential stage
batches. Timings are engine estimates, not observed native insertion latency.

```sh
pnpm chain:expanded-fixtures
pnpm chain:expanded-run --split test --out .workspace/correction-chain-expanded-v2/run-new
```

Commands fail on existing destinations. Runtime environments and models must already
be provisioned using the original study's documented workflow. The wrapper sets
only the corpus and adds source provenance; it reuses actual model inference and
retains the original decoding policies. Completed evidence is copied byte-for-byte
to `benchmarks/runs/`; in-progress execution stays in `.workspace/`.

## Review and replay

The [completed result](../../results/correction-chain-expanded-v2/README.md) records
all 600 annotations and limitations. A single non-blinded agent reviewed script,
intent and delivered output; this is not independent human review. Baseline drafts
are retained. `author_ratings.py` binds explicit decisions to output hashes; it is
not an automated semantic judge. `review.mjs` exports the paired delivered text.

```sh
pnpm chain:expanded-report benchmarks/runs/correction-chain-expanded-v2/test-run benchmarks/results/correction-chain-expanded-v2/ratings.json .workspace/expanded-replay-new.json
pnpm chain:expanded-test
pnpm chain:expanded-verify
```

Report replay needs no models or private environments. It verifies complete original
observations and stage lineage, then aggregates explicit ratings. Test coverage
includes modified observations and arm-dependent grading of identical output.
Current review/report source and inference-time source snapshots are separately
retained because review tooling was completed after inference began. All model
inputs and decoding behavior stayed fixed throughout the run. The archive manifest
pins both snapshots, corpus, outputs, warmups, logs, reviewer drafts and results;
it excludes model weights and private runtimes.
