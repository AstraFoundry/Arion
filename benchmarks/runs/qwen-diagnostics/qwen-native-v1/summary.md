# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160:warm | 14 | 0.0819 | 0.3469 | 0.4545 | 0.0714 | 0.0714 | 61.9480 | 0.1650 |
| xasr-160-rules:warm | 14 | 0.0819 | 0.3469 | 0.4545 | 0.0714 | 0.0714 | 49.6775 | 0.1552 |
| xasr-160-xasr-final:warm | 14 | 0.0234 | 0.2857 | 0.3636 | 0.2857 | 0.2857 | 170.7442 | 0.1804 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
