# Production ASR hotword development probe

This paired development diagnostic measures the actual `asr.Offline.Finalize` port with `ports.FinalizeRequest.Hotwords`. It does not rewrite or replace ASR text. [Results and limitations](../../results/asr-hotwords-v1/README.md) accompany every original output.

`benchmarks/corpora/asr-hotwords-v1/evaluation.json` freezes 16 cases, provenance, source identities, expected canonical terms, forbidden insertions and numeric concepts. `requests.json` is the separate reference-free inference plan. It contains only audio/hash, opaque identities, declared canonical spelling lists and treatment/purpose. The Go driver has no reference, expected-answer, protected-span or scoring fields. It checks audio digests, reuses the production WAV reader, loads one resident production offline recognizer and calls `Finalize` once per request.

The nine relevant lists are controlled fixture-specific development configuration. Approved targets come from already exposed `user-modify-asr-v2/dev/memory.json`; additional manual terms are explicitly enumerated in the evaluation file. This is a development spelling-bias probe, with no held-out memory acquisition or retrieval claim. All original source fixtures are development inputs; no test corpus is read. Aliases, wrong forms, utterance text and evaluator labels never enter prompts.

There are 32 paired calls, two excluded training warmups and four initial Arion context/empty probes. Paired order alternates by fixture. Seven negative cases receive irrelevant lists: four authored TTS ordinary/homophone controls, two original natural-read corpus excerpts and one generated silence. After inspecting the completed main run, a separately retained Codex context/empty sequence was added because it has an observable context effect. This supplemental diagnostic is explicitly post-observation and does not alter the frozen main run or scores. It adds four calls, for 42 completed calls in the paired and supplemental probes, excluding the separately scoped runtime smoke. The Arion sequence remains visible despite being insensitive to context.

The shared model is the already installed catalog `qwen3-asr-1.7b-mlx-8bit`; every catalog file is checked before invocation. Runtime/driver SHA-256s, hardware, power state, repository dirty state and exact source bytes are retained. No weights were downloaded. Model/binary files remain private and are identified by digest. Source dependency pins and build logs are in [runtime smoke reproduction](runtime-smoke/README.md), with its separately scoped actual protocol observations under `benchmarks/runs/asr-hotwords-v1/runtime-smoke-v1`.

## Replay without weights

```sh
pnpm asr:hotwords:verify
pnpm asr:hotwords:report --out /tmp/asr-hotwords-summary.json
```

The report rejects incomplete/changed observations, duplicate/unpaired results, changed treatments, order, source or original audio. It uses the shared ASR normalizer and reports micro-averaged language-specific CER/WER/MER; silence has its own denominator. Canonical recall includes exact case-sensitive matches and case-insensitive whitespace-normalized matches. Both are 4/9 versus 6/9 in this run. Numeric acceptance and unrelated insertion checks are declared evaluator-only proxies. These checks do not establish complete semantic fidelity.

## Fresh actual inference

Build a new driver in an ignored path with the repository's Go toolchain, and a new worker using the pinned native build process. A GPU-capable native macOS environment is required. Do not overwrite an existing directory.

```sh
GOCACHE="$PWD/.workspace/go-cache" go build -o .workspace/asr-hotwords-v1/driver benchmarks/reproduction/asr-hotwords-v1/driver.go
python3 benchmarks/reproduction/asr-hotwords-v1/run.py --worker <new-worker> --model <verified-catalog-model-directory> --driver .workspace/asr-hotwords-v1/driver --out .workspace/asr-hotwords-v1/new-paired-run
python3 benchmarks/reproduction/asr-hotwords-v1/run.py --worker <new-worker> --model <verified-catalog-model-directory> --driver .workspace/asr-hotwords-v1/driver --input benchmarks/corpora/asr-hotwords-v1/sensitive-isolation-requests.json --out .workspace/asr-hotwords-v1/new-isolation-run
node benchmarks/reproduction/asr-hotwords-v1/report.mjs .workspace/asr-hotwords-v1/new-paired-run /tmp/new-summary.json .workspace/asr-hotwords-v1/new-isolation-run
```

The sandboxed model launch failed before readiness and has no inference output. The successful invocation required the host's GPU permission. Both original stdout/stderr streams are retained. Hardware serial/UUID/UDID were removed only from published metadata, with original private metadata hashes and explicit transformation notes; raw inference and source bytes are unchanged.
