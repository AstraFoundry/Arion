# PR 40 / PR 45 native keyboard comparison

## Decision

PR 40 is the stronger base for lowercase English candidate ordering and canonical technical spelling in this authored set. PR 45 adds substantially more complete explicit-case handling and bidirectional mixed partial conversion. Neither version wins the measured native typing latency comparison convincingly. Preserve PR 40's English ordering while integrating PR 45's case/range-aware mixed conversion; do not replace one complete branch with the other.

## Frozen experiment

- PR 40: `d1aaffdd43db876da9f8584ed4074830a293d9cb`.
- PR 45: `264b882fc1f83adfc29285d3e80324f308734671`.
- Apple M4, 32 GiB, native arm64 macOS. Both production source/runtime/data closures were independently staged and hash-checked before and after measurement.
- 113 shared authored inputs; 1,476 main native observations: five alternating nine-slot rounds, one five-slot pass, five 80 ms paced rounds on twelve selected inputs. All five main repetitions produced identical candidates and functional outcomes. No main native process or encrypted-storage failure occurred.
- References stay in the external evaluator. Each observation uses a new process and empty encrypted synthetic profile, two cancelled warmups, production ArionTyping/menu code and an owned NSTextView. Typo/fuzzy remain off; Emoji/context ranking remain on with empty personal history.
- Original PR 40 fixtures are exposed regressions; additions were frozen before comparison and are not independently reviewed. Counts are task outcomes, not general-user accuracy.

## Candidate and commit outcomes

| Task | PR 40 | PR 45 |
| --- | ---: | ---: |
| Complete English: target spelling first, ignoring case | 30/37 | 5/37 |
| Complete English: target spelling on first nine, ignoring case | 36/37 | 30/37 |
| Complete English: exact requested case on first nine | 35/37 | 14/37 |
| Prefix completion: exact target first | 10/24 | 0/24 |
| Prefix completion: exact target on first nine | 17/24 | 16/24 |
| Explicit user case: candidate selected and committed once | 0/12 | 12/12 |
| Mixed segments: exact case and one completed commit | 4/12 | 10/12 |
| Chinese control: first page has no Latin-only entries | 25/28 | 28/28 |

Both variants put Chinese first on all 28 Chinese-control inputs. PR 40 exposes English forms on `li`, `he` and `wo`; these are ambiguous spellings, so a quieter page is not a universal intent-accuracy result. PR 45's stronger suppression also makes intended English harder to reach.

Technical canonical-case tasks are deliberately separate from case-insensitive coverage: e.g. lowercase `github` is present in PR 45, but it does not satisfy a requested `GitHub`. Explicit-case PR 40 failures mean no matching IME candidate/owned composition; ordinary host passthrough can still type literal `Hello`. They do not mean the editor cannot receive uppercase characters.

### Concrete candidate positions

| Input → target | PR 40 position | PR 45 position |
| --- | ---: | ---: |
| `hello` → `hello` | 1 | 4 |
| `artifi` → `artificial` | 1 | 5 |
| `compu` → `computer` | 1 | 9 |
| `softw` → `software` | 1 | 7 |
| `hel` → `hello` | 3 | 25 |
| `github` → `GitHub` | 1 | Absent in first 189 |

The five-slot pass also favors PR 40 for complete English: exact requested-case targets are visible in 35/37 versus 4/37 tasks. Prefix targets appear in 16/24 versus 6/24 tasks. This pass has one functional observation per input/variant; it supplies no independent speed estimate.

## Mixed follow-ups

Two additional 24-process functional runs separate case fidelity from segmentation and default ranking. They are excluded from latency statistics.

- Selecting each segment while ignoring case succeeds in 4/12 PR 40 and 11/12 PR 45 cases. PR 45 still fails `githubnihao`: no separately selectable `github` prefix was found within the same 189-candidate bound.
- Always selecting the current first candidate succeeds in 0/12 cases for both variants. `nihaohello` commits `你好合理咯` and `hellonihao` commits `合理咯你好` on both. These are actual default-candidate selections, not inferred or simulated commits.
- PR 45's gain is explicit partial conversion and case handling; this study does not establish automatic whole mixed-sentence quality.

