# Keyboard and voice vocabulary: practice and evaluation plan

Status: proposed implementation and experiment design. No bridge, model, privacy default, persistent history or acceptance threshold is adopted by this document. No new inference has been run for this plan.

## Objective and evidence boundary

Make a confirmed personal word useful across keyboard typing and speech recognition, with observable provenance, deletion and bounded inference cost. Evaluate the complete acquisition-to-reuse loop, not just a prefilled word list.

The [competitor investigation](keyboard-voice-lexicon.md) establishes different client mechanisms:

| Observed mechanism | Arion proposal | Claim boundary |
| --- | --- | --- |
| Doubao exports selected Pinyin user words into speech hotwords | Retrieve approved personal spellings for local acoustic finalization | No copying proprietary dictionaries or claiming the same model/filter policy |
| Doubao searches ASR entity records during Pinyin conversion | Expose eligible personal terms to a native keyboard candidate producer | Candidate creation is distinct from re-ranking existing candidates |
| Chatterfly includes per-app committed-input history in ASR context | Later experiment with authorized session-only context | Persistent typing-history collection is outside the current privacy contract |
| Chatterfly builds voice-term data consumed by typing candidates | Versioned personal-term projection into an IMK-owned encrypted cache | Account sync is independent and remains deferred |

Existing Arion audio experiments support compact destination-only context as a candidate, not a release decision. [v2](../../benchmarks/results/user-modify-asr-v2/README.md) improved synthetic term hits from 7/20 to 11/20. The [empty-vault v3 loop](../../benchmarks/results/user-modify-asr-v3/README.md) improved 3/72 to 33/72 but worsened one utterance; routed extra compute was p50 739 ms / p95 1,204 ms. Those results used a research MLX conversion, authored speech and simulated corrections. They do not validate the shipped recognizer, keyboard acquisition, native host integration or natural-user accuracy.

## Current implementation and gaps

| Area | Existing implementation | Required extension |
| --- | --- | --- |
| Voice memory | `vocabulary.bin` stores approved `From`/`To` replacements; domain matching and edit learning | Represent a standalone correct term without inventing a wrong form |
| Keyboard learning | IMK-owned encrypted engine learning and context-ranking snapshots | Emit only eligible confirmed lexical selections with pronunciation provenance |
| ASR | `FinalizeRequest` carries audio and draft; `sherpa.go` decodes without per-request vocabulary | Domain-owned session vocabulary request and explicit runtime capability |
| Rewrite | Relevant replacement pairs reach the optional rewriter | Keep separate from recognition-stage treatment and evidence |
| Cross-process transport | Voice composition bridge and keyboard preference file | Bounded vocabulary transport, revisioning, deletion and identity authorization |
| Keyboard candidates | Actual engine candidates plus permutation-based context ranking | Personal-term producer for terms absent from the engine's base candidates |

Implementation entry points are `packages/core/domain/{memory,vocabulary}.go`, `domain/ports/ports.go`, `packages/core/application/dictation.go`, desktop `bootstrap/{app,learning}.go`, `infrastructure/{vocabulary,asr,imebridge}`, and `ime/macos/{keyboard,typing,learning,ranking}`. These paths are ownership references; new APIs below are proposed rather than existing symbols.

## Proposed product and data contract

Use one domain-level personal-term registry with separate projections. Retain the independently licensed keyboard base dictionary, engine frequency/bigram learning and explicit correction-pair rules in their current roles.

| Record | Stored meaning | Allowed reuse |
| --- | --- | --- |
| Personal term | Correct spelling, language and confirmed pronunciation/alias evidence | ASR spelling hint; native keyboard personal-term candidate |
| Correction pair | Explicit observed error and corrected text, with original provenance | Existing correction rules; optional destination spelling projection after eligibility checks |
| Keyboard statistic | Engine selection frequency or previous-word association | Keyboard ranking; no automatic global replacement |
| Temporary context | Already authorized cursor/OCR/recent-dictation context | Session-only retrieval/meaning evidence; no automatic persistence |

