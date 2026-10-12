# Arion

A local-first voice input monorepo for macOS and Windows, built with [MyGo](https://github.com/egoist/mygo) native Go UI. iOS and Android are future targets.

The product combines streaming speech recognition, conservative correction, local candidate ranking, and synchronized personal memory. Users can choose shortcut dictation or system IME integration. A self-hosted service relays encrypted memory; dictation remains usable offline after model provisioning.

## Current state

This change establishes specifications, research, portable contracts, a minimal native shell, and real repository checks. It does **not** implement microphone capture, ASR, correction, ranking, IMK/TSF, persistence, or sync. Their implementation tasks are explicitly tracked in the [agent handoff](docs/project/agent-handoff.md).

## Start here

1. Read [AGENTS.md](AGENTS.md), the [product scope](docs/project/README.md), and [architecture](docs/project/architecture.md).
2. Read the [specification index](specs/README.md) and [research evidence](docs/research/README.md).
3. Choose an unclaimed task from the [agent handoff](docs/project/agent-handoff.md).
4. Follow [local development](docs/operations/local-dev.md) and run checks before handing work off.

```sh
make install
make check
make desktop-build
make e2e
make ablation-plan
make stream-plan
```

JavaScript tooling uses pinned pnpm 10.33.0 and Node 24; npm installation is rejected. Native-window E2E, [evaluation runners](tools/evaluation/README.md), [paper-grounded metrics](docs/research/benchmark-methodology.md) and [streaming dataset research](docs/research/streaming-datasets.md) are provided. No production model scores are claimed. Native desktop builds use Go and the pinned MyGo CLI. See [repository ownership](docs/project/repository.md) for the workspace layout and [release gates](docs/operations/deployment.md) before distribution.
