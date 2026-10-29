# Paper strategies for Qwen dictation pipelines

## Decision scope

Prioritize **streaming ASR, acoustic finalization, bounded approved vocabulary, and one local non-thinking editing proposal**. Investigate a task-trained small Qwen editor and overlap fixed-prefix computation with capture. This is an Arion research recommendation, not a checkpoint approval or a measured improvement from new training.

This review separates paper mechanisms, [executed Arion diagnostics](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/qwen-pipeline-diagnostic-v1/README.md`), and proposed experiments. It does not establish Typeless parity. [008](../../specs/008-evaluation.md) and [009](../../specs/009-contextual-rewrite.md) remain normative. Paper versions below are the versions inspected; third-party evaluations are not Qwen release guarantees. Paper scores use their own datasets, hardware and normalization.

## Qwen mechanisms and their limits

### Qwen3-ASR: contextual transcription, not a general editor

[Technical report, arXiv:2601.21337v1, §§2.1–2.4, 3, 4.5](https://arxiv.org/html/2601.21337v1): AuT supplies audio representations to a Qwen decoder. Dynamic attention windows support offline and streaming use. ASR post-training includes non-speech, streaming and contextual-bias examples; it deliberately suppresses natural-language instruction following. GSPO further trains recognition robustness.

Its streaming comparison uses **2-second chunks, five-token fallback and four unfixed trailing chunks**. Capture packet size, encoder attention window and revisable output are different quantities. This experiment does not validate a 160 ms Arion preview. Reported 92 ms average TTFT uses BF16/CUDA Graph/vLLM serving; it is not local stop-to-insertion latency.

The separate ForcedAligner predicts transcript timestamp slots non-autoregressively. It can propose timing annotations, but requires independent review; alignment cannot certify a transcript or establish intended-turn endings.

**Arion adaptation:** compare Qwen as an acoustic finalizer and, after a real streaming adapter exists, as a preview recognizer. Measure contextual transcription separately from disfluency/scene editing. The current Qwen diagnostic adapter is final-only.

### Qwen3 text: distill the editing task into a small student

[Technical report, arXiv:2505.09388v1, §§4.3–4.5, 4.7/Table 21](https://arxiv.org/html/2505.09388v1): lightweight checkpoints use strong-to-weak distillation. Off-policy training learns teacher responses; on-policy training aligns teacher and student logits on student-generated sequences through KL divergence. Thinking-mode fusion provides a trained non-thinking path.

The reported distillation efficiency comparison concerns math/code on an 8B model. It does not establish dictation fidelity or Arion training cost. The paper's thinking-budget intervention ends reasoning explicitly and then generates an answer; it is different from truncating all generation at a fixed token limit.

**Arion adaptation:** first train final edited text from reviewed teacher proposals; then consider on-policy logit distillation if the teacher, tokenizer and training infrastructure permit it. Sequence imitation alone is not reproduction of the paper's logit method. Compare non-thinking editing against explicitly bounded reasoning that retains an answer budget. Never deliver incomplete reasoning.

### Qwen2-Audio: direct audio conditioning is possible

[Technical report, arXiv:2407.10759v1, §2](https://arxiv.org/html/2407.10759v1): a Whisper-initialized encoder conditions a language model; natural-language task training, SFT and DPO support audio analysis and voice instructions. The model is approximately 8.2B total parameters, including its audio encoder.

**Arion adaptation:** use an audio-conditioned model as an offline research challenger for ambiguity that text-only correction cannot recover. Voice-command following requires a separate dictation contract: spoken text must not implicitly become a command. This report does not validate our streaming lifecycle, ONNX export or faithful scene rewriting.

### Qwen2.5-Omni and Qwen3-Omni: overlap input computation

[Qwen2.5-Omni, arXiv:2503.20215v1, §§2, 2.4](https://arxiv.org/html/2503.20215v1) introduces Thinker/Talker separation and block-wise input processing for incremental prefill. The useful input-method idea is computing available input during speech; speech synthesis is outside our dictation scope.

[Qwen3-Omni, arXiv:2509.17765v1, §§2.2, 2.5, Tables 1–2](https://arxiv.org/html/2509.17765v1) extends chunked/asynchronous prefill and uses AuT audio encoding. Its theoretical **234 ms first speech packet** is a speech-generation measure, not complete corrected text or insertion. A 30B-A3B Thinker activates roughly 3B parameters per token but retains a 30B backbone; active parameters do not determine resident weight memory.

**Arion adaptation:** fixed system-prefix staging is the immediately testable overlap technique. Direct audio-to-edited-text remains a separate research branch with multimodal export, memory, task-fidelity and native lifecycle gates. Removing Talker does not turn the Thinker into a small text checkpoint.

### GSPO: a later training option, not an inference accelerator

[Group Sequence Policy Optimization, arXiv:2507.18071v1, §4](https://arxiv.org/html/2507.18071v1) uses length-normalized sequence likelihood ratios and sequence-level clipping with group-relative rewards. Its training stability findings do not directly reduce runtime latency.

**Arion adaptation:** investigate reward-based editing only after reviewed SFT and a reliable verifier. Use human-reviewed semantic preference pairs plus protected-content checks. A reward based only on lexical constraint passes can favor fluent hallucinations or copied required strings; no such reward is approved here.

### Newer checkpoints are separate experiments

- [Qwen3-4B-Instruct-2507 official card](https://huggingface.co/Qwen/Qwen3-4B-Instruct-2507): a non-thinking-only checkpoint, distinct from the original Qwen3-4B measured here. Compare them after source/export parity; neither the update card nor general writing scores establish safer dictation.
- [Qwen3.5-4B official card](https://huggingface.co/Qwen/Qwen3.5-4B): specifies a hybrid Gated DeltaNet/attention layout. [Gated Delta Networks, arXiv:2412.06464v1, §3](https://arxiv.org/html/2412.06464v1) motivates efficient recurrent memory and hybrid attention. The card is not a Qwen3.5 technical report. Recurrent state, convolution state and attention caches need reset/append/parity verification; existing Qwen3 export success does not approve this architecture. Short dictation speed gains remain unmeasured.

## Related methods worth adapting

| Primary paper and inspected location | Mechanism and evidence boundary | Proposed Arion use |
| --- | --- | --- |
| [CoEdIT, Findings EMNLP 2023, §§3–6](https://aclanthology.org/2023.findings-emnlp.350/) | Task-specific instruction tuning for editing, including composed instructions; not a streaming speech model | Supervise faithful cleanup and scene formatting as explicit tasks; retain a no-edit control and independent intent review |
| [mEdIT, NAACL 2024, §4, Appendix B](https://aclanthology.org/2024.naacl-long.56/) | Multilingual editing; its GEC/simplification training reserves 20% identity pairs to discourage over-correction | Sweep identity-pair fraction on development data; include Chinese/English/mixed repairs and protected spans. The paper's fraction is not an Arion optimum |
| [FastCorrect, arXiv:2105.03842, §3](https://arxiv.org/abs/2105.03842), [FastCorrect 2, Findings EMNLP 2021, §3](https://aclanthology.org/2021.findings-emnlp.367/) | Edit alignment, predicted target-token durations and parallel non-autoregressive correction; the extension aligns real hypotheses using lexical/pronunciation similarity and predicts a source candidate | A trained fast acoustic-error editor is a separate challenger. These architectures cannot be obtained by changing a Qwen prompt and do not establish scene-editing ability |
| [CoVoGER, EMNLP 2025, §§3–6](https://aclanthology.org/2025.emnlp-main.320/) | Multilingual ASR/translation correction from real five-best hypotheses; evaluates Qwen2.5 with LoRA. §6 finds over-correction in zero-shot ASR, task interference and cases where mixed sampling loses to beam search | Compare copy, actual top-1, reranking and generative correction on identical candidates. Use ASR-only versus joint editing training as an explicit ablation. Its punctuation-sensitive TER differs from our CER/WER/MER |
| [DARAG / Failing Forward, Findings ACL 2025, §§3–4](https://aclanthology.org/2025.findings-acl.125/) | Generates training speech, decodes it through the ASR engine, and retrieves named entities for correction; error distributions and unseen entities affect transfer | Obtain training errors from our actual recognizers, retain held-out recognizer/domain tests, and retrieve approved terms. Synthetic spelling noise alone cannot reproduce acoustic errors |
| [MM-ContextASR, arXiv:2609.19765v1, §§3–5/Table 1](https://arxiv.org/html/2609.19765v1) | Tests Qwen3-Omni with fixed current speech and five histories: irrelevant, implicit, explicit, corrected error and repeated error. Condition-matched SFT uses separate adapters; base models can deteriorate on wrong history. A recent preprint, not an Arion replication | Adopt its matched-history evaluation design. Speech-history gains do not authorize audio retention or sync. Text-only and approved-memory controls come first; distinguish adapter-training effects from inference-only context effects |
| [Ontology memory correction, arXiv:2606.13464v2, §§3–4, Appendix A](https://arxiv.org/html/2606.13464v2) | Retrieves entities, aliases and confusion forms from causal history. Chinese RAMC-Corr results improve over direct correction in 9/10 settings, but several still lose to raw ASR. Qwen3.5-4B zero-shot corpus CER is 27.94 raw, 27.61 direct, 27.36 with memory; no universal gain. Recent preprint | Compare bounded structured retrieval against prose history and raw ASR. Measure edit rate, improve/worsen among edits, and user-visible latency. Unlike its automatic memory updates, Arion persists/syncs only approved entries |
| [Agentic-GER, arXiv:2609.29428v1, §§2–4](https://arxiv.org/html/2609.29428v1) | Selects suspicious terms using full-transcript context, re-transcribes audio spans with Qwen3-ASR-1.7B, and iterates decisions. Server/API models and long-form evaluations; terminology gains can coexist with overall-error increases. Recent preprint | Test one bounded local acoustic-evidence request against text-only editing; no unbounded loop or future audio during causal replay. Treat full-utterance context as an offline finalization control, not streaming evidence |
| [Whisper-Streaming, arXiv:2307.14743v2, §§2–3](https://arxiv.org/html/2307.14743v2) | LocalAgreement-2 exposes common prefixes of consecutive hypotheses while re-decoding available audio | Compare stable-prefix preview policies and their delay/erasure tradeoff. Agreement is not correctness and must not authorize target commits; later spoken repairs can invalidate an earlier phrase |
| [Speculative decoding, ICML 2023, §§2–3](https://proceedings.mlr.press/v202/leviathan23a.html) | A draft model proposes tokens; target verification with exact acceptance/rejection preserves the target distribution under the algorithm's assumptions | A later decoding-speed experiment after exact provider support is proven. Measure draft acceptance, verification cost, memory and ASR contention. Fixed-prefix prefill has no draft tokens and is a different technique |

## Fit to the executed diagnostics

The public [development report](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/qwen-pipeline-diagnostic-v1/README.md`) supports these engineering questions, not population conclusions:

