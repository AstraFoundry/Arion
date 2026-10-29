---
title: Why we build our voice input method like an experiment
description: When models and coding agents make prototypes cheap, the scarce thing is knowing whether a change helped. Starting from three experiments that contradicted our intuition, here is how Arion uses held-out sets, ablations and written rules to decide what ships.
date: 2026-10-27
category: engineering
tags: [methodology, benchmarks, ablation, evaluation, ai agents]
image: /blog/building-like-an-experiment/cover.webp
imageAlt: A soft-focus photograph of pink flowers and green leaves
---

Suppose you are dictating a message about your colleague 张伟 (Zhāng Wěi). While you speak, a chat window happens to be open on your screen from someone else: 张巍 (Zhāng Wēi). Same Pinyin, different person.

To help our local voice input method spell names and technical terms accurately, we wired screen context into the polishing model. It ran OCR over the active window and passed the visible text to a local LLM. It took an afternoon to build, and on the handful of sentences we tried by hand, it corrected every name we tested.

If we had shipped based on that demo, the feature would have failed in two opposite ways at once.

On Qwen3 4B, the model saw 张巍's message on screen and replaced the user's dictation about 张伟 with it. Worse, our output validator accepted the replacement. And on Qwen3 1.7B, the model we ship as the default, turning on screen context, cursor text and dictation history scored 9 out of 30 on our held-out test set. Turning all context off also scored 9 out of 30.[^context]

![A chat window on screen, the dictation about 张伟, and the Qwen3 4B rewrite that replaced it](./figures/name-swap.en.webp "The negative case behind the validator change. Qwen3 4B copied the on-screen message from 张巍 over a dictation about 张伟, and validation accepted it. With context credit capped, the same proposal is rejected and the dictation is kept.")

The same feature altered the speaker's meaning on the larger model, and did nothing at all on the default one.

## Three experiments where data contradicted intuition

Arion is a local-first voice and Pinyin input method. You hold a key, speak Chinese, English or a mix of both, and speech recognition plus an optional local polishing model tidy the transcript (turning "Wednesday, no wait, Thursday" into "Thursday") before inserting it at your cursor.

Instead of judging context polishing by a few live tries, we froze a 30-case held-out set that included 7 negative cases: distracting windows, prompt injections on screen, near-homophone names and unrelated dictation history. Then we ran an ablation, testing each context source alone before combining them.[^context]

On this set, text around the cursor added 0 passes, and recent dictation history added 1 to 2. Screen text carried almost all the gain: paired with macOS Vision's `accurate` OCR (107 ms p50), it added 5 to 6 passes on the 4B model, mostly names and terms visible in the window. Switching to Vision's `fast` mode (30 ms p50) lost 2 of those cases because it cannot read Chinese.

Yet model size decided whether context helped at all. Our default 1.7B model scored 9 of 30 in every context mode, ignoring the extra input. Only the 4B model climbed from 9 to 16, at roughly twice the latency and memory.

![Dot chart of held-out passes with and without each context source, for Qwen3 4B and Qwen3 1.7B](./figures/context-ablation.en.webp "Held-out passes out of 30, without and with each context source. Screen text carries the gain on Qwen3 4B; the shipped 1.7B model barely moves with every source on. 4B rows use extracted terms; the 1.7B row is the first pass.")

That 张伟 case led us to cap how much validation credit context spellings can earn, so on-screen text can no longer replace the main body of a dictation. Switching from raw context sections to extracted terms also reduced 4B median latency from 1.84 s to 0.86 s. And because the ablation showed 1.7B ignoring context, we avoided enabling a default setting that spends CPU without improving accuracy.

A second study asked whether a text model can fix recognition errors when given pronunciation evidence.

We tested Pinyin, universal IPA phones decoded from the audio, and the recognizer's own token confidence scores across 154 test cases (924 measured generations on Qwen3 1.7B). Next to those six LLM conditions, we included a control row that used no LLM at all: an offline acoustic second pass over the recorded audio.[^phonetic]

