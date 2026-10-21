# Native Qwen3-ASR 1.7B and accessibility context

**Keep Accessibility window reading; retain Qwen3-ASR 1.7B as an evaluated optional finalizer.** This CPU bundle improves Mandarin and English error rates on the existing development set, but worsens mixed technical speech and exceeds the automatic latency gate. Automatic selection is `xasr-160-xasr-final`; the 1.7B bundle remains available as `xasr-160-qwen3asr-17-final`. No latency threshold or language weighting was relaxed.

## Recognition: completed fresh comparison

Apple M4, macOS arm64, 32 GiB RAM; sherpa-onnx 1.13.8 / CPU, four threads. Both arms retain the same X-ASR 160 ms streaming preview, 20 ms realtime replay and 800 ms endpoint setting. Only the acoustic finalizer differs. Memory, ranking and correction are off. One seed, 118 exposed development clips (48 Mandarin, 40 English, 30 synthetic mixed-language clips), plus two separate warmups per arm. References never reach the recognizer.

| Metric | X-ASR finalizer | Qwen3-ASR 1.7B INT8 |
| --- | ---: | ---: |
| Mandarin CER | 45 / 635 = **7.09%** | 40 / 635 = **6.30%** |
| English WER | 23 / 620 = **3.71%** | 16 / 620 = **2.58%** |
| Mixed MER | 93 / 368 = **25.27%** | 104 / 368 = **28.26%** |
| Explicit stop → review p50 | 90 ms | 747 ms |
| Explicit stop → review p95 | 136 ms | 1,820 ms |
| Corpus real-time factor | 0.171 | 0.351 |

Across the 118 pairs, normalized error count improves in 19, worsens in 18 and is unchanged in 81. Equal error counts do not imply equal words or meanings. The unweighted mean of the three language rates is slightly worse for this Qwen bundle, and its 1,820 ms stop-to-review p95 exceeds the existing 1,000 ms automatic-selection budget. These are development observations, not population estimates or a new held-out/release result.

[Raw observations, warmups and hardware/runtime/artifact identities](../../runs/qwen17-accessibility-v1/qwen17-asr-dev-v1/), [verified standard report](../qwen17-asr-dev-v1/summary.json), [per-case paired analysis](../../runs/qwen17-accessibility-v1/asr-paired-analysis.json), [actual selection result](../../runs/qwen17-accessibility-v1/selection.json).

RSS is a cumulative process high-water mark, not per-model allocation: p95 is about 1.17 GB in the baseline block and 5.39 GB after the Qwen block has loaded. The process previously hosted the baseline; these values must not be used as isolated steady-state memory or minimum-RAM claims. Timing is from a shared development machine, without a thermal/energy isolation study; early verification work is disclosed in the retained limitations and logs. No slow samples were removed. Engine partials and stop-to-review exclude native rendering, microphone capture and editor insertion. Qwen remains a nonstreaming finalizer in this adapter.

## Actual window collection

Twelve owned AppKit window fixtures, three paired reads each; both Accessibility and historical ScreenCaptureKit/Vision return successfully for all 36 pairs.

| Metric | Accessibility | Capture + accurate Vision OCR |
| --- | ---: | ---: |
| Native collection p50 | **0.90 ms** | 140.37 ms |
| Native collection p95 | **5.04 ms** | 156.70 ms |
| Complete fixture text present | **36 / 36** | 27 / 36 |
| Final rewrite review with fixed source text | 12 / 12 | 12 / 12 |

The text-presence check is a whitespace-normalized exact substring proxy, not semantic accuracy. All 12 final outputs are identical between these fixed-source arms. Faster/cleaner collection therefore establishes no additional correction gain on this set. Password and hidden canaries are absent. A subsequent real native probe confirms visible-range extraction excludes offscreen document text, rejects stale focus and discards partial output on the node limit.