- Original 4B intent and two-pass formatting achieved 17/24 and 19/24 lexical constraint passes on authored text. On 56 frozen actual-ASR inputs both achieved 15/56; the second pass had materially higher generation latency. Joint task training is a research alternative to serial calls, but may introduce task interference.
- The dedicated prose-context/terminology cases failed even with 4B. Test retrieval provenance and actual recognizer bias before attributing this to insufficient model size. Large language-model plausibility cannot restore acoustic evidence that is absent from its input.
- Reasoning under a shared 256-token ceiling produced incomplete deliveries. This tests constrained delivery, not reasoning capability under the paper's budget intervention.
- Six matched prefix-staging cases retained identical complete prompts/outputs; parent-clock ASR-result-to-candidate p95 changed from 3.797 s to 2.190 s. Computation moved into speaking time. Native stop-to-insertion, short-utterance effectiveness and contention remain unverified.

## Proposed ablations and dependency order

**All rows below are proposed and unexecuted.** Existing diagnostics are controls, not newly measured paper reproductions. New branches need explicit manifests/drivers; the current runner cannot execute them by naming a paper. Use reviewed development data for selection and freeze a fresh held-out test before confirmatory comparisons.

| ID / priority | Hypothesis and paired treatments | Fixed factors and required evidence |
| --- | --- | --- |
| PAP-00 / P0 | Source FP versus export FP versus INT8/INT4 isolates conversion and quantization damage | Same checkpoint/template/input; token/logit/stop and generated-text parity; calibration is training-only; approved artifact lineage and actual native teardown |
| PAP-01 / P0 | Better acoustics reduce downstream guesswork: current streaming + current finalizer versus Qwen finalizer; then actual Qwen streaming/offline | Same reviewed audio and editing policy; final-only adapters excluded from streaming partial metrics. Reproduce the paper's streaming setting first, then tune smaller windows on development data with declared support |
| PAP-02 / P0 | Memory needs selective evidence: none, current prose, approved deterministic aliases, bounded retrieved terms, actual ASR contextual bias | Same model/audio, approved term pool and scene; each treatment alone before combinations. Report retrieval misses/distractors; do not fabricate n-best or claim a missing recognizer bias API |
| PAP-03 / P0 | Wrong history is a separate failure mode | Same current audio and entity across the five MM-ContextASR history scenarios, plus no history. Same checkpoint for inference-only comparisons; adapter-matched training is a separate experiment. Historical entities grouped across splits |
| PAP-04 / P1 | Updated non-thinking checkpoints outperform the tested original | Original Qwen3-4B versus Instruct-2507; Qwen3.5 only after its architecture/export gate. Same bounded task and tokenizer-appropriate templates; generation limits, effective tokens and truncation visible |
| PAP-05 / P1 | Small task-trained editors outperform generic prompting | Untuned 0.6B/1.7B, reviewed human-pair SFT, teacher-response SFT, then optional on-policy distillation. Same reviewed training inputs, separate data/training compute accounts, at least three training seeds; count teachers and supervision separately |
| PAP-06 / P1 | Copy examples and evidence restriction reduce unnecessary edits | Development-only identity-pair sweep 0/10/20/40%, verified entity evidence on/off, cleanup-only versus scene-conditioned joint training. Same base/data volume/steps; fixed text and token budget. Preserve per-task/language strata |
| PAP-07 / P1 | One model call can clean intent and format without extra facts | One combined task-trained editor, intent-only editor, measured two-call baseline. Same raw/finalized inputs and approved scene; separate intent failures from formatting failures and measure whole pipeline latency |
| PAP-08 / P1 | Early fixed prefill improves stop latency without delaying speech | Serial generation versus fixed-prefix staging; unchanged complete prompts and fresh request state. Short/long utterances, model cold/warm, real ASR queue/drop/preview events, device power and peak resident memory |
| PAP-09 / P2 | Acoustic evidence resolves text ambiguity | Text-only editor versus one suspicious-span re-transcription request versus direct audio-conditioned editor. Identical available audio; total call/compute/latency budgets. Log selection and recovery coverage; gold alignments cannot enter live requests |
| PAP-10 / P2 | Parallel correction or verified speculative decoding beats serial Qwen generation | A trained FastCorrect-style editor, or draft/target decoding after runtime capability checks. These are separate architectures/decoding experiments; include training/extra residency, accepted draft tokens and contention |
| PAP-11 / P2 | Preference/RL tuning improves fidelity beyond SFT | Reviewed SFT versus reviewed-preference optimization, then optionally GSPO. Same start checkpoint and data accounting; held-out independent semantic judges, copy controls and reward-hacking challenges |

