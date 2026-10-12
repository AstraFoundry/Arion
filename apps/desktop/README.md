# Desktop shell

MyGo native Go module with centralized English copy and FSD Go packages. The current window reports that capture/input adapters are not connected. The UI does not simulate dictation or register an input source.

See [local development](../../docs/operations/local-dev.md), [architecture](../../docs/project/architecture.md), and [desktop spec](../../specs/001-desktop-input.md).

Use `go tool mygo dev` from this directory for a bundled development app, or `go tool mygo build` for the current host. MyGo CLI is pinned by this module's tool directive. OS inference/IME/capture implementations belong in `infrastructure/` and implement core ports.
