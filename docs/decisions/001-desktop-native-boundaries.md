# ADR 001: Desktop native shell and configurable input adapters

Status: Accepted scope; platform integration pending. The no-React/no-webview rule covers the desktop app only; the marketing site is governed by [ADR 016](016-marketing-site.md).

## Context

The product needs macOS and Windows voice input, optional system IME and local privacy. The user selected MyGo and both shortcut/IME modes. MyGo native supports desktop windows but does not provide microphone capture, foreign-editor insertion or complete IMK/TSF hosts.

## Decision

Pin MyGo v0.4.6 for native UI in a separate desktop Go module. Keep contracts/application/domain portable. OS infrastructure provides capture, hotkeys, target context and insertion. IMK/TSF host adapters are user-configurable P0 tasks rather than implicit behavior of the MyGo shell. Future mobile adapters reuse semantics without treating MyGo as a mobile framework.

## Consequences

Native compilation/package/signing is required on both OSes. TSF COM and IMK server lifecycle need platform harnesses and potentially separate hosts/IPC. FSD is implemented with exported Go package APIs; no React/TypeScript toolchain is implied. A single app/process design cannot be assumed to register a system IME.