Execute PAP-00 and reviewed data work first. PAP-01–03 establish acoustic/context failure sources. PAP-04–07 require trustworthy source checkpoints and separate training provenance. PAP-08 can proceed on the checked existing checkpoint with native measurement work. PAP-09–11 depend on bounded interfaces and measured unresolved failure modes. The table specifies research dependencies, not a production commitment to every branch.

## Training-data contract for PAP-05–07

Use three separate targets: the faithful spoken transcript, reviewed intended clean text, and optional reviewed scene-formatted text. Spoken self-repair may legitimately change the cleaned text relative to a verbatim transcript; do not use ASR WER to score scene rewriting. Annotate actor/action/object, entities and values, negation, uncertainty, time and which repair supersedes which span. These annotations and acceptable alternatives stay evaluator-only at inference.

Training examples should include correct copy cases, real recognizer errors, explicit repairs, hesitation versus meaningful hedging, repeated emphasis versus stuttering, short utterances, code/URLs, multilingual text, conflicting memory and spoken/quoted attacks. Generate teacher proposals only from training data using an explicitly provisioned local or owned offline environment. Independently review them before training; a teacher is not a gold label. No private content is routed to cloud inference by this plan.

For synthetic augmentation, generate training-only text/audio and run the same actual ASR engines used by the experiment. Retain TTS speaker/style and source-recognizer provenance. Split by speaker/session, entity family and source document before augmentation; descendants inherit their split. Deduplicate acoustic/text variants. Do not train on the existing 24 diagnostic challenges and then report them as a held-out test. Keep natural speech as an independently reviewed test stratum.

