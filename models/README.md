# Model catalog

`catalog.json` is the reviewed, embedded catalog ([006 Models](../specs/006-models.md), [ADR 004](../docs/decisions/004-automatic-model-selection.md)). Weights are never committed: the app downloads them from the pinned URLs and verifies them.


## Native recognition

The catalog contains Qwen3-ASR 1.7B MLX affine 8-bit for `darwin-arm64` and speech.cpp Q8_0 for `darwin-amd64` / `windows-amd64`. Each has a distinct immutable revision, file inventory, digest, runtime and quantization provenance. They are explicitly adopted defaults under [ADR 020](../docs/decisions/020-native-qwen-asr.md), with native measurements recorded independently. Historical X-ASR/ONNX results remain in `benchmarks/` and are not selectable product backends.

`arion-models ensure -artifact qwen3-asr-1.7b-mlx-8bit -root <cache>` provisions pinned files; `-offline` verifies without downloads. `arion-models import-files -directory <downloaded-files> -artifact <id> -root <cache>` verifies local files and uses the same atomic provisioning path. No tokenizer or checkpoint is accepted by filename alone. Old model caches are not automatically deleted.

Weights are outside the installer. Native runtime code, Metal shaders, Swift compatibility libraries and dependency licenses are staged by `scripts/stage-asr-runtime.mjs`; `Package.resolved` and `scripts/asr-runtime.json` pin the source/runtime closure. Intel and Windows acceptance must execute on their actual hosts.

The existing local rewrite model and its evaluation provenance remain in `catalog.json`; recognition adoption does not change rewriting.
