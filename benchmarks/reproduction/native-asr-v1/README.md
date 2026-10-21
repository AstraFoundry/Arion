# Native ASR migration reproduction

[Accepted results and limitations](../../results/native-asr-v1/README.md) distinguish native worker compute, offline signed-package acceptance and real-time coordinator replay. Weights and executable runtimes remain in the ignored workspace/cache; original corpus audio is shared from `benchmarks/corpora/asr-v1`.

- `source-driver/` freezes the Go/core/catalog and Swift sources used for the coordinator measurement before later domain-placement and lifecycle-cleanup edits. It contains no credentials or model weights.
- `run-compute-v1.py` / `run-compute-v2.py` are exact bytes matching their recorded runner hashes. `run.py` additionally supports signed-package network denial and child-process RSS measurement.
- `native-driver.json` records original absolute artifact locations and digests. Recreate it with `prepare-driver.py` after staging a new runtime; never substitute a different binary under an old digest.
- `packaging/` records pinned native runtime sources, Swift/runtime notice metadata and the validation configuration. Swift dependencies are frozen in Package.resolved.
- `report.mjs` replays quality and timing from retained results. `verify.mjs` checks the archive inventory and shared input bytes without inference.

```sh
pnpm asr:native:verify
node benchmarks/reproduction/native-asr-v1/report.mjs benchmarks/corpora/asr-v1/corpus.json benchmarks/runs/native-asr-v1 /tmp/native-asr-replay.json
```

For a new live run, build through `make build desktop`, import/provision the approved model through `arion-models`, and use a fresh output directory. `run.py --worker <signed-app>/Contents/Resources/asr/arion-asr --model <verified-model-directory> --corpus benchmarks/corpora/asr-v1/corpus.json --catalog models/catalog.json --out <new-directory> --deny-network` checks all 118 development inputs, warmups, cancellation and resident-model reuse. The current live coordinator configuration is `benchmarks/configs/desktop-models.json`; `ARION_ASR_WORKER` explicitly selects the staged worker in this evaluation-only launch.

The native GGUF conversion has a different quantization and runtime from MLX. Windows/Intel results cannot be inferred from these Apple-silicon runs. Development activity occurred on the host; these are exploratory measurements rather than an isolated-lab or production-load guarantee. The first-partial and final-latency gates remain failed.