Record base/teacher revisions, tokenizer, task-template hashes, identity fraction, LoRA/optimizer settings, training seeds/steps and all teacher/filter/reviewer decisions. Compare equal supervised examples and report training compute; teacher-response SFT, human SFT and logit distillation have different supervision. Merge/exported adapters require source-to-export parity again. Loss improvement alone cannot establish useful correction.

## Complete evaluation chain

1. **Provenance:** pin source/checkpoint/export/runtime/data hashes and reviewed access terms. Import CoVoGER or MM-ContextASR only after their repository/data permissions and split provenance are verified; this literature review has not provisioned them. RAMC-derived data inherits its source restrictions.
2. **Acoustics:** real-time replay with only elapsed audio, independent onset/end/word labels, per-language raw and finalized CER/WER/MER, silence false emissions, tails, intended-turn endpoint errors, first partial and stable-correct-unit delay. Keep offline full-context challengers separately labeled.
3. **Editing:** compare copy/rules against candidates on gold text and identical frozen actual-ASR text; report source uncertainty. Measure no-edit accuracy, repair resolution, exact protected spans, contradiction/unsupported fact judgments and scene suitability. Lexical proxies and automated model judgments do not replace two independent blind raters.
4. **Memory/ranking:** score retrieval recall, entity-biased and non-biased errors with explicit alignment denominators, distractor-induced errors, wrong/stale-memory sensitivity and edit coverage. Only real candidate sets permit oracle/top-1/MRR. Paper-specific TER/B-CER/B-WER are separate outputs, not aliases for current proxy metrics.
5. **Streaming/native delivery:** record engine tokens and actual rendered preview separately; speech-end-to-review and stop-to-insert on one reconciled clock. Measure erasure, queue age/drops, cold/warm loading, native release, RAM/device allocations, power/thermal state and ASR contention. Cancellation/focus/mode/source changes invalidate proposals and caches. Preview stability never authorizes commits.
6. **Comparison with Typeless:** replay the same authorized natural/synthetic fixtures in an owned editor with identical scene/context exposure where configurable. Record observed competitor output and timing; blind language/intent/style review and editing effort complete the comparison. Unknown competitor internals remain unknown.
7. **Selection:** paired speaker/session-cluster intervals, retained failures and full latency tails; preregister primary metrics, fidelity non-inferiority margins and multiplicity/sample-size rationale before the fresh test. Select a latency/fidelity/coverage frontier using held-out evidence, with human inter-rater agreement and adjudication recorded.

For transcription editing, report `edit_rate = edited / all`, `improve_at_edit = lower_distance / edited`, and `worsen_at_edit = higher_distance / edited`, with empty edited denominators explicitly unavailable. Count changed outputs whose distance ties separately. These are surface-error diagnostics; human harm/intent outcomes remain independent. A system that copies everything can avoid harm while failing to resolve repairs, so always report benefit and coverage together.

The self-hosted sync boundary remains approved vocabulary/preferences. Per-session hypotheses and model-created ontology entries are temporary evidence until user approval; they do not become authoritative memory or synchronized raw audio. A failed validation surfaces an invalid proposal with the original available for review; it must not silently claim a successful rewrite.

## Execution entry points

Use existing [Make/pnpm diagnostic commands](qwen-pipeline-diagnostics.md) only for supported measured branches. Use the [strict rewrite driver protocol](rewrite-evaluation.md) for native acceptance. Keep weights, fixtures, teacher outputs and traces private in `.workspace/`; publish reviewed aggregates only. Add new commands through portable Node entry points and the Makefile when their actual drivers exist. Reading these papers adds no runtime implementation, training result or model provisioning.