Proposed term fields: opaque ID, exact display spelling, normalized lookup form, language, pronunciation values with their sources, origin (`manual`, `keyboard_selection`, `voice_edit`), approval/scope, independent use counters by source, logical revision and tombstone. Normalization affects lookup only and must not silently alter the spelling. Source observation IDs support idempotence; physical clock order alone cannot resolve conflicting edits. Do not store surrounding sentences, raw key events, target text or application titles in these records.

Acquisition rules:

- Manual standalone terms can be used without a replacement pair. Existing pairs remain intact during migration. Promote a pair's destination only when it is a complete eligible word; historical fragments are not expanded by guessing their missing context.
- With cross-mode learning explicitly enabled, a confirmed lexical keyboard selection can contribute a personal term. A highlighted candidate, preedit, raw Enter commit or a whole sentence is not a lexical selection. Read pronunciation/segmentation from the actual engine result; if unavailable, record an explicit unsupported acquisition outcome. Do not infer a surname's pronunciation from a generic character converter.
- Emit the event after the corresponding current composition's actual host insertion succeeds under the native adapter contract. A cancelled selection, stale callback, rejected insertion or unverified delivery cannot claim confirmed acquisition. Engine-private learning timing is not evidence of successful host insertion.
- Keyboard selection is evidence of spelling and pronunciation, not an authorization to replace a homophone everywhere. For example, selecting a person's name must not make the ordinary phrase with that pronunciation disappear.
- Existing voice edit detection remains governed by ADR 014. An edit pair does not prove that the destination is a globally appropriate keyboard word. Ordinary semantic changes and uncertain word boundaries stay out of automatic cross-mode promotion.
- No exposure counter or ASR self-output may approve a new spelling. Separate acquisition events from reuse events so hints cannot reinforce themselves indefinitely. A voice result using a keyboard-origin term does not count as a second independent teaching event.

Cross-mode reuse should have its own disclosed control, initially off pending the experiment. Reuse the existing English/Simplified Chinese copy catalogs during implementation. Show source and removal controls with learned terms. Turning reuse off stops export/recall and invalidates derived caches; deletion propagates a tombstone. Reset keyboard learning removes keyboard-origin contributions without deleting a separately approved manual contribution to the same spelling. A personal-term reset deletes all term contributions and projections; correction-pair reset remains a distinct existing operation.

This changes reuse scope and local persisted data. Before production implementation, record an ADR and amend [003](../../specs/003-memory-sync.md), [005](../../specs/005-privacy.md), [007](../../specs/007-ime-ranking.md), [architecture](../project/architecture.md) and the applicable IPC contract. The experiment runs in an isolated explicitly enabled research profile first. This proposal does not authorize persistent typing history or account upload.

## Storage, ownership and failure behavior

Recommended ownership: the desktop application is the single writer of the semantic term registry; IMK is the single writer of native keyboard learning, an encrypted pending-term journal and its derived personal-term cache. Never have both processes mutate the same encrypted snapshot or read engine-private userdb internals as a public interchange format.

1. IMK emits a bounded confirmed-word event off the key path. A small encrypted journal records event ID, source generation, term/pronunciation and source revision. It is a term journal, not a sentence/history log. Coalesce repeated word observations and enforce a declared cap. Overflow and persistence failure produce an explicit acquisition failure, with no claim that the term was saved.
2. When the desktop app is available, an authenticated, versioned local bridge imports events idempotently. Acknowledgement occurs only after the semantic registry is durably written. Crashes between persistence and acknowledgement cause a replay of the same IDs, not duplicate use counts.
3. The desktop publishes a bounded revisioned projection of eligible terms. IMK validates and persists it using its existing device-only encryption boundary, then rebuilds the personal-term candidate source away from the key path. The bridge does not transfer either process's encryption key.
4. The desktop session reads an immutable term snapshot. Model calls receive this snapshot's selected terms and revision; concurrent edits cannot change an in-flight request. Disable/delete/reset invalidates dependent work through the coordinator's generation mechanism before delivery.
5. If the desktop is stopped, ordinary offline typing and local engine learning continue; pending term events wait within the cap. If the bridge or projection is unavailable, report that cross-mode reuse is unavailable. Do not bypass key/authentication failures, silently substitute plaintext, or report a memory-enabled benchmark as executed without memory.
6. Reset/disable advances a persisted source epoch. Delayed journal events and acknowledgements from older epochs cannot resurrect removed contributions. Bound tombstone/ack retention with a full-snapshot resynchronization protocol for lagging clients; do not discard tombstones solely by age.

