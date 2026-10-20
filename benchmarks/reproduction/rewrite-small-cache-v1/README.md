# rewrite-small-cache-v1

32 measured outputs and8 reviewed warmups. Actual prefix hits and faster requests, but exact cache parity14/16; no adoption.

This is a frozen local Qwen3.5-0.8B text-GGUF diagnostic, CPU4/GPU0, no thinking. No4B or older Qwen inference. Full raw output is retained without content validation or silent cleanup replacement. Scenario assignment is explicit in each row; automatic detection, natural ASR, independent blind ratings, native insertion and low-end hardware acceptance remain separate gates.

See `protocol.json`, `freeze.json`, the corresponding run manifest and complete report. All generated outputs, including declared warmups, were personally inspected by the assistant. Warmups are excluded from measured latency denominators. Repeated orders are not independent samples. Model/runtime binaries are separately provisioned by the pinned latest-small-model acquisition; portable report replay does not open them.

```sh
pnpm rewrite:small-cache:report --out .workspace/rewrite-small-cache-v1-replay
pnpm rewrite:small-cache:test
pnpm rewrite:small-cache:verify
```

Replay targets must be new. Historical failed runs are preserved and cannot supply a completed inference report. These commands use the unchanged portable evidence wrapper.
