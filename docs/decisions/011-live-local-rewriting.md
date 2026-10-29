# ADR 011: Live local rewriting with Qwen while dictating

The current runtime, templates and delivery policy are superseded by [ADR 018](018-staged-gguf-rewriting.md); the live scheduling and insertion ownership requirements remain.

## Status

Accepted (product decision). Supersedes the review-before-insert rule of [ADR 008](008-contextual-rewrite-evaluation.md) for the final rewrite; ADR 008's evaluation boundary otherwise stands. The mid-speech text-model preview is superseded by [ADR 023](023-final-only-polishing.md). Segment closure and the final rewrite remain.

## Context

Typeless, Doubao and Chatterfly show text being corrected while the user is still speaking, and insert a polished result when they stop ([competitor research](../research/competitor-rewrite.md)). Arion's earlier Qwen diagnostics ([report](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/qwen-pipeline-diagnostic-v1/README.md`)) measured 0.6B–4B checkpoints in a Python worker that recomputed the whole prompt per request: 4B took 6–15 s, too slow for interactive use, and smaller checkpoints rarely applied spoken self-corrections.

Reusing the KV state of a fixed instruction-and-demonstration prefix, and extending it as the transcript grows, removes most of that cost. Measured on Apple M4 CPU (see the [live pipeline report](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/live-rewrite-v1/README.md`)), a final Qwen3 1.7B rewrite takes 0.5 s at the median after stop, so a small local model can preview edits during speech and deliver one validated rewrite after stop.

## Decision

- **Pipeline.** Streaming recognition stays the preview source. When the recognizer closes a segment at a pause, the coordinator cuts the segment's audio at the quietest point of that pause (optionally re-decoding it per segment, off by default) and asks the local LLM to rewrite the stable text so far. The display is the validated preview of the stable text followed by the open segment. At stop, previews are cancelled, the remaining audio is finalized and one final rewrite is validated and inserted. Before model organization, `domain.CleanSpeech` applies versioned local deletion spans to the rule-corrected text; this rule producer is independent of the model prompt. Previews never commit.
- **Model and prompt.** Qwen3 1.7B INT4 (onnx-community CPU export) is the default rewriter, chosen over 0.6B and 4B by the held-out benchmark. Prompt `live-v2` puts instructions and 18 demonstrations in a fixed prefix; dictated text is data in the last user turn, with chat control markers rejected. Decoding is greedy and non-thinking, with output budgets scaled to the input.
- **Runtime.** A separate `arion-llm` worker process loads ONNX Runtime GenAI 0.17.0 and ONNX Runtime 1.30.0 through purego (no cgo). It never shares an address space with sherpa's ONNX Runtime, so a crash or library conflict cannot take dictation down. The worker and runtime libraries ship inside the app and are signed with it; only the model weights are downloaded, on the user's explicit choice, as SHA-256-pinned files. Where huggingface.co is unreachable the download continues from hf-mirror.com without asking; the pinned digests make the source irrelevant to integrity, and the source that worked is reused for the remaining files.
- **Delivery.** A proposal that passes validation is inserted automatically. Validation rejects truncation, chat markup and echoes, implausible length changes, new numbers, unsupported new content, dropped negations, a dropped final value after a self-correction, and large deletions without a repair cue. A rejected, timed-out or unavailable organization retains the completed local speech cleanup, and its failure outcome is recorded with the result. The cleanup contract and evaluation limits are defined by LRW-09 in spec 009. The recognized original stays available to copy from the menu bar panel.
- **Scope.** Off by default; turned on in the onboarding model step or Settings › Model, where recognition and polishing are configured together. The interface names neither engines nor model families: recognition options carry plain names ("Accurate", "Fastest"), and polishing is described by what it does. macOS on Apple silicon only: no GenAI runtime is pinned for Intel Macs or Windows yet, and the setting reports that explicitly. No context capture; approved vocabulary is passed as data.

## Consequences

- Dictation with polishing runs fully offline after the one-time model download (about 1.4 GB). Audio and text never leave the device.
- The app bundle grows by the GenAI and ONNX Runtime libraries (about 56 MB) plus the worker.
- Rewriting can still change meaning in ways the validator cannot detect. The benchmark uses exact-string proxies on developer-authored synthetic speech; independent blind meaning, style and hallucination review (RWT-A08), Windows and native IME/E2E lifecycle evidence remain open release gates in [009](../../specs/009-contextual-rewrite.md).
- Per-segment finalization raised recognition error at segment boundaries in the benchmark, so the shipped pipeline previews streaming segments and the finalizer decodes the whole utterance once after stop.

## Alternatives considered

- **Review before insert** (ADR 008): safer for meaning, but it removes the hands-free flow that defines the competitor experience. The validator and the always-available original are the mitigation.
- **4B**: one more held-out pass at twice the latency (stop-to-final p95 2.0 s) and more memory; kept as a benchmark variant.
- **0.6B**: fastest, but three fewer held-out passes; its harmful edits (dropped clauses and values) were blocked only by the validator.
- **In-process GenAI via cgo**: lower IPC cost, but two ONNX Runtime builds in one process and no crash isolation.
- **Qwen3-ASR as finalizer**: evaluated as a bundle option. It did not beat the X-ASR finalizer on these fixtures and costs several hundred milliseconds more per decode.

## References

- [009 Contextual rewriting](../../specs/009-contextual-rewrite.md), [002 Speech pipeline](../../specs/002-speech-pipeline.md).
- [Live pipeline benchmark report](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/live-rewrite-v1/README.md`) and [methodology](../research/live-rewrite-benchmark.md).
- [Qwen pipeline diagnostics](../research/qwen-pipeline-diagnostics.md), [paper strategies](../research/qwen-paper-strategies.md).