The bridge must validate peer identity, payload size, schema, source epoch and revision using the existing local IPC security conventions. Prove that the signed desktop and IMK profiles can access only their intended keys. A shared Keychain access group is not assumed; preserve the separate device-only IMK key. Account sync and its future key/outbox protocol remain independent.

## Recognition and keyboard consumers

### Recognition

Start with local acoustic finalization; preserve the existing streaming preview. Add a typed, bounded personal-term context to the domain-owned finalizer request and an explicit capability for the selected bundle. Do not mutate resident recognizer-wide hotword configuration across concurrent sessions. Confirm that the pinned runtime really consumes per-stream vocabulary; unsupported models produce an explicit capability outcome.

The first candidate policy ranks eligible terms using confirmed spelling/pronunciation aliases against the available first-pass draft, then source priority, confirmed frequency and recency. Send only correct destination spellings into ASR. Wrong forms are retrieval keys, never recognition instructions. Ambiguous competing terms can abstain. Text-derived pronunciation is lexical retrieval evidence, not an acoustic posterior. An empty result records `no_retrieval`; a failed finalizer records a failed treatment rather than silently becoming a successful memory-off observation.

Development sweep: top-K in `{2, 4, 8}`, tokenizer budgets in `{32, 64, 128}`, with a hard maximum selected in development before the held-out run. Count tokens with the actual model tokenizer. If no admissible tokenizer-bound interface exists, mark that configuration unsupported rather than applying an approximate hidden truncation. Neither Doubao's 500-code-unit bound nor Chatterfly's 6000-byte policy becomes an Arion default by imitation.

Test same-audio finalization first. Extra re-decoding after an already completed acoustic finalizer is a separate, more expensive treatment. Preserve audio only in bounded session RAM, cancel on focus/session change and keep raw recognition, hinted recognition, rule-corrected text and rewritten output separately observable in private evaluation.

### Keyboard

Provide personal terms through a native candidate producer, with spelling, declared pronunciation and source identity. A term missing from the original engine candidate set cannot be created by the existing permutation-only ranker. First demonstrate a supported custom dictionary/translator integration against the pinned engine with actual terms absent from the base dictionary; do not simulate success by renaming an existing candidate or committing an arbitrary string.

Keep original engine and personal-term candidate identities stable across paging, partial selection and rank changes. Update the projection between compositions, never midway through a user's selected range. Verify actual commit text, suffix, cancellation, pronunciation variants and UTF-8/UTF-16 ownership. The native consumer continues to work offline from its encrypted projection while the desktop is stopped.

## Implementation order and deliverables

| Order | Priority | Work | Completion evidence |
| --- | --- | --- | --- |
| 1 | P0 | Freeze data/reuse/reset contract and research profile; document ADR/spec amendments required for shipping | Schema examples, lifecycle table, privacy scope and reviewable migration plan |
| 2 | P0 | Capability probes: confirmed lexical/pronunciation extraction, absent-term keyboard generation, per-session native ASR hints, signed bridge/key isolation | Actual pinned-engine/runtime outputs; unsupported paths explicitly failed |
| 3 | P0 | Domain terms, source contribution identity, bounded retrieval and legacy pair migration | Duplicate/conflict/reversal/delete tests; old ciphertext preserved on migration failure |
| 4 | P0 | Encrypted IMK journal, idempotent import, revisioned projection and restart/reset behavior | Crash/replay/old-epoch/key/tamper cases against real storage |
| 5 | P0 | Keyboard-to-ASR adapter, immutable session snapshots and capability reporting | Real audio with rewrite/rule stages disabled for isolation |
| 6 | P0 | Voice/manual-term-to-keyboard producer | Absent-base-word typing, selection/commit and standalone IMK observations |
| 7 | P0 | Freeze development winner; run held-out quality, fault, performance and native latency studies | Complete paired results, workload/latency/resource traces, provenance and archive verification |
| 8 | P1 | User-facing source/reset/disable controls and staged product adoption | Native UI scenarios and amended normative docs; independent review |
| 9 | P1 | Separate authorized temporary-context experiment | Relevant/irrelevant/conflicting context results; no persistent typing-history dependency |

