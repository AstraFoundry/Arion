# Explicit copular typed-value repair

Actual Go execution of156 sources with speech-spans-v2; zero LLM/ASR generation. Exactly sp-self-correct and lp-stress-correction bodies change. The first is new compared with v1; all120 historical natural sources remain unchanged. Earlier frozen inference inputs/results are preserved.

The repair requires an explicit Chinese correction cue, a complete copular typed-value clause and a matching unit. Quotes, negation, uncertainty, unrelated predicates and untyped entities are excluded. Raw pre-fix failing tests, post-fix domain/full-core tests and fuzz execution logs are retained. Source snapshots allow isolated replay; a passing cleanup is not complete model/native acceptance.

```sh
pnpm rewrite:cleanup-copula-v1:verify
```

[Manifest](../../runs/rewrite-cleanup-copula-v1/manifest.json) pins the exact inputs, source snapshot and raw observations. To replay Go manually, create a fresh scratch module `arion.local/core` using the local Go version; copy `sources/domain` to `domain`, this main.go to the module root, and the archived test to domain. Run `GOWORK=off go run .` with the absolute inputs.json path and compare stdout bytes to observations.json. No weights or private recordings are required.
