# Model catalog

`catalog.json` is the reviewed, embedded catalog ([006 Models](../specs/006-models.md), [ADR 004](../docs/decisions/004-automatic-model-selection.md)). Weights are never committed: the app downloads them from the pinned URLs and verifies them.

Public aggregate evaluation reports live under `evaluations/`. Private `scripts/export-catalog.mjs` validates completed raw evidence and exports content-free model aggregates with exact source identities. Public `node scripts/catalog-evaluation.mjs <reviewed-summary.json> --reviewed` imports the reviewed export into `evaluations/<runId>.json` and updates catalog report links. It does not consume private raw-run directories. This promotion boundary is defined in the [repository split](../docs/project/repository-split.md).


## Native recognition

The catalog contains Qwen3-ASR 1.7B speech.cpp Q4_K for `darwin-arm64` and `windows-amd64`; its `darwin-amd64` platform entry and the former MLX artifact are retained as provenance only, since Intel macOS and the MLX runtime are unsupported ([ADR 021](../docs/decisions/021-apple-silicon-only-macos.md)). Each artifact has a distinct immutable revision, file inventory, digest, runtime and quantization provenance. The Q4_K bundle is the explicitly adopted default under [ADR 020](../docs/decisions/020-native-qwen-asr.md), with native measurements recorded independently. Historical X-ASR/ONNX results remain in the private evaluation Git archive and are not selectable product backends. The [repository split](../docs/project/repository-split.md) defines internal evidence references.

`arion-models ensure -artifact qwen3-asr-1.7b-gguf-q4k -root <cache>` provisions pinned files; `-offline` verifies without downloads. `arion-models import-files -directory <downloaded-files> -artifact <id> -root <cache>` verifies local files and uses the same atomic provisioning path. No tokenizer or checkpoint is accepted by filename alone. Old model caches are not automatically deleted.

Weights are outside the installer. Native speech.cpp/ggml runtime code, Metal resources and dependency licenses are staged by `scripts/stage-asr-runtime.mjs`; `scripts/asr-runtime.json` pins the source/runtime closure. Windows acceptance must execute on an actual Windows host.

The existing local rewrite model and its evaluation provenance remain in `catalog.json`; recognition adoption does not change rewriting.