Do not switch the default ASR checkpoint to make the bridge look effective. If a research model is needed, evaluate it as a separate model block and complete native conversion/parity and packaging gates before any adoption claim. Windows uses the same domain contract, but keyboard acceptance remains blocked on its real TSF implementation and Windows execution; macOS or Rosetta results cannot satisfy it.

## Evaluation: isolate each causal step

All variants use the same frozen memory, audio, checkpoint, precision, initial draft, endpoint policy and decoder settings within a model block. No LLM rewriting, new deterministic replacement or candidate ranker participates in the primary ASR comparison. Separately replay the unchanged downstream correction/rewrite settings to measure delivered-text interactions.

| Arm | Treatment | Purpose |
| --- | --- | --- |
| B0 | Shipped pipeline, no cross-mode bridge | Product reference; record its complete model-call count |
| B1 | Fixed selected local finalizer, empty hint context | Matched consumer/model baseline; if identical to B0, reuse the observation with explicit provenance |
| B2 | Same finalizer, frequency/recency top-K eligible terms | Doubao-like bounded user-word projection baseline |
| B3 | Same finalizer, retrieved compact destination-only terms | Main candidate; compare with B1 to isolate hints and with B2 to test selection |
| B4 | Same finalizer, unrelated terms matched in count/token budget | Context contamination control; not a deployment candidate |
| B5 | If extra re-decode is proposed: same extra original-audio pass with empty context | Matched extra-call control; compare hinted extra-pass policy against this and B0 |

B1–B4 have the same finalization schedule and number of model calls. When B3 retrieves nothing, its empty-context call is the B1 path and can be shared, explicitly recording actual call identity. Extra-pass policies must report routed-case overhead separately; overall p95 can conceal expensive routed calls. A research 1.7B checkpoint gets its own B1–B5 block, not a misleading direct attribution against a different production checkpoint.

Temporary context is a later matched study: vocabulary off/on × context off/relevant/unrelated/conflicting, with identical audio and fixed model budget. Context comes only from prior authorized events or independently collected task material, never the later transcript/reference. Add a memory-off two-pass control if current-audio-derived retrieval is used. Separate a term-list effect from semantic history and from call-count changes.

Keyboard comparisons: original engine order, existing context ranking, personal-term producer with fixed ranking, and producer plus existing context ranking. Include unseen base-dictionary words and ordinary homophones. Evaluate candidate generation recall before top-1/MRR; ranker-only comparisons must use identical candidate sets. Do not attribute newly introduced candidates to a ranker-only gain.

## Corpus and leakage controls

### Development smoke study

Start with at least 40 development terms across personal names, company/project names, English identifiers and mixed-language terms. Each has earlier teaching actions and three different later utterances; include at least 120 ordinary/near-homophone/unrelated controls plus silence/noise. Synthetic speech can debug the pipeline and expose failure modes, but cannot satisfy natural-user quality gates. Reuse exposed historical data only as labelled development/regression evidence.

### Held-out confirmation

Target a separate 24-person natural-speech study: 12 development and 12 held-out participants, with disjoint personal term sets and task templates. Each held-out participant enrolls 12 personal terms and later records three different utterances per term, yielding 432 primary recurrence utterances. Use separate teaching/later sessions. Counterbalance keyboard enrollment and voice-edit enrollment across terms, and report the two directions separately. Add an independent untouched-control set of at least 1,000 correct speech utterances and at least 300 silence/noise fixtures; report their actual source/speaker coverage.

These are proposed collection sizes, not available data or a power guarantee. Estimate precision/power from development effect sizes before freezing the final sample count. Natural recordings require explicit provenance/consent; private recordings stay private. Authored/TTS corpora must carry that label.

