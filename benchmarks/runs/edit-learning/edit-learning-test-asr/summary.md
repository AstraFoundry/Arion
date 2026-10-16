# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160-xasr-final:warm | 154 | 0.0582 | 0.0397 | 0.2210 | 0.5867 | 0.5867 | 237.7679 | 0.2123 |
| xasr-160-xasr-final-rules:warm | 154 | 0.0582 | 0.0397 | 0.2210 | 0.5867 | 0.5867 | 156.0539 | 0.1977 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
