# Deployment assets

Desktop packaging lives in `apps/desktop/desktop.json` and the pinned MyGo CLI. IMK/TSF installers must be added with actual adapter lifecycle tests. [Release gates](../docs/operations/deployment.md) define signing, model provenance and registration/removal expectations.

- [Sync relay](sync/README.md) — Dockerfile and compose stack for the ciphertext relay, with its own PostgreSQL 17.
- [Docker](docker/README.md) — image boundary for that relay. Desktop apps are not containers.
- [Kubernetes](k8s/README.md) — optional later operator packaging; not required for initial self-hosting.
