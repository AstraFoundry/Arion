# ADR 038: Contextual keyboard completion

## Status

Accepted for macOS implementation. Updates the settled lane and ID-only boundary of [ADR 036](036-two-stage-keyboard-ranking.md). Installed-editor acceptance and broad quality evaluation remain separate gates.

## Decision

The existing opt-in local keyboard model setting enables a separate text-bearing result for hosts advertising `keyboard-completion-v1`. The macOS host advertises it; Windows and older hosts retain ID-only ranking. The fast lane still ranks existing IDs. The settled lane may generate one whole-input Chinese candidate even when the engine has only one comparable candidate, using the existing resident local model and approved preceding, following and visible-window context. No cloud provider, automatic model download or additional context permission is introduced.

A completion uses the native engine's displayed syllable boundaries. The pinned character-reading dictionary builds a GBNF grammar for full syllables and initials, including zh/ch/sh abbreviations. A full syllable must not be reinterpreted as several initials. The explicit `NONE` sentinel abstains without proposing text; nonempty output must cover every syllable. An independent dictionary check validates output after inference, including when a worker fails to honor grammar. Model-provided pronunciation, confidence or candidate IDs never authorize new text.

The native `allowCompletion` request field must serialize as a JSON boolean. Boxing a C comparison directly produces a number, which the Go boolean contract rejects before scheduling either lane. The native fixture checks the serialized field type.

The completion bounds are 64 raw lowercase ASCII Pinyin/apostrophe bytes, 2–32 syllables/Han characters, and a 256 KiB grammar. The existing snapshot/context limits still apply. The settled lane keeps its 150 ms debounce and now has a three-second total deadline including debounce, context and inference. This is a configured limit, not a latency guarantee. Cancellation, input revision, native activation, navigation, voice takeover, host maintenance and preference changes revoke pending work. Settled abstention does not undo a fast promotion.

Native insertion replaces only the current unselected whole-input Rime menu with a translation containing the provisional candidate and the original lazy candidate stream. It does not insert editor text. The candidate has `arion_completion` provenance; normal Rime selection/commit delivers it, and the original candidates remain selectable. It is a SimpleCandidate, not a dictionary Phrase; a proposal cannot enter ordinary Chinese dictionary learning. Existing confirmed context learning still observes actual commits. The native host freezes further automatic replies after applying a completion, preventing old engine-ID mappings from being reused.

Context acquisition reuses Accessibility. AX exposes a marked composition either before a collapsed caret or as the exact selected text. Both representations require an exact preedit match; selected preedit is not removed a second time from preceding text. Unrelated selections, changed targets, unreadable windows and oversized context still reject the request. No OCR or cross-field cached context is substituted.

Developer-mode diagnostics distinguish a generated proposal, uncertain abstention and `completion_phonetic_mismatch`, without recording text. Bridge delivery remains distinct from native application. Maintenance now cancels active inference immediately; the old short deadline had masked that missing cancellation.

## Evidence and validation

The [feasibility investigation](https://github.com/AstraFoundry/Arion/pull/67) found a new-candidate construction and phonetic integration path in Doubao, and surrounding-context requests and cancellation in Chatterfly. These are static artifact observations, not recovered server algorithms or a parity claim. The local implementation retains Arion's existing privacy and offline boundaries.

Public tests cover phonetic/segmentation validation, capability negotiation, single-candidate requests, generation ownership, selected versus collapsed AX preedit, native commit/cancel/navigation/edit/deactivation, punctuation and original-candidate mapping. The opt-in `TestLocalCompletion` uses the actual local worker on small authored regressions. A successful smoke does not establish the requested population error tolerance or installed-editor context acquisition. Windows generation is not implemented.

Run `go test ./apps/desktop/infrastructure/llm/completion ./packages/core/application ./apps/desktop/infrastructure/imebridge ./apps/desktop/bootstrap ./apps/desktop/infrastructure/diagnostics`, `node scripts/test-keyboard-input.mjs` and `pnpm check`. For real local inference, build `./apps/desktop/cmd/arion-llm` into a temporary path, set `ARION_KEYBOARD_TEST_WORKER` to that executable, `ARION_KEYBOARD_TEST_RUNTIME` to the staged llama-server directory and `ARION_KEYBOARD_TEST_MODEL` to the provisioned model directory, then run `go test ./apps/desktop/infrastructure/llm/completion -run '^TestLocalCompletion$' -count=1 -v`. All three environment values are required.

For the real Safari Accessibility fixture, run `ARION_NATIVE_E2E=1 ARION_E2E_CONTEXT=1 go test -tags arion_e2e ./apps/desktop/e2e -run '^TestReadContextFromWindow$' -count=1`. The runner needs Accessibility access and Safari must remain focused. The fixture verifies selected Pinyin, surrounding text, visible-window context and closed context gates; it does not exercise an installed IME and model together.