Each personal profile starts with an empty vault. Enrollment inputs are an explicitly separated prior partition: actual confirmed typing selections or actual earlier ASR plus the participant's correction. Later evaluation audio, references and acceptable answers never enter enrollment, retrieval or counters. Held-out participants may teach their own terms before the scored later session after the policy is frozen; this is personalization enrollment, not permission to learn from scored test outputs.

The current generic corpus protocol rejects test-derived memory. Extend it with explicit enrollment provenance and ordering checks or use a separate versioned study driver as in v3. Never relabel scored test outputs as `user_approved` to bypass isolation guards. Freeze terms/aliases, corpus bytes, retrieval parameters, budgets, selection criteria and source hashes before scored inference. Failed final-test cases become disclosed development data for the next fresh study; no retuning on this test.

Include these conditions:

- Rare term absent from the base keyboard dictionary; one confirmed use; repeated teaching; multiple legitimate pronunciations.
- Same term in new wording; homophones with different intended meanings; related but unlearned words; ordinary words sharing a name's pronunciation.
- Empty/unrelated/competing vocabulary; 0/100/2,000/20,000-entry research vault sizes with fixed retrieved budget. These are scalability probes, not accepted storage limits.
- Typo or mistaken teaching; undo; reversal; deletion/reset; expired/source-disabled records; duplicate event replay; desktop stopped then restarted.
- Chinese, English and mixed speech; accents, noise, short and long utterances; numbers, negation, questions and code identifiers.
- Cancel/focus change/secure or unknown field; off switch; old session completion; pending preedit and partial candidate selection.

## Measurements and decision gates

Report every stage with its own denominator: eligible acquisition events, terms actually learned, terms present after restart, relevant terms retrieved, correctly recognized terms, wrong-term insertions and actual host commits. The primary term-hit rate includes every planned later recurrence case, including missed learning and missed retrieval. Also report conditional recognition given correct retrieval to locate bottlenecks.

| Dimension | Required measurements |
| --- | --- |
| Acquisition | Complete-term precision/recall, pronunciation correctness, false promotion on non-corrections, source-specific counts |
| Storage/bridge | Durable import/reload rate, duplicate inflation, stale resurrection, epoch convergence, queue cap failures, delete visibility |
| Retrieval | Relevant-term recall@K, precision@K, ambiguity abstention, empty routing, wrong-term exposure and tokenizer budget |
| Speech | Term exact recall, baseline misses recovered, lost baseline hits, CER/WER/MER, correct-input corruption, arbitrary wrong entities and meaning errors |
| Keyboard | Candidate recall@N, top-1/MRR, selections/page actions per term, exact final commit, ordinary homophone displacement |
| Runtime | Acquisition/bridge/retrieval p50/p95/p99, routed extra-call cost, stop-to-review/commit-ready latency, first partial/preview stability, RTF, key-to-preedit and peak RSS |
| Integrity/privacy | Dropped samples, queue age, stale/duplicate commits, canary plaintext in disk/logs/backups, unexpected content traffic, denied-field events |

Use paired speaker/profile-cluster bootstrap intervals; group all a participant's personalized terms/sessions together. Repeated seeds and TTS voices are not independent human participants. Predeclare B3 versus B1 as the primary recognition contrast; secondary ablations are exploratory unless multiplicity is addressed. Report baseline-correct input harm as both a paired difference and a rate with a 95% upper confidence bound. Zero observed errors is not proof of zero population risk. Independent blinded review must inspect arbitrary entity and semantic changes, not only a detector limited to known vocabulary terms.

Provisional development go/no-go targets, to be calibrated on development and frozen before confirmation:

- At least +10 percentage points in full-loop term recall versus the matched empty-context consumer, with the paired 95% interval lower bound above zero on held-out natural recurrence.
- Correct-input corruption 95% upper bound at most 0.5%; ordinary-speech CER/WER/MER paired degradation upper bound at most 0.5 percentage points within each adequately sampled language stratum. An underpowered stratum is pending, not a pass.
- Zero new observed number/negation meaning changes, unrelated entity insertions and nonspeech emissions in the declared critical controls. Review every failure; report confidence bounds and sample counts rather than claiming zero risk.
- Native warm stop-to-review p95 increase at most 300 ms; routed-case extra compute p95 at most 500 ms; key-to-preedit p95 increase at most 2 ms; incremental native peak RSS at most 50 MiB. Measure cold cost separately. These are candidate budgets, not existing performance claims.
- Exact duplicate/restart/reset/undo/tamper/focus ownership invariants must all pass; no plaintext downgrade, raw typing-history storage, unauthorized context read or content network request.