## Native latency and memory

Only inputs with exact successful IME commits on both versions enter paired speed comparisons. No missing-candidate search, unhandled-key passthrough, reference lookup, JSON/controller IPC or pacing sleep is credited as a typing speed improvement. Each row is the median of five per-round p95 values; ranges are the five round p95 values, not confidence intervals.

| Workload | Paired inputs per round | PR 40 p95 median [range], ms | PR 45 p95 median [range], ms |
| --- | ---: | ---: | ---: |
| accelerated | 68 | 0.477 [0.214, 0.666] | 0.586 [0.264, 0.882] |
| paced | 6 | 1.673 [1.239, 1.780] | 1.650 [1.628, 2.017] |

The accelerated median favors PR 40, but the inter-round ranges overlap substantially and desktop load varies. The paced medians are effectively equal in this small shared-success subset. No clear latency winner is established. Native timing covers synchronous key handling, menu generation and NSTextView marked-text state; it does not include system IMK dispatch, a visible candidate window, compositor/GPU presentation or Windows.

On the 68 paired accelerated inputs, the median of five per-round process-RSS medians is about 104.3 MiB for PR 40 and 158.3 MiB for PR 45. These are fixture process peaks, not total application/ASR memory. Startup medians are approximately 76.1 and 69.4 ms with substantial overlapping variation; initialization includes immutable-data verification and text-store creation, excludes process launch and performs no OS-cache eviction.

### Why the timing is limited

Four separate unreported 12,000-iteration sampled workloads (`computer` and `artifi` on each version) show actual main-thread native work. The key-handler subtrees spend substantial time in AppKit marked-text replacement/text-layout/selection maintenance and Rime candidate generation/materialization. PR 40 reaches BilingualTranslation/ScriptTranslator; PR 45 reaches ScriptTranslator/TableTranslator and menu preparation. Driver snapshot/allocation/cancellation work also appears outside the timed key-handler subtree, so full-profile percentages must not be treated as a pure lookup breakdown.

The native measurement is not a dictionary-lookup microbenchmark. Optimizing lookup alone cannot remove the shared AppKit/producer costs observed in these stacks. Profilers ran only after all measured rounds. Background applications were observed rather than stopped; no CPU/OS isolation or performance-equivalence proof is claimed.

## Replay and evidence

The [JSON report](report.json) contains every unique case's first page, exact/folded ranks, target lookup rank, final text, insertion counts and round timing distributions. [Candidate examples](candidate-examples.csv) provide a smaller derived table.

[archive.json](archive.json) pins 5,022 original files by size/SHA-256 and their original paths; [evidence.tar.gz](evidence.tar.gz) preserves their original bytes. It includes main requests/results/logs, smoke observations, profiler stacks, follow-ups, source/runtime/build identities and a preliminary preparation failure. The first smoke invocation started before the PR 45 build identity existed, failed before native execution, and was retained; the subsequent 12-process smoke passed. Native executables, SDKs, compiled runtime/data, keys and profiles remain outside the archive.

```sh
python3 benchmarks/reproduction/ime-pr-comparison-v1/verify.py benchmarks/runs/ime-pr-comparison-v1 --negative-tests
```

The verifier checks every original byte and reproduces the entire report without loading Rime or accessing the original scratch paths. Six additional native negative requests verify both drivers reject reference-label fields before startup. Sources and procedure: [reproduction](../../reproduction/ime-pr-comparison-v1/README.md), [corpus](../../corpora/ime-pr-comparison-v1/README.md).

## Verification result

[Verification](verification.json) passed every retained byte, exact report replay and rejection of incomplete, corrupted, duplicate and reference-leaking evidence. `pnpm check` passed 559 Markdown documents, JSON/JavaScript syntax and Go boundaries; `git diff --check` passed. [Checks](checks.json) records these outcomes. Native negative requests rejected all six injected label fields.
