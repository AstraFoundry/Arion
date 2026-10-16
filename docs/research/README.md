# Research and evidence

Research separates **observed shipped artifacts**, **static inferences**, **vendor statements**, **Arion design choices**, and **unknowns**. Static symbols do not prove runtime execution or server behavior.

- [Typeless](typeless.md) — installed Electron artifact and audio/context/delivery surfaces.
- [Doubao](doubao.md) — installed native IMK artifact, online/offline ASR and composition surfaces.
- [Chatterfly](chatterfly.md) — installer payload and native scenario-aware post-processing paths.
- [Keyboard input methods](keyboard-ime.md) — Doubao/Chatterfly native Pinyin decoding, local candidate ranking, conditional model/cloud enhancement and composition/commit evidence.
- [Competitor onboarding](competitor-onboarding.md) — first-use flows, evidence limits and Arion verified-practice implementation gaps.
- [Competitor contextual rewriting](competitor-rewrite.md) — focused Typeless/Doubao refinement and organization evidence.
- [Competitor disfluency handling](competitor-disfluency.md) — Doubao decoder span annotations, organization request construction, Chatterfly short-input/result policies and language limits.
- [Disfluency implementation benchmark](../../benchmarks/results/disfluency-alignment-v2/README.md) — local deletion spans, independent organization, matched native workers and explicit remaining failures.
- [Rewrite reliability](rewrite-reliability.md) — Doubao/Chatterfly alignment, reproduced source-delivery defects, adopted fixes and remaining generation failures.
- [Learning from corrections](edit-learning.md) — how Typeless, Doubao and Chatterfly detect a user's fix of dictated text, store it and recall it.
- [ONNX rewrite models](onnx-rewrite-models.md) — pinned Hugging Face candidates, native runtime compatibility and load/parity gates.
- [Qwen pipeline diagnostics](qwen-pipeline-diagnostics.md) — actual development inference, scope boundaries and repeatable flows.
- [Qwen paper strategies](qwen-paper-strategies.md) — primary-paper mechanisms, contextual transcription/editing limits, training proposals and staged ablations.
- [Live rewrite benchmark](live-rewrite-benchmark.md) — production-path text and real-time audio benchmarks of the edit-while-speaking pipeline ([ADR 011](../decisions/011-live-local-rewriting.md)).
- [Rewrite evaluation](rewrite-evaluation.md) — isolated quality tests, native latency, contention, ablations and blind human review.
- [MyGo](mygo.md) — pinned source review, capabilities and required OS adapters.
- [On-device speech and ranking](on-device-speech.md) — local runtime/model comparison and evaluation gates.
- [Phonetic ASR correction](phonetic-asr-correction.md) — audio-derived versus text-derived pronunciation, primary papers, decoder uncertainty interfaces and bounded Doubao/Chatterfly evidence.
- [Phonetic ASR measured results](../../benchmarks/results/phonetic-asr-1.7b-v2/README.md) — complete 154-case Qwen3-1.7B ablation, acoustic second-pass control, confidence routing and retained failures; 4B cancelled by the user.
- [Phonetic ASR benchmark procedure](phonetic-asr-benchmark.md) — frozen acoustic/pinyin/score/IPA ablations, reference isolation, corruption metrics and compute-time boundaries.
- [Benchmark methodology](benchmark-methodology.md) — paper mapping, exact metrics, paired CIs and ablation limits.
- [Streaming datasets](streaming-datasets.md) — pinned Hugging Face corpora, permissions and realtime replay.
- [Evaluation procedure](evaluation.md) — reproducible measurements and acceptance boundaries.
- [Evidence](evidence/README.md) — sanitized artifact identity and provenance.

Competitors provide workflow evidence, not reusable implementations or approved models. Do not copy packaged code, private endpoints, keys, audio or personal databases into Arion. Local expanded analysis files stay in ignored `.workspace/research/`.

- [Competitor diagnostics](competitor-diagnostics.md) — recorded Typeless, Doubao and Chatterfly stage evidence and Arion's content-free logging boundaries.
