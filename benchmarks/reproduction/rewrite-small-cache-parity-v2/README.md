# rewrite-small-cache-parity-v2

32 measured outputs and8 reviewed warmups. Tested cache parity16/16 and order parity16/16; cached p95≈1614ms and large order drift. No release or production activation.

This is a frozen local Qwen3.5-0.8B text-GGUF diagnostic, CPU4/GPU0, no thinking. No4B or older Qwen inference. Full raw output is retained without content validation or silent cleanup replacement. Scenario assignment is explicit in each row; automatic detection, natural ASR, independent blind ratings, native insertion and low-end hardware acceptance remain separate gates.

See `protocol.json`, `freeze.json`, the corresponding run manifest and complete report. All generated outputs, including declared warmups, were personally inspected by the assistant. Warmups are excluded from measured latency denominators. Repeated orders are not independent samples. Model/runtime binaries are separately provisioned by the pinned latest-small-model acquisition; portable report replay does not open them.

```sh
pnpm rewrite:cache-parity:report --out .workspace/rewrite-small-cache-parity-v2-replay
pnpm rewrite:cache-parity:test
pnpm rewrite:cache-parity:verify
```

Replay targets must be new. Historical failed runs are preserved and cannot supply a completed inference report. These commands use the unchanged portable evidence wrapper.

The wrapper reuses the immutable cache runner and sets only documented `LLAMA_ARG_UBATCH=1` before native startup. `run-environment.json` records the treatment before inference; logical batch capacity stays at its default. The earlier batch1 failure is preserved separately. Exact parity on this8-case subset does not prove general cache correctness. Forward and reverse timings drift substantially; cause is unproven. Cold prefix warmups and request time are reported separately, with no product stop-to-insertion claim.