The existing v3 routed latency would not satisfy these proposed runtime budgets. Investigate using the existing acoustic finalization slot and a native bounded-context consumer before adding another decode. A quality winner that misses latency or safety gates remains experimental; do not conceal its cost with a global percentile or weaken criteria after viewing test results.

## Performance test plan

Performance acceptance is a separate required result alongside quality and correctness. Run the real encrypted storage, authenticated bridge, candidate engine and supported native ASR consumer; a mocked runtime or unencrypted in-memory store cannot satisfy their performance gates. Component benchmarks isolate bottlenecks, while actual installed-host measurements establish user-visible latency. No performance test in this section has been executed yet.

### Fixed conditions and instrumentation

Pin CPU/architecture, RAM, OS build, storage type/free space, power mode, thread counts, model/tokenizer/runtime builds, application signatures and Git revision/dirty state. Record thermal state and background load. Compare bridge off/on on the same machine with seeded interleaved block order and identical workloads. Test the declared minimum supported hardware as well as the development machine; report platform strata separately. A run only on the development machine leaves minimum-hardware acceptance pending.

Distinguish process-cold, model-cold and warm states. Record file-cache state as observed/unknown; restarting a process does not prove a cold OS page cache. Reset residency only for the explicitly cold stratum. Keep at least two separate warmup fixtures per inference block, excluding their measurements but retaining their outputs. No test-only forced GC, plaintext storage or disabled durability may improve the measured treatment.

Use a monotonic clock per process and record event IDs to correlate stages. Cross-process end-to-end latency must use a verified shared host clock or a sender-observed round trip; never subtract uncalibrated process-relative timestamps. Capture these boundaries:

- Native key receipt → engine/preedit ready → actual native preview/render observation.
- Confirmed host insertion → journal enqueue → durable journal write → durable desktop import/acknowledgement → native consumer activation of the new projection.
- Snapshot request → retrieval complete → tokenization complete → finalizer dispatch → actual decode completion → review ready.
- Delete/reset/disable request → invalidation → durable change → consumer revision acknowledgement, with stale-output rejection traced separately.

Measure service time and queue waiting separately. Trace actual render observations, not a copy of engine events. Measure instrumentation overhead against a minimal-trace control. Only synthetic/authorized research fixtures may carry word text in private traces; production diagnostics remain content-free.

### Workload matrix

| Test | Fixed workload and sweep | Required outputs |
| --- | --- | --- |
| Retrieval/index scaling | 0/100/2,000/20,000 terms; top-K 2/4/8; budgets 32/64/128 tokens; exact/phonetic/absent/ambiguous queries; repeated and unique query streams | Build/rebuild time, query p50/p95/p99, allocations, retained heap, actual tokens and retrieval correctness |
| Encrypted persistence | Single-term insert/update/delete; batches of 10/100/1,000; repeated-word coalescing; snapshots at each supported size | Enqueue versus durable completion, encryption/fsync/rename time, physical bytes written, file growth, write amplification and error rate |
| Bridge transport | Delta batches 1/10/100; full projection at each supported size; desktop stopped then restarted; capped journal backlog | Delivery/import/ack/activation latency, bytes per event, backlog drain rate, duplicates, maximum queue age and explicit rejection counts |
| Native keyboard | Identical Pinyin/key sequences, paging, partial selections and commits with personal-term producer off/on; baseline/10/30 key events per second using scheduled replay | Key-to-preedit and rendered candidate latency, missed events, event-loop stalls, exact commit results and candidate identities |
| ASR/context budget | Same 2/10/30/120-second audio at empty/selected/max context; B1–B4; cold/warm; standalone and mixed workload | First partial, stop-to-review, tokenization/prefill/decode time, RTF, actual call count, routed overhead, drops and quality |
| Concurrent reads and writes | Component snapshot/retrieval readers 1/2/4/8 with one real serialized writer; write arrival 1/10/100 events per second | Latency versus offered load, successful durable throughput, contention, queue depth/age, stale revision rejection and failures |
| Realistic contention | One supported voice session plus keyboard event replay, background journal import/projection rebuild and bounded writes | User-visible typing/ASR regression, capture deadlines, CPU/GPU/RSS, disk I/O and lock/event-loop stalls |
| Lifecycle operations | Cold start, reconnect, full reload, migration, delete/reset/disable with pending writes and inference | Ready/usable time, durable convergence, cancellation latency, stale deliveries and restart recovery |
| Endurance | A 30-minute repeated mixed-workload run, followed by an 8-hour soak for the frozen candidate; alternate active, idle and reconnect phases | Resource/queue/file time series, retained-memory slope, thread/handle counts, throughput drift and all failures |

