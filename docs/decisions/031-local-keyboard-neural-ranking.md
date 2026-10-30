# ADR 031: Opt-in local keyboard neural ranking

## Status

Accepted. Scheduling and context scope superseded by [ADR 036](036-two-stage-keyboard-ranking.md).

## Context

The keyboard already produces offline bilingual candidates and applies personal context counts. The installed polishing model can select an existing Chinese candidate, but authored pilot results include destructive promotions and roughly half-second latency. These results justify an optional enhancement, not a default ranking change or a claim of IME-specific model quality.

## Decision

Expose `keyboard.neuralRanking`, default false, independently of voice polishing and its cloud provider. Reuse the resident local polishing worker. Enabling does not download weights; a separate download action provisions the existing model. Turning off both local polishing and keyboard ranking releases the worker. No keyboard request selects cloud inference or reads credentials.

The existing mutually verified input-method connection advertises `keyboard-ranking-v1`. Requests carry activation, a monotonically increasing revision, and a snapshot of existing candidates; replies echo all identities. Context consists only of the native adapter's recent confirmed commit, never arbitrary surrounding host text. Raw input is bounded to 256 UTF-8 bytes, preceding context to 2,048 bytes, and the leading window to 27 candidates. Only Han candidates with the first candidate's actual consumed range participate. English, Emoji and other-range slots stay in place. No context or fewer than two eligible candidates causes no model call.

The local menu appears immediately. A 75 ms debounce and 750 ms request deadline bound enhancement; one running request and one replaceable pending snapshot prevent backlog. Navigation, selection, edits, cancellation, activation changes, updates and disabling revoke stale proposals. Only a validated stable promotion of an existing ID can update the same untouched first page. Normal engine selection still owns commit. Timeout, malformed output and unavailable service leave that menu intact. Settings surface loading, missing-model and failure states without a redundant ready-state paragraph; they do not promise that each revision receives a model result.

Voice preparation suspends and cancels keyboard completion before audio work. Direct polishing also preempts it. The native path remains usable with the desktop app stopped. Elevated Windows input does not gain permission to connect to voice IPC.

## Validation and limits

Shared menu tests cover fixed non-Chinese slots and range/navigation rejection. Go bridge tests cover verified snapshots, revisions and lifecycle cancellation. The macOS isolated Rime/AppKit fixture checks real candidate promotion, exact commit and stale/canceled reply rejection; supplied responses in this fixture are protocol tests, not inference evidence. A [real local worker smoke run](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/runs/keyboard-neural-runtime-v1/README.md`) returned an applicable candidate promotion within the production deadline and verified speech suspension. This single synthetic case is not an accuracy benchmark. Existing model reports remain historical measurements of their recorded prompt and policy versions. Windows source compilation does not establish native TSF acceptance. Installed TextEdit dispatch and actual Windows host tests remain required independently of isolated tests.

## Alternatives

The original single-model decision is extended by ADR 036. Remote keyboard ranking requires a separate consent and privacy decision. Neither is introduced here.
