# PR 40 English with PR 45 keyboard features

## Result

The integrated implementation preserves PR 40's English lexicon and all 61 English-word/completion first pages in the frozen corpus. All 28 Chinese-control first pages also remain identical. Explicit-case candidate commits improve from 0/12 to 12/12, and explicitly selected mixed segments improve from 4/12 to 12/12. There is no clear native typing latency winner in this experiment.

The integration adds PR 45's typo correction, eleven independent fuzzy-Pinyin switches and explicit personal-vocabulary readings. These switches remain off and the personal projection empty in the paired default-policy comparison. Separate real native and Go tests exercise enabled correction, fuzzy edges, personal candidates, encrypted persistence, deletion, revision ownership and delivery recovery.

## Frozen comparison

- Baseline PR 40 is `d1aaffdd43db876da9f8584ed4074830a293d9cb`, using the exact original comparison binary. Integration starts at `7945cffb0c8fd946fe57729bd49cce3c323aaed6` plus the exact retained product-source patch. The later `4457357c` parent-SDK CI fix is retained in the feature branch and does not change the measured native sources.
- The same [113 authored inputs](../../corpora/ime-pr-comparison-v1/README.md), driver and compiler flags produce 1,476 successful observations. Five alternating nine-slot rounds, one five-slot pass and five 80 ms paced rounds use fresh processes and empty synthetic encrypted profiles. Repeated candidate/functional outputs are identical.
- Both binaries use their actual production Rime library, data, candidate producer, native menu and ArionTyping against an owned AppKit NSTextView. References stay in the external controller. No installed input source or personal vocabulary is changed.
- `arion_english.lexicon` remains 5,598,380 bytes with SHA-256 `6efff3453363bef601ef9e3a48d58575017146b723e1b36d8bf6df7bf632ba87` on both variants. Source forms, frequency ordering and admission policy remain PR 40's. The integration uses exact sorted-key proper prefixes and actual shorter candidate ranges; it does not import Easy English or PR 45's global sentence-generation expansion.

## Candidates and commits

| Task | PR 40 | Integrated |
| --- | ---: | ---: |
| Complete English: requested case on first nine | 35/37 | 35/37 |
| Complete English: case-insensitive target first | 30/37 | 30/37 |
| Prefix completion: requested target first | 10/24 | 10/24 |
| Prefix completion: requested target on first nine | 17/24 | 17/24 |
| Explicit case: exact-once candidate commit | 0/12 | 12/12 |
| Mixed segments: exact-case, exact-once commit | 4/12 | 12/12 |
| Chinese controls: no Latin-only first-page entries | 25/28 | 25/28 |

All 61 English and 28 Chinese first-page arrays are byte-for-byte equal by candidate text/order between variants. Five-slot outcomes preserve English coverage and also reach 12/12 explicit-case and mixed commits. PR 40's zero case score means the requested IME candidate is absent; ordinary host passthrough can still type uppercase text.

Supplemental controls execute `ni'hao` and `xi'an` on five/nine-slot menus. All four paired pages match exactly. They are excluded from the 113-case score and timing. An earlier implementation used empty delimiters in its normal-Pinyin guard, which could introduce Ni/Xi prefixes. The final source uses production space/apostrophe semantics. These controls and the native regressions exercise the repair.

Two separate 24-process follow-ups distinguish explicit selection from automatic ranking. Case-insensitive segment selection improves from 4/12 to 12/12. Repeatedly choosing the current first candidate improves from 0/12 to 3/12. Automatic mixed-sentence disambiguation remains limited; the 12/12 result requires actual per-segment candidate choices.

## Native latency

The following are medians of five round p95 values, with the observed round-p95 range. Only identical inputs that commit successfully through both IMEs enter the paired comparison. Pacing, controller/JSON work and candidate searches stay outside the key-handler interval.

| Workload | Shared inputs per round | PR 40 p95 median [range], ms | Integrated p95 median [range], ms |
| --- | ---: | ---: | ---: |
| Continuous input | 87 | 0.235 [0.220, 0.268] | 0.246 [0.230, 0.274] |
| 80 ms paced input | 7 | 1.968 [1.793, 2.387] | 1.908 [1.802, 2.195] |

The direction changes between workloads and round ranges overlap. These observations establish no clear latency improvement or regression. Fixture process peak-RSS medians are approximately 104.3 MiB versus 104.1 MiB. Background desktop activity is observed rather than isolated. These ranges are not confidence intervals or a performance-equivalence proof.

Four separate 12,000-iteration sampled workloads run only after measured rounds. The actual key-handler subtree includes AppKit marked-text/layout maintenance and Rime/menu candidate generation, including the bilingual producer. Driver snapshots and cancellation also occur outside that subtree. Full-profile percentages cannot be treated as dictionary-lookup cost. This measures synchronous native state, not installed IMK dispatch, visible/GPU presentation, Windows or total application memory.

## Integration validation

- `pnpm core:check` and `pnpm desktop:check` pass. The final desktop gate includes 83/83 real Rime/AppKit cases, all 98 original admission combinations, 4,016 normal-spelling graph controls, four typo operations, all eleven fuzzy pairs and actual personal-vocabulary prism/abbreviation/range/deletion checks.
- Go tests cover encrypted approved-reading projection, grouped vocabulary/ASR preservation, stale editor identity, publisher admission, full-snapshot delivery and cancellation/reconnect/busy handling. The new acknowledged A, rejected B, restored A regression receives success without another wire write.
- Native Go settings snapshots cover English/Chinese correction toggles, reading validation/stale states, light/dark editor variants and pending/failure notices. PNGs are kept on the separate image-only PR screenshot branch.
- The final universal input-method build and development bundle closure verification pass. Neither operation installs or launches an IMK server.
- The host has zero valid signing identities. `pnpm keyboard:vocabulary:check` fails at certificate preflight before signed native execution. PR 45's historical signed/installed results do not establish current integrated acceptance. Actual signed application/IMK delivery, installed restart/offline behavior and current Windows native/TSF execution remain unverified here.

## Evidence and replay

[report.json](report.json) retains every case and per-round distribution. [Preservation](preservation.json) verifies the unchanged English data/pages and matches all 128 current native source files to the measured snapshot. [Candidate examples](candidate-examples.csv) provide derived ranks/pages. [archive.json](archive.json) pins every archived file by original path, size and SHA-256; [evidence.tar.gz](evidence.tar.gz) preserves source, requests, per-observation outputs, profiler stacks, follow-ups and validation logs. Compiled libraries, binaries, runtime environments, keys and profiles remain in `.workspace`.

The first integration benchmark was superseded after review found the delimiter and vocabulary-observer bugs. It retains 1,394 observations, including the intentionally terminated final observation and explicit invalidation reason. Its timings do not enter this report. Earlier transplant compile/assertion failures, the source-stability check invalidated by README edits, and the signing preflight failure remain archived alongside corrected successful runs. An unrelated log-path failure happened before test execution and is recorded separately.

```sh
python3 benchmarks/reproduction/ime-pr-comparison-v1/verify.py benchmarks/runs/ime-english-integration-v1 --negative-tests
```

[Reproduction](../../reproduction/ime-pr-comparison-v1/README.md) describes named variants. `delimiters.py` reproduces the supplemental page controls with the same variant mapping. [Verification](verification.json) records original-byte validation and exact report replay, plus rejection of incomplete, altered, duplicate and reference-leaking results. [Checks](checks.json) records final repository checks and execution limits. This small authored set is regression evidence, not general-user accuracy.
