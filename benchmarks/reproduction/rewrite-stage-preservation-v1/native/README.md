# Native ASR and Vision diagnostic fixtures

These fixtures supply actual production macOS Vision OCR and actual production X-ASR streaming/offline-finalizer observations for rewrite-stage experiments. They do not independently reproduce an LLM error. The parent experiment must replay their observed source pairs through the real rewrite worker and report its results separately.

- `initial/`: eight developer-authored Mandarin named-entity and code-symbol cases, fixed before observing native results.
- `chatterfly/`: supplementary cases created after the user identified `Chatterfly` becoming `chad fly`. These are synthesized variants of that report, not the user's original audio or screen.

Each directory preserves `corpus.json`, synthetic PNGs, original synthesized WAV input bytes, the audio manifest, production ASR/OCR JSONL and logs, source generators, joined `pairs.json` and `provenance.json`. Provenance records archived file hashes, original paths, model verification manifests, executable hashes, host and pipeline settings. No weights or voice assets are included.

`pairs.json.preview` is the final observed streaming ASR `DraftReplaced` event before stop. It is not an LLM preview, and the existing benchmark does not independently expose a closed-segment stability flag. `final` is the coordinator's source after full-audio offline finalization and rule correction. `screen` is the accurate Vision output, never the oracle screen string. Exact expected entity strings are evaluator-only `checks`; they must not be added to inference requests as protected spans or approved memory.

All speech uses macOS `say` with the locally installed Tingting synthetic voice, rate 175, 16 kHz mono PCM16, with 1.2-second pauses between two parts, 0.3-second lead and 0.8-second trail. This records synthesis provenance, not an independent redistribution license for Apple voice assets. No human speaker, microphone, user editor or user desktop content was captured. Images are AppKit renderings of authored text. These are diagnostic cases, with no natural-speech, held-out quality, native UI latency or insertion claim.

For each directory, reproduce with a fresh output destination:

```sh
python3 -I build-live-audio.py corpus.json /new/private/audio
swift render.swift /directory/containing/corpus
arion-ocr -corpus corpus.json -out /new/private/ocr.jsonl
caffeinate -i arion-live-bench -manifest audio/manifest.json -corpus corpus.json \
  -models /verified/model/store -bundle xasr-160-xasr-final \
  -variant native-no-llm-xasr160 -out /new/private/asr.jsonl
```

Image rendering requires a new corpus directory without `images/`. Production binaries build from `apps/desktop/cmd/arion-ocr` and `apps/desktop/cmd/arion-live-bench` with cgo enabled on macOS. Existing outputs must not be overwritten. The original runs used no LLM worker, 4 threads, 20 ms realtime replay chunks, filler cleanup enabled and segment finalization disabled. Accurate and fast OCR outputs are both retained; the fast mode's poor Mandarin recognition must not be substituted for accurate screen context.

## Installed-worker coordinator development reproduction

`coordinator-memory-installed-v1/` runs the real production coordinator, X-ASR stream and offline finalizer, conservative correction and immutable installed worker (`selector-screen-v1-baseline+natural-layout-v1`). Its private copied driver only accepts explicit vocabulary fixtures and injects exact observed Vision text through a tracing pass-through rewriter. No output text is fabricated.

The mappings `Chapter fly → Chatterfly` and `Chanter fly → Chatterfly` were deliberately constructed **after** the initial ASR observations. This is an exposed development reproduction, never independent test data, preexisting user memory or held-out quality evidence.

Case 01 publishes seven validated LLM preview events containing `Chatterfly`. Its live rewrite request is `检查一下 Chatterfly` with recalled term `Chapter fly → Chatterfly`; the installed worker returns `检查 Chatterfly`. At stop the whole-audio recognizer supplies `检查一下 chatter fly，看看它的流式改写是不是正确。`. The vocabulary no longer matches that spelling, the final request has no recalled terms, and the installed worker preserves `chatter fly`. Thus it reproduces the reported class of visible correct preview followed by incorrect final text, with actual native inference. It does not reproduce the exact `chad fly` string, and does not show the formatter changing an unchanged canonical input.

Case 02's canonical live rewrite is cancelled before delivery; retain that result without counting it as a successful preview reproduction. All raw displayed events and actual worker requests/results are archived. Timings may overlap other source-worker experiments and cannot establish comparative performance.

## Candidate replay on the identical native fixtures

`coordinator-memory-candidate-v1/` retains the candidate run, source snapshots and hashes recorded before inference, raw requests/results, native events and exact source WAVs. The source hashes remained unchanged throughout the run. A pre-memory-continuity preparation snapshot is retained separately; no model invocation used that preparation.

All three candidate final rewrites retain `Chatterfly`: original cases 01 and 02, plus a case-01 duplicate with screen context disabled. Cases 01 and its memory-only duplicate each publish five validated preview events with the correct name before final delivery. Case 02's live rewrite is cancelled and is not counted as a successful preview. The offline recognizer still supplies `chatter fly`; the candidate receives the earlier recognized, rule-corrected source in `PreviousText` and relevant approved vocabulary from both recognized sources. It preserves the separate final source and does not substitute generated preview text.

The screen-off duplicate uses the identical WAV and deliberately constructed approved mapping. It demonstrates this exposed development reproduction with memory alone; it is not another independent audio sample or held-out quality result. Candidate runs use the product's 10-second final timeout and 256-token limit. The installed-baseline private driver requested a 60-second timeout and 384 tokens, while its installed formatter capped effective output at 256. These runs establish the observed behavior, not isolated comparative performance. Native UI insertion was not exercised.
