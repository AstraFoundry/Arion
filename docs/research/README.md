# Research and evidence

Research separates **observed shipped artifacts**, **static inferences**, **vendor statements**, **Arion design choices**, and **unknowns**. Static symbols do not prove runtime execution or server behavior.

- [Update notifications and automatic installation](update-notifications.md) — open issues #12/#13, macOS/Windows registration, pinned Magpie update behavior, and staged installation/lifecycle safety proposal.

- [Typeless](typeless.md) — installed Electron artifact and audio/context/delivery surfaces.
- [Doubao](doubao.md) — installed native IMK artifact, online/offline ASR and composition surfaces.
- [Doubao iOS](doubao-ios.md) — App Store IPA voice/keyboard split, permission and account synchronization declarations, native typing engine and FairPlay evidence limits.
- [Chatterfly](chatterfly.md) — installer payload and native scenario-aware post-processing paths.
- [Rewrite stage preservation](rewrite-stage-preservation.md) — revalidated competitor stage boundaries, a real preview-to-final regression, and paired native pipeline diagnostics.
- [Keyboard input methods](keyboard-ime.md) — Doubao/Chatterfly native Pinyin decoding, local candidate ranking, conditional model/cloud enhancement and composition/commit evidence.
- [Emoji candidate proposal](emoji-candidate-plan.md) — open-source references, typed Rime candidates, Unicode data, text-ranking and learning isolation, proposed rollout and native acceptance gates.
- [Installed Chatterfly keyboard](chatterfly-installed-keyboard.md) — matching installed bytes, XNet loaders, model enablement, bounded provisioning search, stored cloud preference and real keyboard probes.
- [Chatterfly candidate UI](chatterfly-candidate-ui.md) — shipped candidate-bar size presets, light/dark themes and input-menu icon format behind Arion's candidate bar.
- [English candidate admission](english-candidate-admission.md) — vendor-grounded exact/suggestion/lookup separation, full-Pinyin evidence, conservative local policy and validation limits.
- [Typing-time candidate re-ranking](context-ranking.md) — current Doubao MLP/LLM fusion and Chatterfly LM/LTR/conditional generation call paths, evidence limits and Arion's local context ranker.
- [General LLM keyboard alignment](llm-ime-alignment.md) — implemented ID-only local/provider adapter, latest-revision scheduler, measured 2B pilot, native integration design and evaluation/adoption gates.
- [Original Rime LLM reranker reproduction](../../benchmarks/results/rime-llm-reranker-v1/README.md) — 100 authored real-candidate fixtures, actual local inference, typing latency, parser/cache defects and model-free report replay.
- [Voice typewriter](voice-typewriter.md) — Doubao's spring-paced ASR typewriter and replace decisions, Chatterfly's speech-bar animations and Arion's paced, fading dictation text.
- [Input-source onboarding](input-source-onboarding.md) — bounded Doubao/Chatterfly register, enable, selection, notification, manual guide and refresh evidence with version limits.
- [Windows IME installation](windows-ime-installation.md) — supplied Doubao/Chatterfly installer extraction, REA evidence, native TSF registration and deployment comparison with explicit runtime limits.
- [WeChat input-source crash](wechat-input-source-crash.md) — matching system NULL-release failures, fresh Chatterfly helper decompilation, Doubao lifecycle comparison, Apple contracts and prioritized repair recommendations.
- [Input-source cache prevention](input-source-cache-prevention.md) — cross-application exposure, matched cache/control reproduction, supported API limits and prioritized lifecycle prevention with native acceptance gates.
- [Competitor input-source refresh](competitor-input-source-refresh.md) — Doubao and Chatterfly registration/selection service restarts, exact caller guards and retry reachability, with cache-remediation limits.
- [IME lifecycle repair proposal](ime-lifecycle-repair-plan.md) — concrete pending-removal and atomic-update transactions, rollback/recovery gates and owned macOS directory-exchange validation.
- [Input menu items](input-menu.md) — Chatterfly/Doubao input-menu items compared with Arion, the implemented alignment and the deferred toggles.
- [Shift language switching](shift-language-switch.md) — Doubao/Chatterfly short-press release gates, chord suppression and composition commit paths, with static evidence limits.
- [IMK typing alignment](imk-typing-alignment.md) — focused competitor paging/editing/mode-switch evidence, Arion composition ownership fixes, full-keyboard implementation gates and restored post-profile diagnostics with system acceptance deferred.
- [Doubao keyboard commit chain](doubao-keyboard-commit.md) — byte-verified keyboard selection-to-host insertion and partial reconversion paths, with exact artifact identity and static evidence limits.
- [Competitor text insertion](competitor-text-insertion.md) — current Chatterfly IMK/system-paste policy and readback, matching Doubao commit transactions, evidence limits and shortcut-mode implications.
- [Competitor onboarding](competitor-onboarding.md) — first-use flows, evidence limits and Arion verified-practice implementation gaps.
- [Competitor contextual rewriting](competitor-rewrite.md) — focused Typeless/Doubao refinement and organization evidence.
- [Competitor disfluency handling](competitor-disfluency.md) — Doubao decoder span annotations, organization request construction, Chatterfly short-input/result policies and language limits.
- [Disfluency implementation benchmark](../../benchmarks/results/disfluency-alignment-v2/README.md) — local deletion spans, independent organization, matched native workers and explicit remaining failures.
- [Rewrite reliability](rewrite-reliability.md) — Doubao/Chatterfly alignment, reproduced source-delivery defects, adopted fixes and remaining generation failures.
- [Scene rewrite alignment](scene-rewrite-alignment.md) — transformation contract, mechanical versus semantic checks, frozen paired experiments and batch/release acceptance gates.
- [Scene rewrite alignment experiment](../../benchmarks/results/scene-rewrite-alignment-v1/README.md) — completed paired generation/delivery-policy study, verifier calibration, failed adoption gates and reproducible raw evidence.
- [Competitor cloud synchronization](competitor-cloud-sync.md) — account vocabulary, selected preferences, optional history, clipboard transfer and cloud-processing boundaries.
- [Keyboard and voice vocabulary sharing](keyboard-voice-lexicon.md) — Doubao user-word hotwords, Chatterfly input-history ASR context and personal voice terms in typing candidates, with exact version and static limits.
- [Keyboard and voice vocabulary practice/evaluation plan](keyboard-voice-lexicon-plan.md) — proposed shared personal terms, encrypted native bridge, causal ablations, held-out enrollment, performance/load/endurance tests, native acceptance and adoption gates.
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