The 20,000-term case is an experimental stress point, not a promised product capacity. If the final schema has a smaller declared limit, test the limit plus explicit over-limit rejection instead of pretending the stress size is supported. Do not expand to every matrix combination: first perform one-factor component sweeps, then test the development-selected typical and worst supported combinations. Freeze those confirmation cells before the final performance run.

Concurrency above one is a component stress probe. It does not imply that the product permits simultaneous dictation sessions or that a serialized native recognizer can serve concurrent inference. Keep one writer and the production session-ownership constraints; label unsupported multi-inference configurations explicitly.

For non-inference components, target at least 10,000 completed operations per steady-state cell, across three independent process runs. Retain timeouts/rejections as offered operations rather than silently retrying them into successes. Use scheduled arrivals independent of completion to expose queueing; record scheduled start, actual start and finish to avoid hiding overload with a completion-driven load generator. Mixed native replay also records scheduler lateness. Any test cutoff preserves its incomplete status.

For expensive inference, use all paired corpus observations and three interleaved performance blocks, with cold/warm and routed/non-routed cases separate. Report per-cell counts; fewer than 1,000 observations makes p99 exploratory rather than a release-tail claim. Repetitions estimate runtime variability, not independent speaker accuracy. Workload phases and durations here define test execution windows, not delivery-time estimates.

### Provisional performance gates

Freeze budgets after development calibration and before confirmation. The existing user-visible budgets remain mandatory; component targets help diagnose their failure, not replace them with a faster isolated benchmark.

| Area | Proposed supported-workload gate |
| --- | --- |
| Retrieval plus actual tokenizer | Warm p95 ≤ 5 ms and p99 ≤ 10 ms at the selected maximum supported term count and hint budget |
| Acquisition persistence | Off-key-path enqueue p99 ≤ 1 ms; single-event durable write p95 ≤ 50 ms / p99 ≤ 100 ms under normal local storage conditions |
| Live cross-mode propagation | Confirmed insertion to consumer activation p95 ≤ 250 ms with both processes available; deferred backlog recovery reported separately |
| Keyboard | Bridge-on key-to-preedit p95 increase ≤ 2 ms versus paired bridge-off baseline; no lost keys, duplicate commits or synchronous disk/IPC wait on the key path |
| Voice | Warm stop-to-review p95 increase ≤ 300 ms; routed extra compute p95 ≤ 500 ms; no additional audio drops; report every over-deadline request |
| Process memory | Sum of per-process native RSS peaks increases ≤ 50 MiB for the bridge/index/projection with unchanged models; report each process and this conservative sum, not a misleading peak from only one process |
| Steady-state resources | After the warmup/cache-fill phase, 8-hour retained RSS growth ≤ 10 MiB and fitted slope ≤ 1 MiB/hour; no unbounded queue/file/thread/handle growth |
| Idle overhead | Bridge-on incremental CPU consumption ≤ 0.01 CPU-seconds per wall-second, where 1.0 equals one fully occupied logical core; no periodic unchanged snapshot rewrites |
| Capacity and overload | At normal declared input rates, no rejected/lost accepted events and no growing backlog; at stress saturation, bounded queues and explicit errors, with recovery after offered load returns to normal |

