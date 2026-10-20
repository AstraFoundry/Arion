# Full historical cleanup/selection/formatting regression

The complete [401-tuple source view](../../corpora/rewrite-history-v1/README.md) is actually run through current speech-spans-v2 Go cleanup, conditional bounded spelling selection and source-only script-routed formatting. Same pinned1.88B tensor model, runtime, Metal99/CPU4/ctx4096/greedy/cache/budgets. No priming in this quality-breadth study; it is not a latency comparison with the primed experiment. All465 actual outputs (64 selectors,401 final bodies) are personally reviewed and retained without validator filtering or source fallback. No stage protocol failures occurred. Per-source attempted/generated outcome records remain independently bound.

Synthetic281 and historical natural120 remain separate; these are previously exposed inputs, not new independent subjects. Canonical spelling utility, source-faithful retention of unresolved ASR, usable organization and true semantic errors are distinct. Same-letter readCash normalization is permitted; unsupported readCache remains wrong. One explicit not-twenty-seconds omission retains the correct final value and is flagged under the unchanged emphasis criterion. An initial definite-failure cancellation judgment was reconsidered as uncertain pragmatics, not promoted to a pass; original review/selection/report bytes and amendment provenance remain available.

```sh
pnpm rewrite:history-chain-v1:report --out .workspace/rewrite-history-chain-v1/replay-new
pnpm rewrite:history-chain-v1:test
pnpm rewrite:history-chain-v1:verify
```

`run.py --out <new-path> --cleanup-worker <pinned-binary>` requires the provisioned model/runtime and exact Go worker. Every raw model call and every source outcome is stored; a malformed selection records missing formatting explicitly and never invents a body. This captured run has complete401-body coverage. Model-input source provenance, full actual messages/context/edits, Go byte-span observations, stage logs, host metadata, review and nonselection decision are archived. Report replay needs no weights or private executable.

No new ASR/OCR/native insertion/concurrent ASR/independent human ratings or AppleSilicon16GB acceptance. CurrentM4/32GB is development evidence only. No4B/olderQwen/Granite/Q8/fresh24 inference or production model/provider/prompt/validator activation.
