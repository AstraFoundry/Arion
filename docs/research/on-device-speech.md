# On-device speech recognition and correction

Status: source-backed research and an evaluation recommendation. No candidate was benchmarked or integrated in this investigation. A framework supporting an OS does not establish that Arion's signed application, model bundle, or Go bindings work on that OS.

## Recommendation for desktop agents

Start with **sherpa-onnx on CPU through its official Go/cgo bindings**, evaluating a streaming Chinese-English Zipformer transducer. Compare streaming Paraformer through the same abstraction. This is a proposed engineering baseline, not a production model selection. It fits a Go/MyGo host, supports incremental hypotheses, and leaves a route to mobile without putting Python in the installed application. The official [Go API](https://k2-fsa.github.io/sherpa/onnx/go-api/index.html) lists macOS arm64/x86_64 and Windows x86_64/x86 packages; Windows arm64 needs a separate build and support investigation.

Use a two-pass design: incremental ASR for preview, then bounded local finalization. Evaluate offline Paraformer or SenseVoice for the second pass and **llama.cpp** for optional meaning-preserving text correction. Keep correction independent from recognition. An LLM cannot recover acoustic evidence from a transcript alone, and sentence cleanup must not substitute for ASR accuracy evaluation.

Benchmark current **Moonshine streaming** alongside the baseline: the current catalog includes Mandarin Tiny Streaming as well as English streaming models. Claims that Moonshine is English-only or has only non-streaming models are outdated. Chinese-English code switching in one checkpoint remains unverified. [Current model catalog](https://moonshine-voice.readthedocs.io/en/latest/models/available-models/).

Keep whisper.cpp as an offline accuracy/reference candidate and an optional later adapter. Its microphone example repeatedly processes windows; it does not establish that a Whisper checkpoint has a stateful causal encoder. MLX is an optional Apple Silicon optimization experiment, not the shared Windows engine.

## Evidence and terminology

This report uses official repository documentation, model cards, headers, and license files. Context7 resolution and documentation queries were performed for sherpa-onnx, FunASR, whisper.cpp, Moonshine, llama.cpp, MLX, Sentence Transformers, ONNX Runtime, TypeSafe AI, and Laya; important model and license details were checked against primary web sources. These are source observations, not observations of installed inference running.

- **Stateful streaming ASR:** consumes sequential audio chunks with retained model/decoder state and emits hypotheses before the utterance ends. Chunked input can be genuine streaming; the distinction is state and future-context requirements.
- **Windowed offline ASR:** repeatedly decodes complete or overlapping windows with an offline model. It can present live text, but requires additional stabilization and repeated work.
- **Acoustic second pass:** re-decodes retained utterance audio to improve recognition after more evidence arrives.
- **Text correction:** changes recognized text using rules, vocabulary, punctuation, or a language model. This requires independent faithfulness checks.
- **Final:** an application lifecycle state, not a claim that the model's text is necessarily correct.

No upstream throughput claim is an Arion latency result. A short GPU inference measurement excludes capture, buffering, lookahead, endpoint delay, queueing, correction, and host insertion.

## Candidate comparison

| Candidate | Streaming behavior | Language evidence | Desktop integration | Arion role |
|---|---|---|---|---|
| sherpa-onnx + streaming Zipformer | Online transducer with incremental decoder state | Documented bilingual Chinese-English checkpoints | C/C++ runtime, C ABI, official Go/cgo packages | First baseline for live preview and contextual biasing |
| Streaming Paraformer | Stateful online model; explicit chunk cache and lookahead | Current streaming model card lists Chinese and English | FunASR Python/C++ ecosystem; compatible exports can use sherpa-onnx | Compare recognition and resource costs with Zipformer |
| Offline Paraformer / FunASR two-pass | Online hypothesis followed by offline revision; offline pass is not causal streaming | Depends on selected checkpoint | Native ONNX runtime exists; deployment path must be checked per OS | Acoustic finalization candidate |
| whisper.cpp | Rolling-window microphone example or VAD-triggered offline decode | Whisper multilingual checkpoints include Chinese and English; `.en` checkpoints are English-only | C/C++, C API; macOS/Windows CPU and optional accelerators | Offline reference and optional finalization |
| SenseVoiceSmall | Offline non-autoregressive ASR; segmenting with VAD does not create a streaming checkpoint | Released Small supports Mandarin, Cantonese, English, Japanese, Korean | sherpa-onnx ONNX or the upstream native runtime | Bounded second-pass candidate |
| Moonshine Voice streaming | Current stream lifecycle API and streaming encoder model families | English sizes and Mandarin Tiny Streaming are catalogued separately | C++ core, C ABI, ONNX Runtime; macOS/Windows builds | Lightweight challenger; mixed-language test required |
| llama.cpp + a small instruction LLM | Token streaming is text generation, not streaming speech recognition | Depends on LLM checkpoint | C/C++, C API, CPU/Metal/CUDA/Vulkan options | Optional text correction, with constrained output and semantic checks |
| MLX / MLX-LM | Framework for model computation/text generation | Depends on LLM checkpoint | Apple Silicon/Metal; current Linux CPU/CUDA packages also exist | Optional macOS experiment; no verified Windows path |

## Runtime and model-specific findings

### sherpa-onnx

Its public C API includes online streams, result snapshots, endpoint detection, and explicit ownership/release operations. The adapter should copy text before freeing a result and serialize engine/stream ownership. The header also exposes per-model provider configuration; availability in the header does not guarantee that a shipped binary includes that execution provider. [C API](https://github.com/k2-fsa/sherpa-onnx/blob/master/sherpa-onnx/c-api/c-api.h).

The first bilingual candidate is the documented `sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20` family. The upstream inventory maps its conversion to a community Chinese-English model. The card declares Apache-2.0, but Arion must still record the precise original revision, conversion provenance, license files, and artifact hashes. A release archive name is insufficient provenance. [Bilingual inventory](https://k2-fsa.github.io/sherpa/onnx/pretrained_models/online-transducer/zipformer-transducer-models.html), [upstream model card](https://huggingface.co/csukuangfj/k2fsa-zipformer-chinese-english-mixed).

Contextual biasing is a concrete advantage for synchronized vocabulary: sherpa's documented hotword path requires a **transducer** and `modified_beam_search`. It is not a generic capability of Paraformer or SenseVoice. Report adapter capabilities explicitly and benchmark over-biasing, especially names that sound like common words. [Hotword constraints](https://k2-fsa.github.io/sherpa/onnx/hotwords/index.html).

The official examples cover punctuation and VAD separately from ASR. Treat punctuation, normalization, and denoising as separately versioned model stages. Start with a CPU backend for a common hardware baseline; add acceleration only after end-to-end measurements demonstrate a benefit. [Go examples](https://github.com/k2-fsa/sherpa-onnx/blob/master/go-api-examples/README.md).

### FunASR and Paraformer

The streaming recipe carries a cache between calls and flushes it with `is_final`. The documented `[0, 10, 5]` configuration uses 600 ms input chunks and 300 ms lookahead; `[0, 8, 4]` reduces chunk granularity. These are model scheduling parameters, not measured end-to-end latency. At 16 kHz, 600 ms is 9,600 samples; do not copy inconsistent prose that confuses a base frame unit with the full chunk. [Streaming recipe](https://github.com/modelscope/FunASR/blob/main/examples/industrial_data_pretraining/paraformer/README.md).

The native two-pass tooling combines online and offline models with VAD and punctuation. This is useful architectural evidence for a preview/finalization split; its server and benchmark configuration should not be copied wholesale into a desktop application. Packaging a complete Python/PyTorch environment has different startup, memory, and distribution costs from the C/ONNX adapter. [Native two-pass interface](https://github.com/modelscope/FunASR/blob/main/runtime/docs/onnxruntime_binary_output.md).

The current `funasr/paraformer-zh-streaming` card publishes weights under Apache-2.0 at its immutable `apache-2.0-20260804` tag and explicitly distinguishes an older revision that lacked the referenced license file. This statement is revision-specific. Existing sherpa release exports do not automatically become the newly tagged artifact; document or reproduce the conversion. [Model license and revision](https://huggingface.co/funasr/paraformer-zh-streaming).

### whisper.cpp

The `whisper-stream` example uses repeated inference and offers a sliding-window/VAD mode. Its example step size must not become a promised first-token latency. Whisper can revise earlier window text, so an adapter needs overlap reconciliation, provisional revisions, and duplicate suppression. [Streaming example](https://github.com/ggml-org/whisper.cpp/blob/master/examples/stream/README.md).

The runtime provides a C-style API, CPU inference, quantization, and Apple Silicon optimizations including Metal and an optional Core ML encoder path. Windows acceleration can be evaluated with appropriate Vulkan or CUDA builds, but device/backend compatibility is a packaging and benchmarking obligation. [Runtime and backend documentation](https://github.com/ggml-org/whisper.cpp).

The runtime is MIT; original OpenAI Whisper code and model weights are also released under MIT. A third-party fine-tune or conversion needs its own provenance check. [whisper.cpp license](https://github.com/ggml-org/whisper.cpp/blob/master/LICENSE), [Whisper model license statement](https://github.com/openai/whisper#license).

### SenseVoice

The released Small checkpoint is a multilingual offline model with language identification and extra emotion/audio-event tags. Strip those tags through a documented decoding/postprocessing step before text insertion. The research's broader language claims do not expand the language scope of the released checkpoint. The model is useful for retained-audio finalization, not evidence of causal partial output. [Checkpoint scope and inference](https://github.com/QwenAudio/SenseVoice), [paper](https://arxiv.org/abs/2407.04051).

The official model card currently declares `model-license`; the repository's MIT code license is a different artifact. A redistribution decision requires the model agreement and exact converted bundle notices. Do not label all SenseVoice weights MIT or Apache-2.0 because an inference runtime uses either license. [SenseVoiceSmall model card](https://huggingface.co/FunAudioLLM/SenseVoiceSmall).

### Moonshine

The current streaming catalog includes Mandarin Tiny Streaming and several English sizes. Its runtime uses ONNX/ORT artifacts; use the catalog corresponding to the pinned runtime, not historical Tiny/Base examples. The catalogue reports evaluation panels with different aggregation methods, so their published Chinese CER and English WER cannot determine a cross-model winner. [Current catalog and evaluation caveats](https://moonshine-voice.readthedocs.io/en/latest/models/available-models/).

Its C API exposes create/start/add-audio/transcribe/stop/free operations, which can be wrapped with cgo. This is a native integration route, not a verified official Go binding. Verify transcript revision/completion behavior and cancellation against the selected release. [C API](https://moonshine-voice.readthedocs.io/en/latest/api/c-api/), [C++ builds and Windows test scripts](https://moonshine-voice.readthedocs.io/en/latest/using/building-from-source/).

Current code and streaming STT models are MIT by default, with listed legacy non-English non-streaming models under the Community License. That agreement contains additional terms; select a specific streaming artifact and retain its notices rather than assuming every Moonshine download has identical terms. [License and exceptions](https://github.com/moonshine-ai/moonshine/blob/main/LICENSE).

### Local correction: llama.cpp and MLX

Use llama.cpp as the first cross-platform LLM experiment. Constrain output to a small correction object or replacement text, cap context/output, and cancel superseded work. Its grammar/JSON-schema support controls structure; unsupported schema features can be skipped, so inspect the generated grammar and validate the returned object in application code. Structural validity does not establish semantic faithfulness. [Grammar limitations](https://github.com/ggml-org/llama.cpp/blob/master/grammars/README.md), [C API](https://github.com/ggml-org/llama.cpp/blob/master/include/llama.h).

A small Qwen checkpoint is one experiment, not an approved correction model. The Qwen3-0.6B card provides a compact multilingual instruction candidate under Apache-2.0 and distinguishes thinking/non-thinking modes. Test non-thinking correction and preserve its required chat template; quantization and exact GGUF provenance remain to be selected. [Model card](https://huggingface.co/Qwen/Qwen3-0.6B). llama.cpp itself is MIT. [Runtime license](https://github.com/ggml-org/llama.cpp/blob/master/LICENSE).

MLX's current installation documentation also includes Linux CPU/CUDA packages, so describing it as exclusively macOS is too broad. Its Apple Silicon path is relevant to Arion, while no supported Windows packaging path was established here. Do not create two correction implementations before common model semantics and evaluation are stable. [MLX supported installation paths](https://github.com/ml-explore/mlx), [MLX license](https://github.com/ml-explore/mlx/blob/main/LICENSE).

## Local candidate ranking for a configurable system IME

Exact-name searches identify **Jev, TypeSafe AI's System One decision model**. The official introduction describes state plus typed questions producing Choice, Score, and Noul outputs. Jev refers to typed decisions rather than an embedding or JEPA model. Its design is relevant to choosing among existing words without generating replacement text. [Official Jev introduction](https://docs.typesafe.ai/introduction), [TypeSafe launch explanation](https://typesafe.ai/blog/introducing-system-one-models-and-jev).

This research proposes a **selection-only ranker**: given permitted context and an existing ordered candidate list, return scores and an ordering of those same candidate IDs. It cannot invent a word, rewrite candidate text, change the composition span, or implicitly commit text. The native IME adapter owns composition and explicit candidate selection. Configurable system-IME behavior and a push-to-talk text-insertion workflow require distinct platform contracts; MyGo application windows do not by themselves provide a system input-method implementation.

### Ranking approaches

| Approach | Scoring behavior | Advantages to test | Important limitations |
|---|---|---|---|
| Source ordering + authorized personal vocabulary | Existing decoder/lexicon preferences; no neural scorer | Cheapest baseline; personalized names and terms | Needs evaluated weighting; memory can over-bias homophones |
| Embedding bi-encoder | Encodes context and candidates independently, compares vectors | Candidate caching; useful for retrieving relevant memory | Semantic similarity does not directly measure whether a word completes the current composition |
| Cross-encoder pair scorer | Jointly processes context and each candidate-bearing continuation | Direct interaction between context and alternatives | More work per candidate; generic retrieval training may not transfer to IME disambiguation |
| Jev-style typed decision model | Choice over supplied options, rubric Score, or Noul truth judgment | Non-generative decisions; bounded answer space | Hosted Jev is not an established local bundle; independent open implementations require their own calibration and IME evaluation |

Sentence Transformers documents the difference between independent embedding representations and joint cross-encoder scores. For a small IME candidate list, a pair scorer is a stronger initial neural evaluation hypothesis; a bi-encoder is a useful memory-retrieval comparator. That is an Arion design inference, not a benchmark conclusion. [Cross-encoder applications](https://sbert.net/examples/cross_encoder/applications/README.html).

**Qwen3-Reranker-0.6B** is one concrete comparison candidate. Its official card declares Apache-2.0 and multilingual support, documents scores from the final-position `yes`/`no` logits, and provides Sentence Transformers integration. Preserve its tokenizer, task template, padding, token IDs, and score transform. The default retrieval instruction is not an IME task specification. Evaluate scoring candidate-bearing completions with an explicit selection instruction; do not run open-ended generation and call it ranking. Published retrieval benchmarks do not measure Chinese homophone or keyboard candidate top-1 accuracy. [Official reranker card](https://huggingface.co/Qwen/Qwen3-Reranker-0.6B).

### Exact Jev search and open implementations

The discovery searched exact `jev` with model, speech recognition, input method, Chinese candidate words, embeddings/rerankers, GitHub, and Hugging Face. The review followed primary vendor documentation and authors' repositories/model cards. Search-result directories, third-party gateways using Jev-like domain names, and community benchmark summaries were discovery leads only; they do not establish official affiliation, local model availability, or license rights. This is a bounded source review, not an exhaustive claim that no other implementations exist.

The official TypeSafe documentation presents an API/SDK model. No downloadable official Jev weights or native local inference package was established in the reviewed vendor docs and linked public tooling. Accordingly, Arion's default offline pipeline cannot depend on hosted Jev. The open official [System One Adapter](https://github.com/typesafe-ai/system-one-adapter-python) is an MIT adapter backed by other LLM APIs, not a release of Jev weights or its training recipe.

| Project | Observed artifact and availability | License boundary | Suitability for Arion |
|---|---|---|---|
| [Laya](https://github.com/NandhaKishorM/laya) | Independent encoder/decision-head engine, published English and multilingual checkpoints, Python and ONNX paths | Apache-2.0 code; verify selected checkpoint card separately | First Jev-style local evaluation candidate |
| [Laya Multilingual checkpoint](https://huggingface.co/convaiinnovations/laya-multilingual) | Published mmBERT-based approximately 322M model, multilingual typed decisions | Model card declares Apache-2.0 | Evaluate Chinese-English candidate Choice; no IME benchmark yet |
| [Laya-MLX](https://github.com/mizorewww/laya-mlx) | Independent Apple Silicon runtime and converted weights; preserves typed-decision heads | Apache-2.0 runtime; conversion provenance and original weights also required | macOS optimization challenger after shared semantics are stable |
| [Chinese-Jev](https://github.com/gulucaptain/Chinese-Jev) | Encoder-based dataset/training/calibration/evaluation code; README explicitly says pretrained weights and real-world datasets are not included, model links pending | Apache-2.0 repository; future weights/data need separate verification | Training research reference, not a ready downloadable ranking model |
| [OpenJev](https://huggingface.co/openjev/openjev) | Published decision-model checkpoint and local endpoint instructions | CC BY-NC-4.0 model card | Research-only comparison pending suitability; exclude from a commercial default bundle |
| [OpenJev-RLCD](https://github.com/ZimmyGao/openjev-rlcd) | Independent training/calibration study, code, result files, and reproduction scripts | MIT code; datasets retain their own terms, including a non-commercial dataset | Calibration/training reference; not a validated desktop candidate-ranker package |
| [Simple Jev](https://github.com/featherless-ai/simple-jev) | Local Transformers/PyTorch server reads open model next-token logits for structured decisions; no separate classifier-head training required | Apache-2.0 code; backbone model terms still apply | Useful logits-based baseline; native Go port and deployment remain work |
| [hotchpotch/jev-reranker](https://github.com/hotchpotch/jev-reranker) | Python pointwise/pairwise/listwise relevance filtering using the hosted TypeSafe API | MIT client code, not local model weights | Ranking-policy reference; cannot meet offline inference by itself |
| [uspraveen/Jev-Reranker](https://github.com/uspraveen/jev-reranker) | Hosted Jev relevance/utility/conflict judgments plus deterministic memory-selection policy | Client implementation does not license or provide the model | Useful separation of model judgments from policy; no demonstrated IME word-ranking result |

**Proposed shortlist:** evaluate pinned `laya-multilingual` using a single Choice over the existing candidate list against source ordering/personal vocabulary and Qwen3-Reranker-0.6B pair scoring. A Choice distribution is conditional on its option set; it is not an acoustic likelihood and cannot be compared across independently pruned groups as though all candidates were scored together. A separate per-candidate Noul asks a different, absolute question. Official Jev documentation warns against reusing thresholds across these primitives and against assuming arithmetic identities between independent answers. [Primitive limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).

For Laya, inspect input truncation and the shared option-token budget; long option descriptions can erase distinctions between candidates. Check order sensitivity by permuting labels. Pin the multilingual checkpoint explicitly during mixed-language evaluation rather than allowing a heuristic router to change the engine between measurements. Fit/check confidence calibration on held-out IME decisions and separately report top-1 accuracy. Vendor and community claims of type safety or no free-form hallucination do not imply that a selected candidate is correct. [Upstream runtime constraints](https://github.com/NandhaKishorM/laya).

The current Laya ONNX adapter rejects non-sequential `option_layout`; its exported inputs include `input_ids`, `attention_mask`, `marker_pos`, `marker_mask`, and `qtype`, with `logits` and `act_logits` outputs. Native integration needs its special sequence construction, marker positions, typed-head decoding, and temperature handling, not just generic text tokenization. Use the inspected [ONNX adapter](https://github.com/NandhaKishorM/laya/blob/main/laya/onnx_agent.py) and [exporter](https://github.com/NandhaKishorM/laya/blob/main/scripts/export_onnx.py) as the parity reference. Unsupported layouts must fail explicitly.

Laya-MLX is a relevant local macOS implementation, with author-reported short-input M3 Max latency and validation evidence. Those numbers exclude model load and are not Arion keyboard/ASR measurements. Its Python-facing MLX runtime also does not automatically give a Go host a native embedding interface; assess the adapter/package boundary separately. It provides no established Windows runtime. [Port and benchmark boundaries](https://github.com/mizorewww/laya-mlx).

### Native deployment boundary

Sentence Transformers is suitable for offline evaluation/export, rather than an assumed Python dependency in the shipped desktop host. Its ONNX documentation supports export and CPU-oriented dynamic int8 quantization for compatible CrossEncoders. The documented generic route uses sequence-classification exports; this alone does not prove that a causal-logit Qwen3 reranker can use the same route unchanged. Verify exporter and exact checkpoint compatibility before choosing ONNX, a native llama.cpp scorer, or another runtime. Match quantization to actual arm64/x86_64 hardware and test rank changes. [Export and quantization documentation](https://sbert.net/docs/cross_encoder/usage/efficiency.html).

A compatible exported graph can be loaded through ONNX Runtime's C ABI from a Go/cgo infrastructure adapter. Tokenization, pair formatting, model input names/types/shapes, output interpretation, and any normalization/pooling outside the graph remain adapter responsibilities. Compare native scores/ranks against a pinned reference implementation; loading an ONNX file is insufficient validation. [Native C integration](https://onnxruntime.ai/docs/get-started/with-c.html). ONNX Runtime is MIT; Sentence Transformers is Apache-2.0, and selected weights/conversions retain their separate license obligations. [Runtime license](https://github.com/microsoft/onnxruntime/blob/main/LICENSE), [export-tool license](https://github.com/huggingface/sentence-transformers/blob/main/LICENSE).

### Candidate and evaluation gates

- **Real alternatives:** the candidate producer must actually expose distinct alternatives. The inspected sherpa public online result describes a transcript/token snapshot, not a documented scored n-best list. `modified_beam_search` retaining internal paths does not prove the chosen Go/C API exports them. Advertise alternatives only after API/runtime verification; a single result provides nothing to rerank. For typed input, require the configured IME candidate engine's own candidate API; do not assume access to every third-party IME's candidates. [Inspected ASR result contract](https://github.com/k2-fsa/sherpa-onnx/blob/master/sherpa-onnx/c-api/c-api.h).
- **Selection invariant:** output contains only input candidate IDs; preserve exact text, source scores/order, and phonetic/composition eligibility. Bind each result to the composition revision and context snapshot; cancel obsolete work and use deterministic ties. Surface unavailable/error states explicitly.
- **Quality:** report candidate-set oracle recall separately from top-1 accuracy and mean reciprocal rank. Measure improvement and regression for Chinese homophones, names, code terms, mixed language, numbers, and short/absent context against source ordering and personal-vocabulary baselines. If the intended word is absent, a selection-only ranker cannot recover it.
- **Latency and privacy:** measure warm p50/p95 per candidate count and context length, cold load, tokenizer time, memory, and concurrent ASR contention. Keep ranking off the capture callback and IME composition thread. Test disconnected inference and sensitive-field context exclusion. Set the product's keyboard interaction budget before accepting a model; no latency result is claimed here.
- **Selection:** exact Jev is identified; require reproducible accuracy gain for a local candidate, bounded interaction latency, native/reference parity, OS packaging, and complete artifact provenance. Laya Multilingual is the proposed Jev-style evaluation candidate, not a locked or approved production model.

## Proposed application pipeline

These are Arion design recommendations, not claims about competitors:

1. Capture through the OS adapter; convert to the selected model's required mono sample format outside the real-time capture callback.
2. Feed bounded frames to a single-owner ASR stream. Keep sequence/sample offsets and explicit overflow errors; never silently drop frames.
3. Publish revisioned hypotheses to Arion's preview surface. Never stream unfinished rewrites into an arbitrary target application.
4. Use endpointing or explicit stop to close an utterance and flush ASR state. Retain only the bounded audio necessary for the optional acoustic second pass.
5. Apply versioned normalization/punctuation and supported vocabulary biasing. Preserve the raw ASR hypothesis separately from corrected text.
6. If enabled, run local text correction on a stable phrase with an allowlisted context snapshot. Treat that context as data and validate protected spans and meaning.
7. Commit once using a target/focus lease validated by the platform adapter. A stale correction may update a preview only if its revision still matches; it cannot replace already committed user text implicitly.
8. Release audio and inference state at completion/cancel. Synchronize user-authorized vocabulary/preferences/correction memory independently from the audio path.

Memory sync should supply explicit vocabulary and permitted context, not force a network dependency into recognition. Missing, corrupt, incompatible, or unlicensed model artifacts must produce a visible local error. Do not silently switch to a remote service.

## Required evaluation before model selection

Measure with identical recorded inputs paced as live audio and then repeat with OS microphone capture. Report cold startup separately from a warm loaded model. Pin runtime commit, weight revision, checksums, conversion tools, quantization, sample format, endpoint settings, thread count, execution provider, hardware, power state, and OS version.

| Dimension | Required evidence |
|---|---|
| Latency | Speech onset to first nonempty hypothesis; first stable prefix; stop/last voiced sample to final text; correction and insertion times; p50/p95 distributions |
| Compute | Inference/audio real-time factor, capture queue lag, missed frames, CPU/GPU use, resident/peak memory, model download/disk footprint |
| Languages | Chinese CER, English WER, mixed-language token error rate, numbers/dates/currency, proper names and code terms |
| Stability | Revision count, changed characters per partial, duplication at chunk boundaries, endpoint splitting/merging |
| Correction | Improvement and regression counts; invented/deleted information; changed numbers/names/negations; instruction injection from context |
| Conditions | Quiet/noisy speech, accents, rapid code switching, silence/music, long dictation, device loss, cancel/restart, overlapping inference jobs |
| Privacy | Fully disconnected transcription/correction, bounded audio lifetime, sanitized logs, no inference-related network connections |
| Packaging | Clean-machine macOS arm64/x86_64 and Windows x86_64 loads; dependent libraries and signatures; separately investigate Windows arm64 |

Acceptance thresholds belong in the product specification and must be declared as targets, never presented as measured results. Select a production bundle only after this matrix is reproducible and the model license/provenance record is complete.

## Remaining unknowns and handoff

- No Arion-specific latency, battery, accuracy, or memory measurements exist yet; upstream benchmark results are insufficient to rank the candidates for this input-method workload.
- No speech engine was loaded, no model weights were downloaded, and no macOS/Windows capture-to-insertion pipeline was exercised.
- Verify the exact sherpa Go package/build chain alongside MyGo, including Windows DLL deployment and compiler compatibility; framework OS support alone is insufficient.
- Establish artifact-level permission/provenance for Zipformer, every Paraformer conversion, SenseVoice, VAD, punctuation, and the selected LLM before distributing a bundle.
- Evaluate Mandarin-English code switching for Moonshine; separate catalog languages do not prove bilingual decoding.
- Decide whether acoustic finalization is worth a second model's memory/load cost before enabling it by default.
- Validate correction quality on product-specific content. Neither a runtime nor a small general LLM constitutes a proven real-time correction framework.

The next agent can implement an engine-neutral replay benchmark and a sherpa streaming adapter using these sources, then compare candidate models without changing capture, session, or insertion contracts.
