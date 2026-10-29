# Current-candidate LLM ranking inputs

Reuse the existing production native engine compiler and `arion-ime-rank-eval` completion driver. This version replaces v1's historical no-span candidates with current real first-page candidates and exact consumed UTF-8 ranges. It does not modify the frozen v1 experiment or its historical scores.

```sh
node scripts/stage-keyboard-runtime.mjs
node benchmarks/reproduction/imk-quality-v1/run.mjs prepare .workspace/benchmarks/new-ranking-build
node benchmarks/reproduction/llm-ime-alignment-v2/prepare.mjs .workspace/benchmarks/new-ranking-build .workspace/benchmarks/new-ranking-inputs
node --test benchmarks/reproduction/llm-ime-alignment-v2/requests.test.mjs
node benchmarks/reproduction/llm-ime-alignment-v2/verify.mjs benchmarks/runs/llm-ime-alignment-v2
```

Preparation runs 101 independent native processes: one unrelated warmup input and all 100 exposed authored contextual fixtures from rime-llm-reranker-v1. Each process uses a fresh encrypted synthetic profile and two cancelled engine warmups. Typo correction is on, fuzzy pairs off, English enabled. Reference-free model requests contain two warmups plus paired original-context/no-context requests, deterministically ordered by SHA-256 of seed 42 and session ID. The same native warmup snapshot supplies both model warmups.

Only id/input/warmups/spelling cross the engine boundary. Only identity, raw Pinyin, original context and real candidate id/text/type/range cross the model boundary. `acceptable` and `purpose` stay outside both drivers. Candidate ranges are validated with the shared native evaluator. Tests reject missing coverage, mismatched producer identity, invented text and invalid endpoints, and preserve same-input alternatives of different output lengths.

Retain original sources, requests, native outputs, driver logs and hashes when archiving a new run. The checked-in historical runs use hash-pinned `evidence.tar.gz` bundles; `verify.mjs` expands them temporarily and cleans up after replay. The verifier validates manifest bytes and source identities, then reconstructs every native and model request from actual snapshots and source corpus. Prepared status is not successful LLM inference. Existing v1 report code assumes equal text length and must not score v2. `report.mjs` now validates span-equivalent stable permutations, identities, failures and full denominators; actual current inference is archived separately in `benchmarks/runs/llm-ime-alignment-v2-inference`. Local model weights, runtime and executables remain in the workspace.


## Actual local execution and report

Build the existing `arion-ime-rank-eval` and `arion-llm` commands with `CGO_ENABLED=0`. The exact local invocation is retained in the inference archive's `run/environment.json`; paths refer to the original workspace. Provision matching paths explicitly; never use stored cloud credentials. Verify the catalog model hash and pinned runtime archive/member bytes before running. Preserve exit status, source hashes before/after, command log and original observations even on failure.

```sh
node benchmarks/reproduction/llm-ime-alignment-v2/report.mjs EXPANDED_CANDIDATE_ARCHIVE NEW_INFERENCE_WORK NEW_REPORT.json
node --test benchmarks/reproduction/llm-ime-alignment-v2/report.test.mjs
node benchmarks/reproduction/llm-ime-alignment-v2/verify.mjs benchmarks/runs/llm-ime-alignment-v2 benchmarks/runs/llm-ime-alignment-v2-inference
```

Report rejection tests cover partial-span promotion, invented/duplicate candidates, stale revision, nonfinite scores, hidden no-context inference, failed status, incomplete observations and explicit-error denominators. Preserve preparation and inference as separate archives, including their original manifests. Post-run evaluator snapshots are labeled separately from executed model sources.

## Historical versus current execution

The two checked-in bundles were executed before main's bilingual integration. Their original source manifests remain authoritative. `verify.mjs` accepts compact bundles or original expanded directories. `report.mjs` accepts expanded directories; it never runs inference. To inspect the historical preparation manually, extract its `evidence.tar.gz` into a new workspace directory. For a current experiment, rebuild the native driver and prepare fresh requests using the commands above; historical quality numbers must not be reused for the changed producer. The v1 live preparation emits no actual ranges and is not a v2 inference input. Its frozen report replay remains valid.

[Post-rebase request preparation](../../runs/ime-ranking-range-rebase-v1/README.md) verifies the integrated main producer and retains the rejected stale-runtime attempt. It is independent of the historical inference scores.
