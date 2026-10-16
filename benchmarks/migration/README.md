# Benchmark research archive

Reusable benchmark inputs and evidence have been copied from the local scratch workspace into `benchmarks/`. Original local files remain available. No historical corpus, observation, prompt, run status or published JSON result was rewritten. Git attributes disable line-ending normalization for benchmark assets and preserve original whitespace in frozen evidence, including historical logs. The archive is retained in Git; temporary execution data remains in the local workspace.

## Retained evidence

[archive.json](archive.json) maps every original relative path to an archived path and pins its exact SHA-256 and size. Its historical checkout root resolves old absolute references without changing the evidence. [import-plan.json](import-plan.json) records the explicit source/destination selection. The [earlier inventory](inventory.json) is the pre-migration audit and remains unchanged, including its then-incomplete phonetic observations.

| Category | Files | Bytes |
| --- | ---: | ---: |
| Corpora, observations, warmups, metadata, prompts, scripts, source snapshots, logs and published JSON | 611 | 35,616,297 |
| Recorded WAV/PCM and intermediate AIFF audio | 359 | 60,504,132 |
| Authored context HTML and rendered PNG fixtures | 92 | 1,952,571 |
| Total | 1,062 | 98,073,000 |

The archive includes eight report families plus their shared inputs, exploratory experiments and failed/cancelled attempts. It is not 1,062 independent tests. The 22 excluded compiled executables/other artifacts have retained identities in the manifest; weights, environments and installed competitor binaries are not bundled. No external binary archive has been uploaded by this task.

The new phonetic material includes the cancelled two-model parent, its two measured 4B generations/warmups, a complete 1.7B checkpoint view, offline acoustic control, confidence results and the earlier invalid attempt. Cancellation remains `cancelled_by_user`. The older model-selection attempt still records `running`; it is an incomplete historical snapshot and is not presented as a complete measurement.

## Layout

- [Corpora](../corpora/README.md): case definitions, references, constraints, original speaker/session/split identities, exact audio and authored context fixtures.
- [Runs](../runs/README.md): actual observations, warmups, run/failure metadata and logs, grouped by study.
- [Reproduction](../reproduction/README.md): frozen experiment scripts, prompts, evaluators, baseline sources and original report configurations.
- [Artifacts](../artifacts/README.md): acquisition/runtime manifests and identities for artifacts that must be provisioned separately.
- [Results](../README.md): existing human-readable reports and aggregate JSON, with unchanged measured values.

A shared corpus is retained once. File versions with matching case IDs are not silently merged. Corpus provenance sidecars describe migration and study use without changing the corpus bytes. Historical exposed test sets remain useful regression data; they cannot become an unseen future test set.

## Verify and replay

Use the exact pnpm version from `package.json`:

```sh
pnpm bench:archive:verify
pnpm bench:archive:report --out .workspace/archive-report-repeat
```

Equivalent Make entries are `benchmark-archive-check` and `benchmark-archive-report OUT=...`. Report output directories must be new and outside the immutable benchmark archive. These commands do not load a model, start an inference worker or acquire weights.

Verification checks every archive entry, the selected source identities, 37 metadata-to-result/warmup hash links, 347 corpus-to-audio references, the exact 1.7B selection from the cancelled parent and all eight confidence strata. Report replay regenerates and compares:

| Report family | Comparison |
| --- | --- |
| Disfluency alignment v1 and v2 | Every published field except report-generation timestamp |
| Live rewrite, context ablation and scene switching | Every published field except report-generation timestamp |
| ASR model selection | Complete JSON |
| Qwen diagnostic aggregate | Complete JSON |
| Phonetic 1.7B primary aggregate | Complete JSON |

The current live scorer adds fields that were absent from older reports; extra fields are allowed, but every historical field must match. [verification.json](verification.json) records the actual archive/replay check. The [relocated-checkout check](clean-check.json) also passed all eight families using a copy containing only benchmark assets and evaluation tooling, without the original workspace, models or environments. Edit-learning native measurements, scene classifier, prefix/OCR subreports and the phonetic supplemental report are retained and hash-verified; this command does not regenerate those subreports. Confidence groups are separately recomputed.

The archive reader uses only declared archived entries, rejects changed bytes, path traversal, symlinks and unmapped paths, and never substitutes the developer's old workspace. The phonetic CLI supports `--archive benchmarks/migration/archive.json` only for report replay; ordinary inference retains its existing explicit paths and artifact checks.

## Historical provenance and remaining boundaries

- Original `run.json`, source snapshots, prompts, raw rows and published JSON remain byte-identical. Live report configurations now point to archived inputs; each records its original configuration path/hash. Absolute paths inside old metadata are historical evidence, resolved through the explicit map for scoring.
- Original scripts in `reproduction/` retain old imports and machine paths. They are evidence, not automatically portable inference commands. Use current evaluation entry points and provision declared model/runtime artifacts for new inference. Exact excluded executable bytes are not available from this Git archive.
- TTS regeneration can vary with OS/voice versions. Recorded audio bytes are retained, including intermediate AIFF files, rather than pretending a builder guarantees the original waveform.
- Source dataset revisions, declared terms and existing attribution notes are retained. Authored inputs without a licence field receive provenance sidecars, not a newly invented licence assertion. The migration does not change the recorded dataset terms or product distribution decisions.
- Installed competitor code, account state, private app data and unrelated personal recordings are outside this archive. The context images retained here were generated from authored test windows.
- Failed prompts, output-contract mismatches, partial runs, cancelled checkpoint observations and reused test data retain their limitations. Archiving does not promote them to complete inference, independent review, native insertion or release acceptance.

The original workspace was not deleted or relocated, and no cancelled inference was resumed. Future experiments should run in new scratch output paths, then publish an explicit versioned research snapshot using this retention contract.
