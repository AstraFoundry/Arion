---
title: "AI-native product work: building a product like running an experiment"
description: When models and coding agents make code and plausible results cheap, the scarce thing is knowing whether a change helped. How Arion uses benchmarks, ablations and written rules to decide what ships.
date: 2026-10-27
category: engineering
tags: [methodology, benchmarks, ablation, evaluation, ai agents]
---

Arion is a voice input method. You hold a key, speak Chinese, English or both, and text appears where your cursor is. Speech recognition, the optional polishing model and the Pinyin keyboard all run on your computer. Much of the code in the repository is written by AI coding agents, in pull requests that people review.

That combination changes what is hard. Writing a new prompt, swapping a model or adding a context source now takes minutes. Each change also comes with a convincing demo. What does not get cheaper is finding out whether the change made dictation better for the next thousand sentences, and whether it quietly made some of them worse.

So we try to treat product questions the way a lab treats research questions. Each one gets a hypothesis, a fixed test set, a baseline, a recorded result, and a decision written down where the next person (or agent) will find it. This post describes how that works in practice, with the numbers from our own benchmark reports, including the ones that went against us.

## Write the benchmark before the feature

The clearest example is polishing: a local language model that tidies the recognized text and applies spoken self-corrections ("Wednesday, no wait, Thursday") before it is inserted.

Before tuning the prompt, we wrote two sets of cases. A development set of 24 was for iterating on the prompt. A held-out set of 30 was written and hashed before any prompt tuning, and used only to compare finished candidates.[^live] Every candidate ran through the production worker and the production validator on the same machine. The question was not "is the output good?" but "how many held-out cases end up correct, how many get worse, and how long does the user wait after they stop speaking?"

| Model | Held-out passes (source: 12 of 30) | Worsened | Final rewrite p50 / p95 |
| --- | --- | --- | --- |
| Qwen3 0.6B | 24 | 0 | 0.18 / 0.36 s |
| **Qwen3 1.7B (shipped)** | **27** | **0** | 0.50 / 1.67 s |
| Qwen3 4B | 28 | 0 | 0.89 / 2.15 s |

The table made the decision. 4B buys one more case at roughly twice the latency and memory. 0.6B is fastest, but it delivers three fewer cases, and its harmful edits (dropped clauses and values) were stopped only by the validator. We shipped 1.7B.

The same report also changed the pipeline around the model. Finalizing recognition one segment at a time cut stop latency slightly, but it raised recognition error at segment boundaries. So the shipped pipeline decodes the whole utterance once after you stop. Nobody would have guessed that from a demo.

## Change one thing at a time

A benchmark tells you whether a bundle of changes helped. An ablation tells you which part of the bundle did the work. We use ablations whenever a feature has several inputs that could plausibly matter.

**Context for polishing.** Polishing can read three kinds of context: text around the cursor, text on screen, and your recent dictations. The intuitive design turns all three on. The ablation tested each source alone and then all together, on 30 held-out cases that include 7 negatives: distracting windows, injected instructions, similar names and unrelated history.[^context]

- **Screen text was the useful source.** It added 5–6 passes on the 4B model, mostly names and terms visible in the window.
- **Cursor text did not help** in this set, and history helped by one or two cases.
- **Model size decided everything.** The 1.7B model, our default, scored 9 of 30 in every context mode; it ignores context. The 4B model went from 9 to 16.

That last finding is not one we wanted. It means context is only worth its cost with a larger model, and the ablation is the reason we can say so plainly instead of shipping a feature that does nothing on the default setting.

**Phonetic evidence for correction.** A second study asked whether a text model can fix recognition errors when you give it pronunciation evidence.[^phonetic] Six correction conditions ran on all 154 test cases (924 measured generations), next to an acoustic-only second pass as a control.

| Condition | Mandarin CER | English WER | Mixed MER |
| --- | --- | --- | --- |
| Raw streaming recognition | 8.17% | 7.63% | 36.39% |
| Offline acoustic second pass | **5.82%** | **3.97%** | **22.10%** |
| Text-only correction | 8.42% | 7.53% | 36.39% |
| Text + recognizer token scores | 85.77% | 39.27% | 231.27% |

The control won. Re-decoding the audio helped; asking a text model to edit the transcript changed little; and passing raw confidence scores to an unadapted model made it repeat itself and copy demonstrations. Error rates above 100% are reported as measured, not clamped. Without the control row, a modest-looking text-correction result could easily have been read as progress.

## Keep the failures in the record

An experiment is only as honest as what it is allowed to throw away. Our rules say: failed and cancelled runs keep their status, incomplete runs cannot produce a completed report, and no failed model proposal is replaced by its source to make a table look cleaner.[^eval] Some of the most useful entries in the archive are failures.

- **The 张伟 case.** In the first context pass, the 4B model replaced a dictation about 张伟 with an on-screen message from 张巍, and the validator accepted it. That one held-out case led to a cap on how much credit context spellings can earn, so context can no longer stand in for most of a dictation. The fix is recorded next to the case that caused it.
- **A reproduction that did not reproduce.** We reproduced a published LLM reranker for the Rime input method on real candidates. Top-1 accuracy fell from 56 to 45 of 100, with 3 rescues and 14 breakages, and real model calls took about 289 ms rather than the roughly 1 ms claimed.[^rime] We did not adopt it, and the report says why.
- **Contention, reported.** One audio fixture failed with an explicit `AUDIO_DISCONTINUITY`: three inference thread pools competed for the CPU and the capture queue overflowed. The run counts it as a failure; it is not hidden by a retry.
- **Cancelled is a status.** When a 4B phonetic run was stopped partway, its two completed generations stayed in the archive marked `cancelled_by_user`, and the report uses only the complete 1.7B checkpoint.