| Condition | Mandarin CER | English WER | Mixed MER |
| --- | --- | --- | --- |
| Raw streaming recognition | 8.17% | 7.63% | 36.39% |
| **Offline acoustic second pass (control)** | **5.82%** | **3.97%** | **22.10%** |
| Text-only LLM correction (Qwen3 1.7B) | 8.42% | 7.53% | 36.39% |
| Text + recognizer token scores | 85.77% | 39.27% | 231.27% |
| Audio IPA phones only | 314.73% | 334.69% | 275.20% |

Error rates above 100% in the bottom rows are not a bug in the scoring script. Our evaluation rules report insertions as measured rather than clamping error rates at 100% or replacing a failed model output with its source transcript. When we passed raw log-softmax token scores to an unadapted 1.7B model, it began repeating words, echoing prompt field names, and copying few-shot demonstrations until it hit the 256-token output limit. It corrupted 26 of the 60 utterances that streaming recognition had transcribed without error. Even the plain text-only editor changed almost nothing, and on all 4 pure-silence test cases it emitted a pair of quotation marks instead of an empty string.

The control won across every metric. Re-decoding the audio with an offline acoustic model took 0.13 s at p50, lowered all three error rates, improved 60 utterances, and corrupted none of the originally correct ones. Without that control row, we could easily have spent weeks tuning prompts on a text model for a problem the acoustic pass already solved in a tenth of the time.

![Diverging bar chart of utterances worsened and improved for each correction condition, with correct utterances broken](./figures/phonetic.en.webp "Of 154 test utterances, how many each condition made worse or better than raw streaming recognition, and how many of the 60 originally correct ones it broke. Only the control with no LLM improves more than it harms.")

A third study reproduced an external design.

On the Pinyin keyboard side, a published LLM reranker for the Rime input method reported roughly 1 ms latency for reranking candidates with a local model. We reproduced the unchanged upstream Lua filter against 100 real native Rime candidate snapshots using Qwen3.5-0.8B on local oMLX.[^rime]

Top-1 accuracy fell from 56 of 100 to 45 of 100, with 3 rescues and 14 breakages. Because real Rime candidate lists mix full-word conversions with shorter prefix candidates that consume only part of the Pinyin, the prompt promoted single-character prefixes ahead of full words: `朋友` (friend) became `彭`, `学习` (study) became `学`, and `电脑` (computer) became `点`.

Where did the 1 ms latency figure come from? In our run, HTTP calls to the local model server took 288.85 ms at the median. Reading the upstream Lua code explained the difference: its timer called `os.clock()`, which measures *process CPU time*. While LuaSocket blocked waiting for the model server to respond, the Rime process was asleep, so its CPU clock barely moved (`0.787 ms` p50) while the user waited nearly a third of a second.

![Timeline of one rerank request: the Rime process asleep while the model server generates, with CPU time and wall time compared](./figures/clock.en.webp "One rerank request. os.clock() counts the Rime process's CPU time, which stops while it sleeps on the socket, so the log shows under a millisecond while the typist waits almost a third of a second.")

Inspecting the raw responses revealed a second defect. The Lua parser placed multibyte UTF-8 Chinese punctuation inside byte-level character classes (`[^...]`). Whenever an ordinary Chinese character contained a matching byte, the parser split the word in half. Given the context `婚礼上许下` ("vowed at the wedding") and Pinyin `shi yan`, the model returned the right candidate, `誓言` (vow). The parser truncated the bytes and left `实验` (experiment) on screen. Out of 89 real model responses, 38 failed to parse.[^rime]

![UTF-8 bytes of 誓言 and 、 showing a shared byte that splits the word, so Rime keeps showing 实验](./figures/bytes.en.webp "Why 誓言 became 实验. Lua bracket classes match bytes, so the 0x80 byte that 言 shares with 、 counts as a separator, and the parser drops the model's first choice.")

## When code and demos are cheap

Before we measured them systematically, all three ideas looked sound: testing a few windows by hand on the 4B model made screen context look ready to ship; inspecting individual text-correction examples without an acoustic control made prompt tuning look promising; and reading the `os.clock()` log showed sub-millisecond numbers in the terminal.