Product `Targets.ReadContext` uses system Accessibility, without screenshot/OCR fallback or a Screen Recording request. Editable fields use visible ranges; secure/unknown targets and denied applications are not read. Traversal is bounded by deadline, nodes, depth and text size. The Go adapter retains its own AX handle so a slow read does not hold the target registry lock. Settings now asks for Accessibility when permission is missing. Historical OCR APIs remain only as explicit benchmark controls. Windows uses UI Automation; it is cross-compiled, not run on Windows. Third-party app/canvas/provider coverage is unmeasured.

[Paired reads and hash-bound reviews](../../runs/qwen17-accessibility-v1/window/), [combined context summary](../../runs/qwen17-accessibility-v1/combined-context-summary.json).

## Combined audio/context/final-rewrite diagnostic

The 12 authored window scenarios are also synthesized with Tingting, recognized by each real native ASR arm and then passed to the same installed Qwen3.5-2B rewrite worker. Old arm uses recorded OCR; new arm uses recorded Accessibility text. Both apply the existing rule correction and cleanup. Assistant inspection requires correct names, case-sensitive code identifiers and preserved constraints/uncertainty; harmless punctuation changes pass.

- Old pipeline: **10 / 12** pass.
- New pipeline: **8 / 12** pass.
- Gain: `loadcache` becomes the correct `loadCache`.
- Regressions: `getProfile` becomes `GetProfile`; the member 程语楠 remains 成语难; `getCash` remains `getcash`.
- Both pipelines miss the intended surname 林 in the recipient case and deliver 李 instead.

[Every ASR output, final rewrite and explicit review](../../runs/qwen17-accessibility-v1/integration/), [original AIFF and hash-bound WAV fixtures](../../corpora/asr-window-v1/README.md). This is assistant inspection of authored synthetic material, not independent human ratings. It replays recorded window text into final-only requests; earlier stable-recognition evidence, live previews, native insertion and simultaneous ASR/rewrite contention are not replayed. The two-change comparison cannot attribute each difference to a single component.

## Retained failure and protocol correction

The [original complete comparison](../../runs/qwen17-accessibility-v1/pre-envelope/summary.md) exposed `language Chinese<asr_text>` and English equivalents in some delivered text. The actual SDK token probe returns an initial `" language"` token. Pinned sherpa code accepts only an exact initial `"language "` prefix, so the leading whitespace defeats its native metadata removal. The adapter now follows the official metadata/text contract at the ASR boundary, while preserving text-only results and literal delimiters within the body. No model, prompt, seed or decoding parameter was tuned.

Three real previous-leak cases verify the fix. The fresh full run has zero delimiter leaks; all 118 baseline texts repeat exactly, and all 118 Qwen transcription bodies match the original run after metadata separation. Both complete runs, their four warmups, the initial failed artifact-attestation run and provision failures are retained. Failed records are not converted into passing results.

[Diagnosis and source snapshots](../../reproduction/qwen17-accessibility-v1/README.md), [failed hello](../../runs/qwen17-accessibility-v1/failed-hello/), [check logs](../../runs/qwen17-accessibility-v1/checks/).

## Reproduction and scope

Use the [documented native commands](../../reproduction/qwen17-accessibility-v1/README.md). `pnpm qwen17:context:report` reproduces collection and optional combined-stage review from archived bytes; `pnpm qwen17:context:test` checks incomplete reads, altered outputs, leaked canaries and invented ASR inputs. `pnpm qwen17:verify` checks the [archive inventory](archive.json) without model weights. The ordinary ASR reporter replays the completed native run with `pnpm bench:report --results benchmarks/runs/qwen17-accessibility-v1/qwen17-asr-dev-v1`.

The six community ONNX/tokenizer payloads total 2,404,222,421 bytes and are individually revision/size/SHA-256 pinned. Source-checkpoint floating-point parity is unverified. Earlier [MLX 1.7B memory diagnostics](../user-modify-asr-v1/README.md) use another runtime and corpus; they do not measure this Go/ONNX CPU bundle. No Windows, physical Intel, minimum-hardware, air-gapped network audit, cold-run or full microphone-to-editor acceptance is claimed. Private weights, native binaries and runtime environments remain outside the archive.
