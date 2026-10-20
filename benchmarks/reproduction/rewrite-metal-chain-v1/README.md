# Continuous Metal cleanup/selection/formatting diagnostic

The156 exposed raw sources are unchanged:36 synthetic scene regressions and120 historical public read-speech ASR texts. The current speech-spans-v2 source snapshot is compiled as a persistent Go JSONL worker; each case actually calls it before candidate preparation. Only18 sources have produced hypotheses and invoke the actual spelling selector. All156 invoke source-only language-profile-v2 formatting. The same uniform zero-hypothesis rule applies to both cohorts; no empty selector response is fabricated. No output validator, source fallback or model substitution is used. Grammar constrains selector choices to declared keep/replace options; natural formatting remains generative.

This composition reuses versioned payload builders, not historical selector outputs. Complete source/OCR enters the spelling decision; formatting sees the corrected source. Known-gold references and old outputs never enter model messages. All174 actual model outputs must be personally reviewed before a completed report can be built. Meaning uncertainty is distinct from both failure and usable prose.

```sh
pnpm rewrite:metal-chain-v1:report --out .workspace/rewrite-metal-chain-v1/replay-new
pnpm rewrite:metal-chain-v1:test
pnpm rewrite:metal-chain-v1:verify
```

Inference uses `run.py --out <new-directory> --cleanup-worker <pinned-binary>` after explicit local model/runtime provisioning. To build the cleanup executable, copy the archived source snapshot from rewrite-cleanup-copula-v1/sources/domain into a fresh module arion.local/core, copy cleanup.go as main.go, and build with the recorded Go toolchain. The private binary must match freeze.json; source replay with a different toolchain is not the original inference treatment. Worker sources, runtime/model hashes, host/build metadata, raw logs, all actual requests/responses, reviews and selection decisions are preserved in the run archive.

Continuous per-case monotonic timing includes Go cleanup IPC, candidate preparation, actual selection when needed and final format response. Model startup is separate. Four persistent model workers, cache disabled, no explicit warmup. These are not cold/warm strata, concurrent-ASR or native stop-to-insertion measurements. Maximum sampled RSS is neither peak nor total application memory. Actual M4/32GB is not minimum AppleSilicon16GB acceptance. No new ASR, OCR, independent ratings or held-out speakers are implied. Fresh24 remain ungenerated and production model/provider/prompt/validator remain unchanged.
