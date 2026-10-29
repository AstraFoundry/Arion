# PR 40 / PR 45 native keyboard comparison

This study compares frozen production sources and packaged data from PR 40 and PR 45 on one shared authored input set. It measures candidate ordering, case-sensitive selection and exact-once mixed commits through the actual Rime engine, production `ArionTyping` and an owned AppKit `NSTextView`.

The driver receives raw letters and real page-relative indices. The external Python controller holds expected spellings and mixed-segment references. It never changes producer dictionaries, candidate scores, source code or input based on measured outcomes. Uppercase keys carry Shift. When the IME declines a key, the owned editor receives the ordinary host insertion and records it; this cannot count as a successful IME candidate selection.

## Reproduce

Freeze both Git heads and export `scripts`, `apps/desktop/ime`, `apps/desktop/desktop.json` and `package.json` into two separate snapshot roots. Preserve `versions.json` with `pr40` and `pr45` SHA values. No active checkout, installed app, input source or user dictionary is modified. Use each snapshot's production staging script:

```sh
node SNAPSHOT/scripts/stage-keyboard-runtime.mjs
node benchmarks/reproduction/ime-pr-comparison-v1/build.mjs SNAPSHOT NEW_BUILD_DIRECTORY
```

The build directories are named `build40` and `build45` beside snapshot directories `pr40` and `pr45`. Both use native Apple silicon, the same toolchain, `-O2`, the production encrypted provider with an injected synthetic key, and their own production Rime library/schema/dictionary. Rime PR 45 includes its reviewed source patch; PR 40 uses its pinned upstream release. The binaries are measurement tools, not installed IMK hosts.

```sh
python3 benchmarks/reproduction/ime-pr-comparison-v1/run.py WORK_DIRECTORY NEW_RUN_DIRECTORY
python3 benchmarks/reproduction/ime-pr-comparison-v1/report.py RUN_DIRECTORY --output NEW_REPORT_JSON
```

A smoke run accepts `--smoke` and is excluded from the main report. Main execution alternates variants per input, shuffles input order deterministically and uses five repetitions. Every observation starts a fresh process and encrypted profile. Shared `xuetang`/`heishan` warmups are cancelled without commits. Typo/fuzzy correction stays off; Emoji and context ranking retain enabled defaults with no prior personal history. Nine-slot pages receive five repetitions, five-slot pages one compatibility pass, and twelve paced inputs five repetitions at 80 ms between keys. Target lookup stops at 189 candidates; a missing target remains missing, not an engine error.

An integration comparison can reuse this exact corpus and driver with `--variants VARIANTS_JSON` on `run.py`, `profile.py`, `defaults.py` and `archive.py`. The JSON maps exactly two variant names to `{ "snapshot": "relative-snapshot-directory", "build": "relative-build-directory" }`; `versions.json` must use those same names and identify the source commit plus any uncommitted source-patch hash. `report.py` obtains names from the retained run metadata. Keep the successful main directory named `run-v1` for archive replay. Supplementary validation, superseded attempts and explicit delimiter controls may be retained under `validation`, `attempts` and `apostrophe-check`; they never contribute to the frozen corpus score.

## Measurement and scoring

- Key latency starts immediately before production `handleEvent` and ends after its synchronous native/menu/AppKit update (and ordinary host insertion when declined). JSON, controller IPC, pacing sleeps and later candidate paging are outside this interval. Per-key states prove input was consumed. Compare speed only on the same inputs with successful exact IME commits on both variants.
- Selection timing covers the production candidate-index/revision action and synchronous text-store update. Startup timing includes immutable-data verification, engine/session and owned text-store construction, excluding process launch. Peak process RSS includes the AppKit fixture. No OS-cache eviction is performed.
- Exact case and case-insensitive first-page coverage are separate. A lowercase dictionary result does not satisfy an explicitly requested canonical spelling, but its presence remains visible in the folded metric. Arbitrary user casing is a separate input group.
- Mixed success requires every explicit segment, the exact final concatenation, one insertion and no remaining marked/composing text. It is explicit partial selection, not automatic whole-sentence conversion.
- Chinese controls record whether a Chinese answer leads and whether English occupies the first page. These are intent fixtures, not universal disambiguation labels.
- Repetitions establish stability, not independent language samples. Existing PR 40 inputs are exposed regressions. Newly authored additions are frozen before comparison and not independently reviewed. No general-user accuracy, installed IMK dispatch, Windows or GPU-visible latency claim follows.

The controller verifies hashes for both native binaries, all frozen engine/runtime sources and data before and after the experiment. A report rejects incomplete runs, altered result bytes, duplicate observations, request/reference leakage and mismatched raw keys. Failures and preliminary attempts are retained separately. Profiles, SDKs, compiled runtimes and binaries remain in `.workspace`; archived evidence retains their hashes and source provenance.

## Unreported profiling and functional follow-ups

Run these after all measured repetitions, so profiling and extra workloads cannot perturb timing. The default strategy selects the first current candidate repeatedly; the folded strategy searches the same bounded menu for each requested segment while ignoring case. Neither follow-up changes the producer.

```sh
python3 benchmarks/reproduction/ime-pr-comparison-v1/profile.py WORK_DIRECTORY NEW_PROFILE_DIRECTORY
python3 benchmarks/reproduction/ime-pr-comparison-v1/defaults.py WORK_DIRECTORY NEW_DEFAULT_DIRECTORY
python3 benchmarks/reproduction/ime-pr-comparison-v1/defaults.py WORK_DIRECTORY NEW_FOLDED_DIRECTORY --strategy folded
```

The retained study uses sibling directories `profiles-v1`, `defaults-v1` and `folded-mixed-v1`. Report replay independently scores the latter two if present beside `run-v1`. `archive.py` preserves the original main/preliminary/profile/follow-up files and source identities in a new destination; compiled assets remain private.

```sh
python3 benchmarks/reproduction/ime-pr-comparison-v1/archive.py WORK_DIRECTORY NEW_ARCHIVE_DIRECTORY
python3 benchmarks/reproduction/ime-pr-comparison-v1/verify.py ARCHIVE_DIRECTORY --negative-tests
```

The [completed findings and replay](../../runs/ime-pr-comparison-v1/README.md) retain all unique case results and performance limits.

[Integrated PR 40 results](../../runs/ime-english-integration-v1/README.md) reuse the frozen inputs with PR 45 case, mixed, correction and personal-vocabulary features. `delimiters.py WORK_DIRECTORY NEW_OUTPUT_DIRECTORY --variants VARIANTS_JSON` supplies separate normal-apostrophe page controls.