Working with AI coding agents makes this trap easier to fall into. Much of Arion's repository is written by coding agents and reviewed by engineers in pull requests. Writing a new prompt, swapping an ONNX model or adding a context source takes minutes, and every change comes with a convincing demo.

Coding agents also optimize for passing whatever check is in front of them. If you ask an agent to run an evaluation or fix a failing pipeline, the shortest path to a green check is often to fall back to the source text when the model output fails validation, retry a dropped audio buffer, or mock a missing native driver.

Code and demos got cheap, but finding out whether a change helps the next thousand dictations, or quietly hurts some of them, did not. To avoid fooling ourselves or being fooled by an agent's fallback logic, we treat product questions like lab experiments.

## How we run the experiments

### Write the benchmark before tuning the model

Consider how we selected the live polishing model. Before tuning the prompt, we split the cases into two sets: a 24-case development set for iteration, and a 30-case held-out set written and SHA-256 hashed beforehand, used only to compare finished candidates.[^live] Every candidate ran through the production worker and production validator on the same machine.

| Model | Held-out passes (unpolished baseline: 12 / 30) | Worsened | Final rewrite p50 / p95 |
| --- | --- | --- | --- |
| Qwen3 0.6B | 24 | 0 | 0.18 / 0.36 s |
| **Qwen3 1.7B (shipped)** | **27** | **0** | **0.50 / 1.67 s** |
| Qwen3 4B | 28 | 0 | 0.89 / 2.15 s |

The 4B model gained one case at roughly twice the latency and memory. The 0.6B model was fastest, but it missed three cases and its harmful edits (dropped clauses and numbers) were only stopped by the validator. We shipped 1.7B.

Measuring the end-to-end audio pipeline in the same report also changed the architecture around the model. Finalizing recognition one speech segment at a time lowered stop latency slightly, but raised recognition error at segment boundaries; and when streaming ASR, per-segment Qwen3-ASR and the 1.7B LLM ran concurrently, one fixture failed with an explicit `AUDIO_DISCONTINUITY` because three ONNX Runtime thread pools contended for the CPU and overflowed the capture queue.[^live]

Because our protocol forbids masking a failure with a retry, that error remained in the report and ruled out per-segment finalization. The shipped pipeline decodes the whole utterance once after you stop speaking. Later, the same evidence chain led to [ADR 023](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/023-remove-live-rewrite-preview.md), which removed mid-speech LLM previews after benchmark data showed the final pass always replaced them without changing what was inserted.

### Measure quality and latency in the same run

After ruling out the 289 ms LLM reranker for Pinyin, we built a small on-device context ranker instead: it learns only from your recent commits in the same application (within 120 seconds), and limits a single confirmation to a two-place boost so a one-off selection cannot take over top-1.[^imk]

Across 50 homophone pairs and 3 seeds, first-occurrence accuracy was unchanged (14%). On later occurrences, standard frequency learning alone scored 32.2% top-1 because homophones used in different contexts kept displacing one another. With the context ranker enabled, later-occurrence top-1 rose to 88.7% (MRR from `0.612` to `0.933`), with 47 pairs improving and 0 regressing.

The same run also measured keystroke latency. The ranker itself took 0.02 to 0.06 ms per key, yet letter-key p95 latency was 2.92 ms in the initial run. A sampling profile showed that macOS `orderFront` (re-ordering the candidate window in the window server after every commit) dominated letter-key cost. Keeping the candidate bar transparent and click-through while idle reduced letter-key p95 to 0.81 ms.[^imk]

### Keep failures in a 98 MB reproducible archive

Specification 008 (`specs/008-evaluation.md`) sets three rules: failed and cancelled runs keep their status, incomplete runs cannot produce a completed report, and no failed model output is replaced with its source to make a table look cleaner.[^eval]

When a Qwen3 4B phonetic run was stopped partway, its two completed generations stayed in the archive marked `cancelled_by_user`, and the report used only the complete 1.7B checkpoint.

