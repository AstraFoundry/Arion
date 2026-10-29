# Local keyboard runtime smoke evidence

The unmodified `run.log` records `TestLocalKeyboardWorker` using real Qwen3.5-2B Q4_K_M weights, the resident local worker and the production 750 ms / 75 ms scheduler configuration on macOS arm64. The synthetic request selected `c1` (线) from 西安 / 线 after 这条 in 473 ms. Voice suspension rejected the next request without inference. This single smoke case is not an accuracy benchmark, installed IMK acceptance or Windows execution.

`identities.json` hashes the source, executable and weight bytes used. Weights and runtime binaries remain outside Git. The fixture and assertions are in `apps/desktop/infrastructure/rewriting/keyboard_native_test.go`; no personal input or credentials are used.

Reproduce with explicitly provisioned paths (the model directory must contain `Qwen3.5-2B-Q4_K_M.gguf`):

```sh
ARION_KEYBOARD_TEST_RUNTIME=/absolute/runtime-directory \
ARION_KEYBOARD_TEST_MODEL=/absolute/model-directory \
go -C apps/desktop test ./infrastructure/rewriting -run TestLocalKeyboardWorker -count=1 -v
```

The native candidate application test separately uses real Rime and AppKit with a supplied protocol reply. Run `node scripts/test-keyboard-input.mjs`; it does not claim model inference or installed-host dispatch.
