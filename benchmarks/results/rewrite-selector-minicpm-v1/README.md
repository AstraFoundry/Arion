# MiniCPM5-1B comparison result

Both baseline and balanced configurations are rejected:80/77 conforming choices of101,11/4 harmful choices and10/20 missed repairs;37/27 required items repaired of47. Prior exact-request Qwen controls conform92/94 with3/4 harmful and6/3 missed.55 choice differences include6 improvements,35 regressions and14 both-correct normalization differences.

[Protocol and reproduction](../../reproduction/rewrite-selector-minicpm-v1/README.md), [complete review](review.md), [machine summary](summary.json), [raw completed and interrupted evidence](../../runs/rewrite-selector-minicpm-v1/README.md).

All202 actual choices,12 complete native probes,32 failed source/context/output tuples,55 changed pairs and1 interrupted native-log completion were personally read. The latter has no captured response envelope and is disclosed separately; it is not a complete case. No new final bodies, ASR, Vision, production integration, independent blind or actual16GB acceptance. Compatibility and cache parity do not establish semantic quality.
