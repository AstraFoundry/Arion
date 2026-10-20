# rewrite-language-profile-v1

156 formatter outputs: development 36 faithful/usable; natural 117 faithful, 2 unfaithful, 1 uncertain, 73 usable, 118 preserve language.

All complete outputs were personally reviewed. Raw responses are preserved without validator rejection or source fallback. Natural sources are the same120 historical ASR text inputs across three recipes, not new audio recognition, independent subjects or spontaneous dictation. Development and natural scores remain separate.

```sh
pnpm rewrite:language-profile-v1:report --out .workspace/rewrite-language-profile-v1/replay-new
pnpm rewrite:language-profile-v1:test
pnpm rewrite:language-profile-v1:verify
```

Run requires the frozen model and runtime in private storage and a fresh output directory; inspect `run.py --help` for required arguments. Offline report replay does not load model weights. [Raw run](../../runs/rewrite-language-profile-v1/run/manifest.json), [full review](../../runs/rewrite-language-profile-v1/report/review.md), [selection](../../runs/rewrite-language-profile-v1/run/selection.json).

Native Qwen3.5-2B Q4 has1,881,825,088 tensor elements; actual process logs record25/25 GPU layers on M4/32GB. Timings cover formatting requests only. They do not establish two-call end-to-end latency, concurrent ASR, native insertion or AppleSilicon16GB acceptance. Formatter inputs retain frozen speech-spans-v1 cleanup and reused development selectors; the later speech-spans-v2 fix is not applied retroactively. Fresh24 remain ungenerated. This recipe is not enabled in production.