- [Focused scene prompts v2](../../benchmarks/results/scene-rewrite-prompt-v2/README.md) — optimized language/scene-specific templates, 144 reviewed outputs, improved structure and failed fidelity adoption gate.

- [Context-supported correction and structural editing](../../benchmarks/results/context-repair-v1/README.md) — user-selected style anchors, 88 inspected local generations, supported ASR-text repair and corrupted/unrelated OCR controls.

- [Prompt optimization and complete text-validator removal](../../benchmarks/results/rewrite-no-validator-v1/README.md) — 128 reviewed proposals and 384 full policy deliveries; better organization, OCR regression and new bad deliveries when validation is removed.

- [OCR spelling alignment prompt correction](../../benchmarks/results/ocr-alignment-v2/README.md) — 128 reviewed generations plus 12 parity probes; improved supported repairs, remaining context-replacement/ambiguity failures and unchanged production defaults.

## Cleanup-aligned rewrite diagnostics

- [rewrite-private-spans-v1](../../benchmarks/reproduction/rewrite-private-spans-v1/README.md)
- [rewrite-natural-text-review-v1](../../benchmarks/reproduction/rewrite-natural-text-review-v1/README.md)

## Input-object and2B artifact diagnostics

- [rewrite-data-object-v1](../../benchmarks/reproduction/rewrite-data-object-v1/README.md)
- [rewrite-2b-precision-v1](../../benchmarks/reproduction/rewrite-2b-precision-v1/README.md)

- [User-aligned spelling review](../../benchmarks/reproduction/rewrite-spelling-alignment-v1/README.md) and [bounded edit study](../../benchmarks/reproduction/rewrite-bounded-edit-v1/README.md):confirmed normalization policy, immutable original scores and explicit intermediate/final evidence.

- [Bounded-edit v2](../../benchmarks/reproduction/rewrite-bounded-edit-v2/README.md), [formatter v3](../../benchmarks/reproduction/rewrite-formatter-v3/README.md) and [Metal preflight](../../benchmarks/reproduction/rewrite-metal-preflight-v1/README.md):visible-source formatting, explicit comparison criteria and actual macOS provider evidence.

- [Complete-reference message separation](../../benchmarks/reproduction/rewrite-selector-reference-turn-v1/README.md):404 actual choices; full OCR preserved in separate reference turns; both variants rejected after complete review.

- [Unverified unique-label exposure](../../benchmarks/reproduction/rewrite-selector-unique-label-v1/README.md):404 actual choices; only the unique-status field is omitted; all4 changed choices regress, both variants rejected.

- [Added semantic contrast demonstrations](../../benchmarks/reproduction/rewrite-selector-contrast-demo-v1/README.md):404 actual choices after zero-inference native token-budget checks; one improved and five regressed choices, both variants rejected.

- [MiniCPM5-1B comparison](../../benchmarks/reproduction/rewrite-selector-minicpm-v1/README.md):202 unchanged-request choices plus12 native responses; both configurations rejected. One interrupted native-log completion is retained separately, with no fabricated response credit.

- [MiniCPM F16 export comparison](../../benchmarks/reproduction/rewrite-selector-minicpm-f16-v1/README.md):202 unchanged-request choices plus12 native responses; more required repairs but15/17 harmful choices, both configurations rejected.
- [Youtu1.96B development comparison](../../benchmarks/reproduction/rewrite-selector-youtu-v1/README.md): native compatibility passes, semantic gate fails; research-only custom license.
- [LFM2.5 actual1.17B comparison](../../benchmarks/reproduction/rewrite-selector-lfm25-v1/README.md):47/47 required repairs but22 harmful choices per arm; both rejected.

- [Existing ten-example order](../../benchmarks/reproduction/rewrite-selector-demo-order-v1/README.md):404 actual choices; moving keep examples last reduces harmful edits but loses supported repairs.3 improvements,23 regressions,1 allowed change; both treatments rejected.

- [Native d1 decision packaging](../../benchmarks/reproduction/rewrite-d1-evidence-v1/README.md):72 actual typed responses over six sources/six layouts, all reviewed. Native96/45 piece clipping identified; full-policy correction remains3/6 per backend. No final bodies or production adoption.

- [Chatterfly Chinese/English caret hint](chatterfly-language-hint.md) — traced hint window, default animation dispatch, timing and Arion visual differences.
- [Rime installation and input-method updates](rime-input-method-updates.md) — pinned official Weasel/Squirrel installer comparison and Arion's shared automatic maintenance with platform-specific protected publication.

- [Native Qwen3-ASR 1.7B and accessibility context](../../benchmarks/reproduction/qwen17-accessibility-v1/README.md) — paired production ASR development comparison, owned-window Accessibility versus OCR collection and downstream rewrite controls.

- [IME lifecycle acceptance](ime-lifecycle-acceptance.md) — implemented repair, automatic update policy and actual verification limits.