Benchmark corpora, raw per-case outputs, warmups, failed attempts, recorded audio fixtures and exact prompts live in the repository under `benchmarks/`, identified by byte-level SHA-256 digests: 1,062 files and about 98 MB of evidence (model weights and personal recordings stay outside Git).[^archive] Anyone can verify the archive and rebuild the reports without loading a model:

```sh
pnpm --dir /path/to/Arion-evaluation bench:archive:verify
pnpm --dir /path/to/Arion-evaluation bench:archive:report --out .workspace/archive-report-repeat
```

The evaluation protocol fixes the statistical details that otherwise drift between runs: a single text normalization, corpus error rates computed as total errors over total reference units, and paired speaker-cluster bootstrap intervals. Test references never enter inference, memory or context, and memory learned from test data is rejected (`EVAL-A04`).[^method]

### Turn conclusions into written rules

A benchmark answers one question at a time. To make the conclusions stick, each decision is written down in three places.

Specifications in `specs/` state what the product must do, with acceptance IDs such as `EVAL-A04` and `LRW-A05`; a feature is not done until its scenarios pass on the real OS. Decision records in `docs/decisions/` capture why, and are superseded rather than deleted (28 so far): polishing went from manual review before insertion (ADR 008) to live previews and automatic insertion (ADR 011) to removing mid-speech previews (ADR 023), with each step linked to the evidence behind it. And agent instructions in `AGENTS.md` and `CLAUDE.md` turn those lessons into constraints that coding agents read before writing code: never substitute mock output for inference evidence, do not add fallback logic that hides an uninvestigated problem, and update docs in the same change set as the code.

If a driver or model is missing, the run fails visibly, and a missing capability is never reported as a passing test. For a coding agent, the easiest path to a green build is the honest fix.

## What we have not proven

Every benchmark report opens with its limits. Today three main gaps remain:

Most corpora are authored by one developer, and much of the speech is synthesized or read aloud. Exposed test sets remain useful regression data, but they cannot serve as a fresh blind test set again.

Rewriting quality is measured with exact-string proxies. Blind human review of meaning preservation, style and hallucination (`LRW-A06`) is an open release gate.

Most numbers come from one Apple M4 (32 GiB) machine. Windows paths are often compile-checked and unit-tested rather than measured end to end.

<aside>

**Checking the numbers in this post.** Every figure and failure case above comes from a benchmark report in the repository; follow the footnotes to inspect the corpora, raw outputs and stated limits of each run.

</aside>

## What we would tell another team

If you build with local models and coding agents, the code is no longer the bottleneck. Decide what "better" means before you build, and freeze a held-out set with a hash. Change one variable at a time and include a control that does not use the new model. Record failures as carefully as successes, write the decision down where the next agent will read it, and supersede old decisions instead of erasing them. None of this makes a product good on its own, but it makes it much harder to fool yourself.

[^live]: [Live rewrite pipeline results (v1)](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/results/live-rewrite-v1/README.md) and [ADR 011](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/011-live-local-rewriting.md).

[^context]: [Rewrite context ablation (v1)](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/results/context-ablation-v1/README.md) and [ADR 013](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/013-rewrite-context-on-device.md).

[^phonetic]: [Phonetic evidence ablation](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/results/phonetic-asr-1.7b-v2/README.md).

[^eval]: [Specification 008: Evaluation and automation](https://github.com/AstraFoundry/Arion/blob/main/specs/008-evaluation.md).

[^rime]: [Rime LLM reranker reproduction](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/results/rime-llm-reranker-v1/README.md).

[^archive]: [Benchmark research archive](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/migration/README.md).

[^method]: [Benchmark methodology](https://github.com/AstraFoundry/Arion/blob/main/docs/research/benchmark-methodology.md).

[^imk]: [Context ranking results](https://github.com/AstraFoundry/Arion/blob/43586d3277719eec7ebdfcaa377fbfd4b649bd81/benchmarks/results/imk-context-v1/README.md).
