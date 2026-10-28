# Production recognition-stage hotwords

The new native hotword path changes actual Qwen3-ASR recognition through `asr.Offline.Finalize`. In this small controlled development probe, exact canonical spelling recall rises from **4/9 to 6/9**. Codex and TypeScript improve; Arion, MyGo and API remain incorrect. This is evidence that context reaches audio decoding, with limited lexical benefit on exposed fixtures. It is not a held-out accuracy or general product-quality claim.

| Metric | Empty context | Canonical context |
| --- | ---: | ---: |
| Relevant canonical spellings, exact | 4/9 | 6/9 |
| Mixed MER, 8 speech cases | 15/76 = 19.74% | 7/76 = 9.21% |
| Mandarin CER, 3 speech cases | 2/32 = 6.25% | 2/32 = 6.25% |
| English WER, 4 speech cases | 1/32 = 3.13% | 1/32 = 3.13% |
| Unrelated-term insertions, 7 negative cases | 0 | 0 |
| Silence emissions, 1 generated-silence case | 0 | 0 |
| Preserved numeric concepts | 1/1 | 1/1 |
| Offline finalization p50 / p95 | 395 / 520 ms | 409 / 553 ms |

Sixteen archived development inputs contain 13 authored synthetic TTS fixtures, two original public natural-read recordings and one generated silence. The nine relevant contexts are explicit fixture-specific controlled spellings; the seven negatives use irrelevant spellings. Original audio, checkpoint, PCM reader, decoding and output budget are the same in both arms. Paired order alternates; there are no independent repeats or inferential confidence claims. Timing excludes model load and audio read, and is complete-utterance offline compute rather than live stop latency. Model load was 1,063 ms. The verified host reports Mac16,12, 10 CPU cores, 32 GB and macOS 26.6.2; its hardware tool reported an unknown chip name. AC power was connected and low-power mode was off. Ordinary background activity was not controlled.

All raw changed outputs are retained:

| Case | Empty context | Canonical context |
| --- | --- | --- |
| Redis | 缓存服务使用Redis。 | 缓存服务使用 Redis。 |
| Codex | 用寇代克斯检查这个修改。 | 用Codex检查这个修改。 |
| TypeScript | TPS2的例行检查没有通过。 | TypeScript的类型检查没有通过。 |
| API / number | 我们的 R T 响应时间大概是两百毫秒。 | 我们的 RT 响应时间大概是两百毫秒。 |

The API spacing change reduces the shared mixed-unit error count but does **not** recover API. Redis changes only spacing. Aggregate MER therefore overstates the number of recovered canonical terms. No scored case regresses; all negative outputs are byte-identical across arms. The public Mandarin annotation still disagrees with both outputs, and no new independent transcript review was performed.

The main run contains 38 actual calls: 32 measured, two excluded training warmups and four Arion context/empty observations. Those Arion outputs are unchanged regardless of context, so their state-isolation sensitivity is limited. A separate post-observation Codex probe adds four calls in one resident worker. `Codex → empty → Codex → empty` produces `用Codex检查这个修改。 → 用寇代克斯检查这个修改。` twice. Both empty results match the paired empty baseline and both contextual results match the paired context result. This is an observed request-isolation check with a visible treatment effect; it does not prove arbitrary hidden-state equivalence.

The model is the installed catalog Qwen3-ASR 1.7B MLX 8-bit checkpoint at `a8379a2e2f9e313c9292cdf1af4055ab56d50d55`; model payloads, new Swift worker, production-port driver and exact source snapshot are SHA-256 pinned in run metadata. The production context uses canonical terms joined with spaces, with no correction aliases or transcript replacement. No new model was downloaded. BF16 parity, unseen speakers/terms, natural recurring user names, Windows execution, live preview latency and real microphone delivery remain unmeasured here. The separate native smoke verifies prompt capability, explicit oversized-context failure, and direct worker context reset.

The original sandboxed attempt failed before inference with a native model-load error; the same binary/model succeeded with sandbox escalation. A GPU sandbox cause is inferred. Its empty raw stream and failure metadata remain [archived](../../runs/asr-hotwords-v1/failed-sandbox-load/run.json). A default Go-cache build denial also remains recorded. Successful GPU-enabled runs use the same worker/model; no fallback was introduced.

[Replayable summary](summary.json), [raw paired outputs](../../runs/asr-hotwords-v1/paired-production/raw.jsonl), [sensitive reset outputs](../../runs/asr-hotwords-v1/sensitive-isolation/raw.jsonl), [protocol and reproduction](../../reproduction/asr-hotwords-v1/README.md). Run `pnpm asr:hotwords:verify` and `pnpm asr:hotwords:report --out <new-json-file>` to verify/replay without private models.

After these measurements, production extracted the existing ASR vocabulary eligibility predicate into `ASRHotwordEligible` for reuse by the UI badge. The measured driver supplies its controlled canonical list directly to `Offline.Finalize`; it does not call vocabulary selection. `ASRContext` and the measured decoding path are unchanged by that extraction. The retained pre-run source snapshot and original observations remain unchanged; this note does not claim an inference rerun against the later source tree.
