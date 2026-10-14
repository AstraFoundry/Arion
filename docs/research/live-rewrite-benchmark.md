# Live rewrite pipeline benchmark

## What it measures

The live pipeline ([ADR 011](../decisions/011-live-local-rewriting.md), [009](../../specs/009-contextual-rewrite.md)) previews LLM rewrites of stable text while the user speaks and inserts one validated rewrite after stop. Two benchmarks measure it with the production code:

- **Text stage** (`arion-rewrite-bench`): rewrites fixed text through the packaged `arion-llm` worker. Each case replays the live schedule: one live request per closed speech segment (cumulative stable text), then the final request on the same session, so prefix reuse and incremental prefill are exercised as in the product. It records proposals, validation verdicts, delivered text and engine timings.
- **Audio pipeline** (`arion-live-bench`): replays pause-segmented speech in real time through the production coordinator, using the catalog bundle's streaming recognizer and finalizer, the live worker and the final rewrite. It records every displayed draft (marking validated previews) and the final review on one monotonic clock.

Scoring (`tools/evaluation/live`) applies the exact-string proxies of the [Qwen diagnostics](qwen-pipeline-diagnostics.md): required, forbidden and protected strings, and verbatim identity. Audio runs also report recognizer error against the spoken script per language (CER for Mandarin, WER for English, MER for mixed text), stop-to-final latency and whether a validated preview appeared before the speaker stopped. `worsened` counts cases where the source text passed and the delivered text does not; `improved` is the reverse.

## Corpora and splits

- **Development:** the 24-case [diagnostic corpus](qwen-pipeline-diagnostics.md), used for prompt iteration (`live-v1` → `live-v2`).
- **Held-out:** 30 new developer-authored cases with pause scripts, written and hashed before any prompt tuning. They are synthetic challenges in the same categories (self-corrections across segments, fillers, repetition, negation, hedging, copy controls, email/list scenes, approved terms, instructions spoken as dictation, code/URLs, verbatim). The prompt was frozen before the held-out run; none of its demonstrations reuse held-out content.
- **Speech fixtures** (`build-live-audio.py`): each pause-separated part is synthesized with macOS `say` (Tingting for Mandarin and mixed text, Samantha for English) and joined with 1.2 s of silence, longer than the 0.8 s endpoint, so segments close as they would when a speaker pauses. Verbatim and unspeakable cases (code, URLs) are text-only.

Fixtures, rows and weights stay in `.workspace/`; only content-free aggregates are published.

## Scene selection

[Scene selection](../../specs/009-contextual-rewrite.md#scene-selection) is measured twice:

- **Classifier** (`make scene-bench`, `arion-scene-bench`): scores `domain.DecideScene` against [labelled app and site contexts](../../benchmarks/configs/scene-contexts.json) — common mail, messaging, coding, AI-assistant and search apps and sites, plus documents and unlisted targets. It reports accuracy, per-scene precision and recall, the confusion matrix, which signal decided and the decision time. The developer who wrote the rule tables also wrote the labels, so it measures coverage of listed targets; ambiguous cases are labelled by intended use so path-dependent misses stay visible.
- **Rewrite by scene** (`make live-text-bench … SCENE=auto CORRECT=1`): text cases carry a `target` (`app`, `field`, `site`) and an expected `scenario`. `SCENE=auto` decides each case's scene from its target exactly like the product; `SCENE=chat` reproduces the product before scene selection; `SCENE=corpus` uses the labelled scene. `CORRECT=1` applies the product's default rule correction before the rewrite, as dictation does. Rows record the scene, the deciding route and the rewriter's input; the scorer reports passes, worsening and scene agreement per expected scene. Layout proxies are exact strings: paragraph breaks for email, numbered lines for enumerated work and coding requests, no sentence-final punctuation for search, a single paragraph for chat.

Two developer-authored scene corpora exist: a 33-case set written before any scene prompt work and then used for diagnosis (now the development set), and a 30-case held-out set written and hashed after that diagnosis and before the final pipeline ran.

## Rewrite context

The context ablation ([ADR 013](../decisions/013-rewrite-context-on-device.md), [report](../../benchmarks/results/context-ablation-v1/README.md)) measures whether text around the cursor, the screen and recent dictations help the rewrite fix misheard names, terms and codes, and whether they leak into the output.

- **Fixtures.** Each case renders a realistic window (chat, mail, editor, document, issue page) as HTML to a 2× PNG with headless Chrome, and carries its true visible text, the field's text around the cursor, recent dictations, an ASR-like source with the error and exact-string proxies. Negative cases (distracting screens, prompt injection in the window, a different person with a similar name, unrelated history) must stay unchanged and must not take on screen text.
- **Screen text.** `make context-ocr` runs the production Vision recognizer (`macos.RecognizeImage`) on every image at the accurate and fast levels and scores character error against the true text. The rewrite bench takes screen text as `SCREEN=oracle` (true text, isolating recognition errors), `accurate` or `fast`.
- **Delivery.** `CONTEXT=cursor,history,screen` selects sources; `CONTEXT_AS=prompt` sends context sections with the `context-v1` template, `terms` sends only `domain.ContextTerms` in the plain prompt's terms (the product's choice), `both` does both. Validation always uses the whole context, as the product does.
- **Splits.** The 30-case held-out set was written and hashed before the context demonstrations. After its first ablation was inspected, a 16-case development set was written to choose the delivery mode and calibrate the validator; held-out was then run once more for the final rows.

## Limits

One author wrote every case; there are no independent annotations, no natural speech and no human ratings. Exact-string proxies can pass a fluent hallucination or fail an equivalent spelling. Synthetic speech has no disfluent prosody, and the inserted pauses make segmentation easier than real dictation. The host was not isolated from other desktop tasks. Native preview rendering, IME insertion and Windows were not measured. These results select defaults for development; they do not establish Typeless parity or satisfy RWT-A07/A08.

## Commands

```sh
make live-fixtures CORPUS=.workspace/live-heldout-corpus.json OUT=.workspace/live-audio-heldout
make live-text-bench CORPUS=.workspace/live-heldout-corpus.json LLM_MODEL=.workspace/ablation-models/qwen3-1.7b OUT=.workspace/benchmarks/live-text/heldout-1.7b.jsonl
make live-audio-bench MANIFEST=.workspace/live-audio-heldout/manifest.json CORPUS=.workspace/live-heldout-corpus.json MODELS=.workspace/models/store BUNDLE=xasr-160-xasr-final LLM_MODEL=.workspace/ablation-models/qwen3-1.7b OUT=.workspace/benchmarks/live-audio/heldout-1.7b.jsonl
make live-report RUNS=benchmarks/configs/live-rewrite-report.json OUT=benchmarks/results/live-rewrite-v1/summary.json
```

`LLM_RUNTIME` defaults to the staged `apps/desktop/resources/darwin-arm64/llm` (`make desktop-libs`); `SEGMENT=1` turns on per-segment finalization. Output files must not exist. The audio command keeps the host awake with `caffeinate` for the run.
