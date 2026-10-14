# Docker

The relay image is [deploy/sync/Dockerfile](../sync/Dockerfile): a multi-stage build of `services/sync`, static binary, uid 65532, no shell. [compose.yaml](../sync/compose.yaml) runs PostgreSQL 17, a one-shot migrate, and the API on `127.0.0.1:8080`. Desktop apps and IME hosts stay native packages.
