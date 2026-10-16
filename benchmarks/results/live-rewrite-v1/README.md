# Live rewrite pipeline results (v1)

These are real local CPU runs of the production code paths on Apple M4 (32 GiB, macOS arm64): the `arion-llm` worker (ONNX Runtime GenAI 0.17.0 / ONNX Runtime 1.30.0, prompt `live-v2`, greedy, 4 threads) and the production dictation coordinator replaying speech in real time. [The content-free summary](summary.json) holds every number below together with the hash of each corpus, fixture manifest and archived result file. The [methodology](../../../docs/research/live-rewrite-benchmark.md) defines the corpora and metrics.

**Limits.** One developer authored every case. Speech is macOS TTS with inserted 1.2 s pauses. Scores are exact-string proxies, not semantic accuracy, and there are no human ratings. The host was not isolated. Native preview rendering, IME insertion and Windows were not measured. The held-out set was written and hashed before prompt tuning; the development set was used to tune the prompt.

## Text stage (live schedule, then final rewrite)

| Model | Dev passes / 24 (source 8) | Held-out passes / 30 (source 12) | Worsened | Held-out final p50 / p95 | Live previews accepted |
| --- | --- | --- | --- | --- | --- |
| Qwen3 0.6B | 15 | 24 | 0 | 0.18 / 0.36 s | 31 / 45 |
| **Qwen3 1.7B** | 17 | **27** | 0 | 0.50 / 1.67 s | 43 / 45 |
| Qwen3 4B | 20 | 28 | 0 | 0.89 / 2.15 s | 40 / 45 |

"Passes" counts delivered text: the validated proposal, or the source where validation rejected it. Validation rejected every observed answer to a spoken instruction (translation, a poem) and 0.6B's dropped clauses and values, so nothing worsened. Four development cases need field context or prose memory, which the product does not supply. Compared with the [earlier diagnostic worker](../qwen-pipeline-diagnostic-v1/README.md) (1.7B 11/24 at 2.0 s, 4B 17/24 at 6.6 s p95), prefix KV reuse and the demonstration prompt make the 1.7B model both faster and more accurate.

## Audio pipeline (held-out, 27 fixtures, 21 multi-segment)

| Pipeline | CER zh / WER en / MER mixed | Passes (source 12) | Worsened | Stop → final p50 / p95 | Preview before stop |
| --- | --- | --- | --- | --- | --- |
| X-ASR 160 + X-ASR finalizer, no LLM (previous product) | 0.3 / 1.1 / 10.9% | 12 | 0 | 0.13 / 0.21 s | – |
| X-ASR 160 + Qwen3-ASR finalizer | 0.3 / 4.3 / 10.9% | 12 | 0 | 0.77 / 1.67 s | – |
| X-ASR 160 + Qwen3-ASR per segment | 1.7 / 2.1 / 8.7% | 12 | 0 | 0.37 / 1.49 s | – |
| Qwen3-ASR per segment + 1.7B | 1.8 / 2.1 / 8.7% | 21 of 26¹ | 0 | 1.55 / 3.97 s | 17 / 20 |
| X-ASR per segment + 1.7B | 1.0 / 3.2 / 19.6% | 21 | 0 | 0.48 / 0.99 s | 20 / 21 |
| X-ASR finalizer + 0.6B | 0.3 / 1.1 / 10.9% | 24 | 0 | 0.33 / 0.80 s | 18 / 21 |
| **X-ASR finalizer + 1.7B (shipped)** | 0.3 / 1.1 / 10.9% | **25** | **0** | **0.50 / 1.25 s** | **19 / 21** |
| X-ASR finalizer + 4B | 0.3 / 1.1 / 10.9% | 26 | 0 | 1.00 / 2.00 s | 20 / 21 |

¹ One fixture failed with an explicit `AUDIO_DISCONTINUITY`: three ONNX Runtime thread pools (streaming ASR, Qwen3-ASR, LLM) competed for CPU and the capture queue overflowed. The failure was reported, not concealed (SPH-02).

The development audio set (22 fixtures, source 6) went from 6 to 11 passes with 1.7B (0 worsened, 0.61 / 1.11 s, previews before stop in 16 / 17). Its English WER of 36% mostly reflects references that write digits and a URL ("2 pm", "120 dollars", "https://example.com/v2") which the recognizer returns as spoken words.

## Decision

- **Default rewriter: Qwen3 1.7B.** 0.6B is fastest but delivers three fewer held-out passes and needs the validator to block harmful edits; 4B adds one pass for twice the latency and memory.
- **ASR: keep the X-ASR whole-utterance finalizer.** Per-segment finalization lowers stop latency slightly, but raises recognition error at segment boundaries (and with Qwen3-ASR, contends for CPU). Previews therefore rewrite streaming segments, and the finalizer decodes the whole utterance once after stop. Qwen3-ASR stays a catalog bundle option; it did not beat X-ASR on these fixtures.
- **LRW-A05 gate: met.** Text stage: +15 held-out passes over source, 0 worsened. Shipped audio pipeline: 0 worsened, previews before stop in 90% of multi-segment fixtures, stop-to-final p95 1.25 s.
- KV reuse parity (`TestReuseParity`) passed on the 1.7B model: previews, other sessions and rewinds leave the greedy final answer unchanged.

Open gates: blind independent review (LRW-A06), natural speech, rendered preview and IME timing, Windows, power/thermal and a same-fixture Typeless/Doubao comparison.
