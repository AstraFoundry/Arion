# 009 Contextual local rewriting

## Status and scope

This spec defines an optional local LLM stage and its evaluation boundary. The repository implements its evaluation harness, not the ONNX LLM runtime or a production rewrite driver. Competitor artifact findings do not establish that Arion or a competitor executes this proposed pipeline.

Rewriting is separate from acoustic recognition, conservative correction and Laya candidate ranking. A ranker permutes existing producer candidates; a rewriter can generate a new text proposal. Baseline dictation remains available with rewriting disabled. macOS shortcut, Windows shortcut and optional IMK/TSF hosts share the portable request/result contract; host integration must be verified on the actual OS.

## Product pipeline

1. Obtain revisioned ASR text and conservative correction. Preserve their original text independently of any rewrite.
2. Capture a bounded, consent-gated context snapshot and approved memory. Select a scenario from explicit user choice and permitted application hints. Unknown/secure targets deny context acquisition before reading their values. Clipboard, screen text and conversation history require distinct opt-in capabilities; their existence in competitor symbols does not authorize Arion to collect them.
3. Build a versioned prompt with trusted rewrite instructions separated from untrusted transcript/context/memory. Pin template, tokenizer, checkpoint, ONNX graph, conversion, quantization and execution provider. User text may describe commands; it never grants tool execution, networking or target writes.
4. Run bounded non-thinking generation locally. Keep token streams and provisional rewrite previews separate from recognized drafts and owned IME composition. Partial-source speculation is optional: every new source revision cancels it, and it cannot become the final candidate by mistake.
5. Validate the final proposal against the selected mode, output budget and protected content. Verbatim mode forbids changes to code, punctuation, capitalization and spacing. Names, numbers, URLs, negation, modality and speaker intent need preservation; string checks alone cannot prove semantic preservation.
6. Present the original and proposal for review. Initial rewrite mode requires explicit candidate acceptance. A generated instruction, reasoning block, skill invocation or model refusal is never inserted as ordinary dictated text. Rejection, timeout, missing model and unsupported runtime are explicit outcomes; any choice to use the original is visible to the user.
7. Commit only through the existing once-only target capability after source identity, focus and mode checks. Source revision, cancellation, focus change or mode switch invalidates the proposal and owned composition. A late token, preview or callback cannot write to a target.

## Runtime requirements

- Work offline after explicit model provisioning; no cloud inference route or silent model substitution.
- Model residency and per-session KV state are different lifecycles. Warm sessions may reuse immutable weights, but private context/KV state cannot leak between independent fixtures or users. Cold evaluation releases native model allocations and cache state, then includes reload cost.
- Capture callbacks never execute LLM inference. ASR and LLM workers have bounded queues, cancellation and recorded scheduling/thread policies. Concurrent ASR quality and queue pressure must be evaluated before enabling speculative generation.
- Cancellation tokens alone do not prove safety. Check generation/session/source revision at native preview publication, IME mutation and final target commit.
- No checkpoint, latency target or acceleration provider becomes an accepted product default without actual-host measurements and reviewed model/export provenance. CPU is the comparison baseline; provider acceleration is a separate experiment.

## Evaluation requirements

Use [008 Evaluation](008-evaluation.md) for ASR and native product safety, plus the [rewrite methodology and protocol](../docs/research/rewrite-evaluation.md). The rewrite harness uses the persistent JSONL transport with explicit schema `arion-rewrite-v1`; existing ASR drivers must fail on unsupported rewrite operations.

Evaluate both reviewed gold-text inputs and frozen real-ASR inputs. Hold each source text constant across rewriting variants. Report these strata separately so ASR mistakes do not become alleged rewriting failures or improvements. End-to-end speech evaluation still includes actual audio, endpointing and source-revision changes in the native driver.

Factorial comparisons cover rule versus LLM, context, approved memory and scenario hint. A verbatim control exposes unsolicited changes. Separate FP and quantized artifacts with the same reviewed source checkpoint expose conversion/quantization loss; if lineage differs, label the comparison as model/precision rather than causal quantization. Fix prompt/tokenizer/output budgets/no-thinking decoding/provider/threads within a comparison. Report warm/cold residency and solo/concurrent-ASR strata independently.

Automated checks cover protected exact occurrences, declared constraints, unsupported numeral and attack-marker proxies, refusals, delivery coverage, native timing/resource traces and lifecycle invalidation. Meaning, style, fluency and unsupported factual claims require blind independent human review. A release cannot claim semantic success from reference overlap, embedding similarity or an LLM judge alone. Any native stale publication or target commit in the candidate benchmark invalidates the run. Existing output directories, incomplete coverage and changed traces cannot produce a completed report.

## Acceptance

| ID | Required evidence |
| --- | --- |
| RWT-A01 | Provisioned local rewriting works offline; neither audio nor context goes to an inference service |
| RWT-A02 | Session/generation/source revision matches; cancellation and revision invalidation discard tokens/proposals and prevent stale preview/write |
| RWT-A03 | Focus/mode changes invalidate shortcut and owned IME composition; final commit rechecks ownership exactly once |
| RWT-A04 | Secure/unknown target context is denied before reads; clipboard/screen/history use explicit bounded capabilities; prompt data cannot authorize commands |
| RWT-A05 | Verbatim/code and protected meaning constraints hold; original and rewrite remain distinct, user reviews generated proposals |
| RWT-A06 | Missing artifact, unsupported provider, rejection and timeout are explicit; no fabricated live result or silent cloud routing |
| RWT-A07 | Reviewed hashes, isolated splits, held-out labels, matched factorial controls, warm/cold/load strata and real token/native traces produce a reproducible checked report |
| RWT-A08 | Blind meaning/style/fluency/hallucination review meets declared independent-rater coverage; output coverage differences invalidate paired quality claims |

## Implementation order

P0: reviewed corpus and native rewrite driver; model/tokenizer/export parity; source-bound cancellation and preview; deterministic mode/constraint checks. P1: blind quality review and held-out budget selection; provider and ASR-contention comparisons; native shortcut/IME acceptance. P2: speculative partial rewriting and broader personalization, after stability and harm measurements justify them.
