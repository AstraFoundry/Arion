# ADR 048: Selection action URLs

Status: Accepted. Backend contract for [011](../../specs/011-selection-actions.md). The popover, onboarding page, and settings card are SEL-06 and SEL-07; the disable menu is SEL-08.

## Context

A selected passage can be copied, saved as a hotword, or handed to a local client. The client handoff has to be a fixed URL. A stored template that can become a shell command, a keystroke, or an Apple event would put selected text on an execution path.

## Decision

Keep selection preferences on the existing preferences record. The master switch and both built-in clients default off. The clients are Claude (`claude://claude.ai/new?q={{text}}`) and ChatGPT (`codex://new?prompt={{text}}`). Normalize restores any other template to the built-in for that id and drops unknown clients.

Fill `{{text}}` with the percent-encoded prompt or the raw selection. Allow only `https`, `claude`, and `codex`. Refuse every other scheme, including `chatgpt://`, `doubao://`, and `feishu://`, and refuse a second placeholder. Cap the Claude query at 14,000 Unicode characters before encoding.

Open the filled URL with the operating system's URL opener. Do not interpolate text into a shell, synthesize a keystroke, or send an Apple event. Read the selection on mouse-up from Accessibility on macOS and from `TextPattern.GetSelection` on Windows. Copy uses the existing clipboard writer. Vocabulary uses `AddHotword`. A secure field is refused before either write. Do not log the selected text.

## Verification

Portable tests cover encoding, the length cap, refused schemes, and a missing client. Native tests cover the reader, the popover action list, copy, hotword reopen, URL equality, an empty selection, secure refusal, and the master switch. They do not launch Claude, ChatGPT, Doubao, or Feishu.

## Consequences

The popover renders the in-memory offer without changing the URL rules. Onboarding ends with the selection page DSK-14 describes.
