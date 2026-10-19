# Research and evidence

Research separates **observed shipped artifacts**, **static inferences**, **vendor statements**, **Arion design choices**, and **unknowns**. Static symbols do not prove runtime execution or server behavior.

- [Typeless](typeless.md) — installed Electron artifact and audio/context/delivery surfaces.
- [Doubao](doubao.md) — installed native IMK artifact, online/offline ASR and composition surfaces.
- [Doubao iOS](doubao-ios.md) — App Store IPA voice/keyboard split, permission and account synchronization declarations, native typing engine and FairPlay evidence limits.
- [Chatterfly](chatterfly.md) — installer payload and native scenario-aware post-processing paths.
- [Keyboard input methods](keyboard-ime.md) — Doubao/Chatterfly native Pinyin decoding, local candidate ranking, conditional model/cloud enhancement and composition/commit evidence.
- [Installed Chatterfly keyboard](chatterfly-installed-keyboard.md) — matching installed bytes, XNet loaders, model enablement, bounded provisioning search, stored cloud preference and real keyboard probes.
- [Chatterfly candidate UI](chatterfly-candidate-ui.md) — shipped candidate-bar size presets, light/dark themes and input-menu icon format behind Arion's candidate bar.
- [Typing-time candidate re-ranking](context-ranking.md) — Doubao's feature/MLP candidate sort and user-bigram context, Chatterfly's 上下文调频 option and Arion's local context ranker.
- [Input-source onboarding](input-source-onboarding.md) — bounded Doubao/Chatterfly register, enable, selection, notification, manual guide and refresh evidence with version limits.
- [IMK typing alignment](imk-typing-alignment.md) — focused competitor paging/editing/mode-switch evidence, Arion composition ownership fixes, full-keyboard implementation gates and restored post-profile diagnostics with system acceptance deferred.
- [Doubao keyboard commit chain](doubao-keyboard-commit.md) — byte-verified keyboard selection-to-host insertion and partial reconversion paths, with exact artifact identity and static evidence limits.
- [Competitor onboarding](competitor-onboarding.md) — first-use flows, evidence limits and Arion verified-practice implementation gaps.
- [Competitor contextual rewriting](competitor-rewrite.md) — focused Typeless/Doubao refinement and organization evidence.
- [Competitor disfluency handling](competitor-disfluency.md) — Doubao decoder span annotations, organization request construction, Chatterfly short-input/result policies and language limits.
- [Disfluency implementation benchmark](../../benchmarks/results/disfluency-alignment-v2/README.md) — local deletion spans, independent organization, matched native workers and explicit remaining failures.
- [Rewrite reliability](rewrite-reliability.md) — Doubao/Chatterfly alignment, reproduced source-delivery defects, adopted fixes and remaining generation failures.
- [Competitor cloud synchronization](competitor-cloud-sync.md) — account vocabulary, selected preferences, optional history, clipboard transfer and cloud-processing boundaries.
- [Learning from corrections](edit-learning.md) — how Typeless, Doubao and Chatterfly detect a user's fix of dictated text, store it and recall it.
- [Recognition-stage correction memory](user-modify-asr.md) — Qwen3-ASR-1.7B context interfaces, MLX/sherpa feasibility, known Doubao matching limits and a measured five-arm audio-stage diagnostic.
- [Recognition memory optimization](user-modify-asr-optimization.md) — contextual-ASR papers, correction-pair failure hypotheses, compact term retrieval and measured original-audio re-decoding on a frozen new test.
- [Closed-loop memory and contextual disambiguation](closed-loop-memory.md) — user feedback papers, empty-vault acquisition/reload/audio evaluation, ordinary-word ambiguity, OCR-to-prompt limits and retrieved pronunciation examples.
- [ONNX rewrite models](onnx-rewrite-models.md) — pinned Hugging Face candidates, native runtime compatibility and load/parity gates.
- [Qwen pipeline diagnostics](qwen-pipeline-diagnostics.md) — actual development inference, scope boundaries and repeatable flows.
- [Qwen paper strategies](qwen-paper-strategies.md) — primary-paper mechanisms, contextual transcription/editing limits, training proposals and staged ablations.
- [Live rewrite benchmark](live-rewrite-benchmark.md) — production-path text and real-time audio benchmarks of the edit-while-speaking pipeline ([ADR 011](../decisions/011-live-local-rewriting.md)).
- [ASR correction chain benchmark](../../benchmarks/results/correction-chain-v1/README.md) — six actual ASR/correction/intent-rewrite ablations, correction harm, acoustic controls and source-linked replay.
- [Flexible semantic reassessment](../../benchmarks/results/correction-chain-semantic-v1/README.md) — all original outputs reviewed with paraphrase/structure freedom, explicit agent-review limits and successful FastCorrect acquisition.
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

- [Expanded correction-chain study](../../benchmarks/reproduction/correction-chain-expanded-v2/README.md) — new 100-case synthetic corpus and paired flexible semantic evaluation.

- [Input-method consent and payload follow-up](../../benchmarks/runs/imk-keyboard-v11/README.md) — observed development-host identity, user-reported consent limits, corrective installer sources and new formal build validation.