Every report also opens with its limits: one developer authored the cases, speech is synthesized, scores are exact-string proxies, the host was not isolated. Those sentences are part of the result. They tell the reader which claims the numbers can carry.

## Make the evidence reproducible

Results are only useful if someone else can check them. Benchmark corpora, raw per-case outputs, warmups, failed attempts and the exact scripts and prompts live in the repository under `benchmarks/`, with byte-level SHA-256 identities: 1,062 files and about 98 MB of evidence, including the recorded audio.[^archive] Model weights and personal recordings stay out. The archive can be checked and its reports rebuilt without loading a model:

```sh
pnpm bench:archive:verify
pnpm bench:archive:report --out .workspace/archive-report-repeat
```

The evaluation protocol fixes the details that otherwise drift between runs. These include one text normalization, corpus error rates as total errors over total reference units, and paired speaker-cluster bootstrap intervals. Test references never enter inference, memory or context; memory learned from test data is rejected outright.[^method] Confidence intervals are labelled exploratory, and the protocol says plainly that no significance claim follows from them.

This discipline pays off when results are good, too. When context ranking for the Pinyin keyboard lifted later-occurrence top-1 from 32.2% to 88.7% on 50 homophone pairs, with no pair regressing, the same report also showed where typing latency went. A sampling profile found that re-ordering the candidate window after every commit dominated letter-key cost. After the fix, letter-key p95 is 0.81 ms.[^imk] Quality and latency are measured in the same run, so one cannot quietly be traded for the other.

## Turn results into rules

Measurements decide one question at a time. To make them stick, each decision becomes a written rule.

- **Specifications** in `specs/` state what the product must do, each with acceptance IDs such as `EVAL-A04` ("cross-split speaker/audio/session or test-derived memory is rejected"). A feature is not done until its scenarios pass on the real OS.
- **Decision records** in `docs/decisions/` capture why. There are 28 so far, and they are superseded rather than deleted. Polishing shows the chain. ADR 008 required review before insertion. ADR 011 replaced that with live previews and automatic insertion once the benchmark and validator supported it. ADR 023 removed the mid-speech preview, because the final pass discarded it and it never changed what was inserted. Each step links to the evidence that justified it.
- **Agent instructions** in `CLAUDE.md` and `AGENTS.md` turn the lessons into constraints that coding agents read before they write code. Examples: "never substitute mock output for inference evidence"; no fallback logic that hides a problem you have not understood; read the spec before implementing a feature; update the docs in the same change set as the code.

The last point is where the AI-native part matters most. An agent will happily produce a plausible table, a passing synthetic test or a defensive default that makes an error disappear. The rules are written so that the cheapest path for an agent is also the honest one. If a driver or model is missing, the run fails visibly, and a missing capability is never reported as a passing test.

## What we have not proven

The method only works if we are as specific about the gaps as about the wins. Today:

- Most corpora are authored by one developer, and much of the speech is synthesized. Exposed test sets remain useful regression data, but they cannot become a fresh test set again.
- Rewriting quality is measured with exact-string proxies. Blind human review of meaning, style and hallucination is an open release gate.
- Most numbers come from one Apple M4 machine. Windows paths are often compile-checked, not measured.

<aside>

**This post's own limits.** Every figure above comes from a benchmark report in the repository; follow the footnotes to see the corpus, the raw outputs and the stated limits of each one.

</aside>

## What we would tell another team

If you build with models and agents, the code is no longer the bottleneck. Decide what "better" means before you build, and freeze a held-out set. Change one factor at a time and keep a control. Record failures as carefully as successes, write the decision down where the next agent will read it, and supersede old decisions instead of erasing them. None of this makes a product good on its own. It does make it much harder to fool yourself, which is most of the job.

[^live]: [Live rewrite pipeline results (v1)](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/results/live-rewrite-v1/README.md) and [ADR 011](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/011-live-local-rewriting.md).

[^context]: [Rewrite context ablation (v1)](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/results/context-ablation-v1/README.md) and [ADR 013](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/013-rewrite-context-on-device.md).

[^phonetic]: [Phonetic evidence ablation](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/results/phonetic-asr-1.7b-v2/README.md).

[^eval]: [Specification 008: Evaluation and automation](https://github.com/AstraFoundry/Arion/blob/main/specs/008-evaluation.md).

[^rime]: [Rime LLM reranker reproduction](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/results/rime-llm-reranker-v1/README.md).

[^archive]: [Benchmark research archive](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/migration/README.md).

[^method]: [Benchmark methodology](https://github.com/AstraFoundry/Arion/blob/main/docs/research/benchmark-methodology.md).

[^imk]: [Context ranking results](https://github.com/AstraFoundry/Arion/blob/main/benchmarks/results/imk-context-v1/README.md).
