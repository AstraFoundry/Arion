# ASR, correction and intent rewriting ablation

Research-only local reproduction on macOS. This reproduces the three-stage
composition, not Apple's unpublished ECLM checkpoint or FastCorrect's reported
scores. Production defaults are unchanged.

## Treatments

| Arm | Actual model path |
| --- | --- |
| `asr` | Streaming X-ASR INT8, accelerated 20 ms chunks |
| `asr_correct` | Same actual first pass, then BART4CSC (zh/mixed) or FLANEC Base CD (en) |
| `asr_llm` | Same first pass, then the production Qwen3-1.7B worker |
| `asr_correct_llm` | Actual first-pass text, actual correction output, then the same worker |
| `offline` | Independent offline X-ASR on the same original audio |
| `offline_llm` | Actual offline result, then the same worker |

The LLM treatment includes the product's independent speech-span cleanup and
validation. Raw proposals, cleanup, verdict and actual delivered text are all
retained. No model receives intent references or outcome constraints. Scene is
author-supplied task metadata; scene classification is not evaluated.

BART4CSC is a 116M Chinese spelling-correction checkpoint, not ASR-specific
training. FLANEC Base CD is a 248M English ASR correction checkpoint trained
with N-best inputs. Here its published prompt receives one genuine hypothesis;
this is a top-1 adaptation, not a reproduction of its N-best scores. There are no
invented alternatives. These are independently selected language treatments,
not a single multilingual correction model.

FastCorrect's official Drive request failed with curl exit 35 before receiving
weights. Preserve the provisioning failure; do not call BART a FastCorrect
substitute in paper-score comparisons. NeuralSpeech code is MIT; downloaded
weight licensing must be independently checked before packaging. Model cards
mark BART4CSC Apache-2.0 and FLANEC MIT; upstream manifests retain exact revisions.

## Fixtures and execution

`build_fixtures.py` freezes one author's 24 new synthetic spoken challenges,
two strict silence controls and five development warmup cases before inference.
Test and development use different system voices. Test scripts cover Mandarin,
English and mixed technical speech, repairs, negation, uncertainty, fillers,
copy controls, quoted commands, email and list layouts. The labels live in
`labels.json`, separately from inference inputs. Test has only three spoken
voice identities; this is not population-level or spontaneous-speech evidence.

Requirements: pnpm as pinned in the root, uv, Python 3.12, macOS `say`, ffmpeg,
Go, provisioned X-ASR online/offline assets matching the existing benchmark
driver manifest, the existing phonetic Python environment with sherpa-onnx,
Qwen3-1.7B GenAI assets and staged native ORT libraries. `requirements-resolved.txt`
pins the separate correction worker environment. No Python environment ships
in the desktop application.

```sh
pnpm chain:env
pnpm chain:build
pnpm chain:provision
pnpm chain:fixtures
pnpm chain:run --split dev --out .workspace/correction-chain/dev-new
pnpm chain:run --split test --out .workspace/correction-chain/test-new
pnpm chain:report .workspace/correction-chain/test-new .workspace/correction-chain/report-new.json
node --test benchmarks/reproduction/correction-chain-v1/report.test.mjs
```

`EVAL_PYTHON` overrides the orchestrator interpreter; use Python 3.12. Correction
and acoustic workers explicitly use their separately provisioned environments.
Provisioning, fixtures and run directories must not exist. Commands never replace
existing results. Failed runs remain failed; provision both models before running.
Make equivalents are `chain-env`, `chain-build`, `chain-provision`, `chain-fixtures`,
`chain-bench SPLIT=test OUT=...` and `chain-report RESULTS=... OUT=...`.

## Measurement and interpretation

- Acoustic CER/WER/MER compares raw ASR, dedicated correction and offline control
  against the spoken script, separately by language. It excludes LLM editing.
- Intent constraints check authored required/forbidden substrings, with case folding
  and English word boundaries. Explicit numeric aliases accept spelled/digit forms.
  Silence requires a strictly empty result, including punctuation. Truncation fails.
- Intent reference overlap is a separate per-language edit proxy, not ASR accuracy.
  No aggregate mixes character, word and mixed units. Independent semantic/style
  ratings remain pending; neither overlap nor a constraint pass proves fidelity.
- Paired gains/harm compare the same inputs. Correction harm is relative to raw ASR;
  downstream benefit is `asr_correct_llm` versus `asr_llm`. Report unchanged correct
  source corruption and raw proposals separately from validated delivery.
- Models run resident, in sequential stage batches. ASR uses accelerated replay,
  not wall-clock speech pacing. Per-case serial estimates add actual ASR compute,
  correction parent latency and native rewrite parent latency. They exclude loads,
  batch gaps and native UI and are not observed end-to-end stop-to-insertion timing.
  Native rewriting shares its fixed instruction prefix and ends each private session.
- Process RSS is cumulative high-water, not per-request memory or combined pipeline
  residency. No energy, cold start, concurrent capture, Windows, IME or live-preview
  evidence is claimed. Host contention is not controlled.

`run.json` records source/model/runtime/audio identities, environment, hardware,
Git state and effective policies. All measured/warmup outputs and failed attempts
remain distinct. Reports reject incomplete status, modified outputs, duplicates,
missing coverage and broken stage lineage. Archive reusable fixtures and outputs
under `benchmarks/`; weights, environments and active execution stay in `.workspace`.
