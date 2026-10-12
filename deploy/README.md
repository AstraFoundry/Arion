# Deployment assets

Reserved packaging and self-hosted sync assets. No deployable server image or system IME installer exists in this foundation.

Desktop packaging lives in `apps/desktop/mygo.json` and the pinned MyGo CLI. IMK/TSF installers must be added with actual adapter lifecycle tests. [Release gates](../docs/operations/deployment.md) define signing, model provenance and registration/removal expectations.

- [Docker](docker/README.md) — reserved self-hosted relay boundary.
- [Kubernetes](k8s/README.md) — optional later operator packaging; not required for initial self-hosting.