Separate ASR/model accelerator memory from host RSS and retain GPU/Metal resource measurements when available. The sum of RSS peaks is conservative and can count shared pages more than once; it is not unique physical memory. Measure CPU time per successful operation, bytes written per unique learned term and projection rebuild count to catch work amplification even when latency passes. Resource peaks alone do not prove absence of a leak; retain baseline-subtracted time series and active/idle phase comparisons without forcing collection.

Fault-injected disk stalls, full storage and unavailable keys are correctness/recovery strata, not expected to meet normal-operation durability latency. They must still leave typing/capture responsive and report explicit acquisition failures. Report saturation, queue-full outcomes, incomplete operations and thermal changes; never merge failures into a successful-throughput denominator or discard slow samples as outliers without a predeclared instrumentation-fault rule.

### Performance report and reproduction

Deliver workload configuration, raw operation/event timing, per-process resource time series, histograms, p50/p95/p99 with counts, throughput/offered-load curves, failure/rejection rates, queue-age curves and paired bridge-on/off deltas. Include a gate table with `pass`, `fail`, `pending` or `unsupported` per OS/hardware/model and supported workload. Hardware unavailable, absent native render traces or insufficient tail samples cannot produce a general passing verdict.

Latency percentiles follow the repository's type-7 definition. Bootstrap whole run blocks for component runtime comparisons and retain speaker/profile clustering for audio-quality comparisons; do not treat correlated operations as independent users. Explain any quality/latency tradeoff and archive slow or failed attempts alongside successful ones.

Add the performance mode to the same versioned bridge-study driver, portable Node entry point and Make wrapper. Store replayable workload manifests, timing/resource schemas and report-generation sources in the proposed archive family below. A performance report must regenerate from retained raw events without rerunning the model. These commands are implementation deliverables; they are not claimed to exist yet.

## Native scenarios and evidence packaging

On actual macOS, test an owned AppKit editor and owned supported external-editor fixture with the installed signed IMK source. Start empty, select an absent-base personal word, restart desktop/host as appropriate, dictate different wording, remove the term and repeat. Reverse the direction using a manual/voice-learned spelling with declared pronunciation, then type it. Record actual candidate text, selection identity, host insertion and reset outcomes. Engine-only harnesses are prerequisites, not evidence of installed IMK dispatch. Windows and physical Intel claims require their own hosts.

Inject crash-before-write, crash-after-write-before-ack, reordered messages, stale projection, bad tag/key loss, corrupt migration, full journal and app absence. Repeat disable/delete/reset during an in-flight recognition and during a keyboard composition. Any late response must be rejected or recomputed against the new authorized revision before insertion. Tests use owned namespaces and restore their state.

Follow [008 Evaluation](../../specs/008-evaluation.md) and [tooling](../../tools/evaluation/README.md). Use the exact root pnpm version and frozen lockfile, portable Node entry points and Make wrappers. Extend the existing persistent driver rather than inventing another shell-only report path. New bridge-study commands and treatment fields are deliverables, not existing commands to run today.

Proposed archive family `keyboard-voice-lexicon-v1`: reusable authorized fixtures/provenance under `benchmarks/corpora/`, reproduction sources under `benchmarks/reproduction/`, raw observations/warmups/failures under `benchmarks/runs/`, and verified summaries under `benchmarks/results/`. Weights, environments, keys, personal recordings and unfinished execution remain in ignored workspaces. Research outputs using synthetic terms may retain term/projection details; product diagnostics remain content-free.

Every scored result binds corpus/audio, enrollment event chain, source profile/epoch/revision, selected term IDs/token count, model/tokenizer/runtime hashes, initial draft, actual model-call IDs/count, all stage outputs, monotonic timing, effective switches, hardware/power and Git dirty state. Reference annotations stay evaluator-only. Random IDs/nonces may differ on rerun; verify semantic state and each run's recorded bytes rather than expecting identical ciphertext.

Completion requires corpus/manifest checks, no leakage, complete paired quality and performance execution, native supported-workload and endurance gates, independent review, original-byte/hash archive verification and model-free report replay from a clean checkout. Missing capabilities and interrupted attempts retain failed/incomplete status. The adoption report names which direction and which OS/model/hardware passed; it never reports a general cross-platform or competitor-parity success from an isolated positive result.
