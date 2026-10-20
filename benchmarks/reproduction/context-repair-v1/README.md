# Context-supported ASR-text correction diagnostic

The desired contract allows a plausible ASR word error to be corrected from
relevant OCR context. Source/candidate string differences are not inherently
errors. Evaluation compares against the intended message plus retained
conditions, quantities, uncertainty and actor/action relationships. Context
instructions or unrelated facts may not become dictated content.

The frozen corpus has four correction families (recipient name, platform term,
function identifier, ordinary homophone), each with wrong+relevant, clean+relevant,
wrong+no-context and wrong+unrelated-context controls. Eight additional controls
cover conflicting dates/amounts, ambiguous names, bad OCR, hostile instructions,
negative questions, conditional facts and unrelated facts. Production, structural
v3 and context-repair candidates each produce all 24 observations.

All inputs are authored ASR-like/OCR-like text. No audio was recognized and no
OCR capture was executed. This measures local LLM behavior given those inputs,
not acoustic error frequency, OCR accuracy or end-to-end product acceptance.
All cases are development data; the structural anchors are exposed demonstrations.
No independent human review or held-out performance is claimed.

`expectations.json` is evaluator-only. The runner transmits only scene, source,
context and an opaque request ID; no intended text or judgment is sent. Positive
context examples teach correction in each scene; scene examples retain natural
organization. Frozen inputs and templates are never tuned from these outputs.

Use the pinned pnpm from the repository root. Inference requires the
[v1 isolated build and installed macOS runtime/model](../scene-rewrite-alignment-v1/README.md):

```sh
pnpm rewrite:context:run --out .workspace/context-repair-new-development
```

Inspect every source/context/output pair and record source-hashed `review.json`
using the archived schema. Fidelity here includes required supported spelling
repair in `wrong-relevant` cases; preserving an erroneous ASR name is recorded
as a missed correction, distinct from inventing a new fact. Obvious ordinary
spelling cleanup can also be legitimate without context. Other controls preserve
coherent source facts when context is unrelated, stale, corrupt or ambiguous.
Mechanical validator acceptance and semantic review stay separate.

Without model inference, replay and verify the completed evidence:

```sh
pnpm rewrite:context:verify
pnpm rewrite:context:test
pnpm rewrite:context:report --context benchmarks/runs/context-repair-v1/development --structure benchmarks/runs/scene-structure-v3 --out .workspace/context-report-replay
```

Make exposes `rewrite-context-bench` (`OUT`), `rewrite-context-report` (`CONTEXT`,
`STRUCTURE`, `OUT`), `rewrite-context-verify` and `rewrite-context-test`.
Output directories must be new. Model-free report validation checks complete
unique coverage, frozen input/template identity, raw results and each reviewed
source/output hash. Archive hashes cover both experiments and their report.

Each case now displays its fixture-assigned scene, effective model scene, scene
evidence and worker-reported prompt version. Scene assignment was explicit in
these experiments; application/browser scene detection was not exercised.
`scenes.json` exposes the same metadata for all cases. The structural-only run
did not record a separate model-scene field, so its direct scene argument is
reconstructed from the frozen runner and explicitly labeled as such.
The earlier rendering and generator are retained under
`report-before-scene-metadata/`; raw observations and judgments are unchanged.
