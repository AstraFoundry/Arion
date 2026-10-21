# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160-qwen3asr-final:warm | 118 | 0.0850 | 0.0290 | 0.2989 | 0.5130 | 0.5130 | 977.3213 | 0.2855 |
| xasr-160-qwen3asr-17-final:warm | 118 | 0.0630 | 0.0258 | 0.2826 | 0.5565 | 0.5565 | 1765.4350 | 0.4154 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
