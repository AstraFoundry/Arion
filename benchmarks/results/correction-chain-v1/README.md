# ASR -> dedicated correction -> intent rewrite: six-arm diagnostic

Status: completed exploratory local inference on Apple M4, 32 GiB RAM, macOS
arm64, four CPU threads. The test contains 24 newly authored synthetic spoken
cases (12 Mandarin, eight English, four mixed) and two strict silence controls.
Five separate-voice development cases and all warmups remain separate. No
production model selection, pipeline policy or desktop default changed.

## Finding

The tested correction stage does not justify adoption. Its downstream constraint
benefit is small and mixed: full-chain delivery passes 16/26 versus 15/26 for
ASR -> rewrite, with three paired gains and two losses. Offline acoustic
recognition -> rewrite passes 19/26, with four paired gains and no losses versus
the streaming ASR -> rewrite arm. These are authored string proxies, not measured
semantic accuracy or statistical significance.

| Arm | Delivered constraint passes | Serial engine estimate p50 / p95 |
| --- | ---: | ---: |
| Streaming ASR | 10/26 | 1.26 / 4.38 s |
| Streaming ASR -> correction | 12/26 | 1.55 / 4.70 s |
| Streaming ASR -> intent rewrite | 15/26 | 2.52 / 13.64 s |
| Streaming ASR -> correction -> intent rewrite | 16/26 | 2.91 / 8.26 s |
| Offline acoustic control | 11/26 | 0.17 / 0.87 s |
| Offline acoustic control -> intent rewrite | 19/26 | 1.38 / 7.74 s |

Serial estimates sum each case's actually observed stage durations. They exclude
load, inter-batch gaps, real-time audio pacing and native UI. Do not interpret
these as microphone stop-to-insertion latency or the p95 differences as a stable
speed ranking. The largest retained case estimate is 61.97 s; no outlier was
removed. Host contention was not controlled. Correction generation p50/p95 is
0.218/0.407 s. Acoustic/correction process high-water RSS is approximately
1.01/1.70 GB; the native text benchmark does not populate rewrite RSS, which is
reported as null rather than zero. Stage residency is sequential, not a combined
pipeline memory measurement.

## Correction before rewriting

| Source | Mandarin CER (190 units) | English WER (81 units) | Mixed MER (53 units) |
| --- | ---: | ---: | ---: |
| Streaming ASR | 1.58% | 3.70% | 18.87% |
| Dedicated correction | 1.58% | 18.52% | 9.43% |
| Offline acoustic control | 0.00% | 2.47% | 20.75% |

Correction improves two utterance edit counts and worsens six; three of 16
originally correct cases are corrupted (the 16 include two silent controls).
BART4CSC is a 116M Chinese spelling-correction baseline, not an ASR-trained
ECLM. FLANEC Base CD is a 248M English ASR correction checkpoint; its published
N-best prompt receives one actual hypothesis here. Both are FP32 CPU; Qwen3-1.7B
is the existing INT4 native GenAI package. Neither the Apple paper's checkpoint
nor its reported results were reproduced. FastCorrect's official Drive request
failed before weights arrived; that attempt remains archived.

Examples retained in the actual traces:

- English repair: ASR retains both the superseded `twelve` and final `aid`
  (misrecognized `eight`); FLANEC outputs `Order twelve notebooks`, deleting the
  repair and final value. The full chain retains the wrong quantity.
- Recipient repair: ASR retains `Helen ... no ... Robert`; FLANEC drops the
  Robert clause. The full chain consequently addresses Helen.
- Email: direct rewriting retains the spoken `Thank you`; correction drops it,
  and the final chain cannot recover it. A complete three-stage trace does not
  imply preserved intent.
- Mixed input: BART joins `A P I` and `Mac O S`, enabling the downstream output
  to satisfy the API/macOS constraints. This is partly tokenizer/format
  normalization, not proof of learned acoustic disambiguation. No normalization-only
  control was run, so that mechanism cannot be isolated from model correction.

Both silent inputs stay empty through ASR and correction. The rewrite model emits
nonempty proposals that the validator rejects as markup; actual delivery remains
empty. Raw proposal and delivered results are retained separately.

## Scope and evidence

One author, synthetic read speech, three spoken test voices, no independent
semantic/style review and only one measured run per treatment. English aliases
and case folding are explicit; equivalent paraphrases outside the frozen labels
can still fail. Conversely, a pass can miss an invented fact or filler: the
English `huh` substitution passes the current `um`/`uh` constraint. The mixed
stratum has only four cases. Do not generalize these rates to real dictation.

The LLM treatment includes production `speech-spans` cleanup, `live-v2` prompting,
validation and explicit failure delivery. It is final-only, without live previews,
context, approved terminology or target insertion. Dedicated correction must not
silently remove spoken self-repair evidence needed by intent reconstruction.

- [Full test summary and per-case outputs](summary.json)
- [Development summary](dev-summary.json)
- [Frozen inputs, labels and original audio](../../corpora/correction-chain-v1/manifest.json)
- [Original attempts and source/model provenance](../../runs/correction-chain-v1/manifest.json)
- [Reproduction, treatment contracts and commands](../../reproduction/correction-chain-v1/README.md)

The archive keeps the failed host-Python preflight, failed model-readiness run,
superseded BART space-decoding development run, corrected development run and
complete test run. Output decoding was repaired from upstream source before
any test inference. No prompt or correction selection was tuned on test results.
A model-blinded export of 156 output tasks is prepared, explicitly unrated;
independent intent review remains pending.

```sh
node benchmarks/reproduction/correction-chain-v1/archive.mjs verify
pnpm chain:report benchmarks/runs/correction-chain-v1/test-run .workspace/correction-chain/replay-new.json
pnpm chain:test
```
