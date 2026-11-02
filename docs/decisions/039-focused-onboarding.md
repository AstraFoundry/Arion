# ADR 039: Focus onboarding on ready offline dictation

## Status

Accepted. The placement of account, polishing and translation only in Settings is superseded by [ADR 040](040-onboarding-account-polish-translation.md). Recognition download confirmation, combined dictation practice, the keyboard remaining in Settings, and sign-in not granting usage consent remain.

## Context

Optional accounts, keyboard installation, polishing and translation delayed the first dictation. Separate key and translation pages repeated configuration. Setup could finish before recognition was ready, and startup provisioned missing speech resources before download confirmation.

## Decision

Follow the required sequence and gates in [DSK-14](../../specs/001-desktop-input.md). Request explicit initial recognition download consent with the size shown. Load verified installed recognition resources without network access during unfinished onboarding. Completed profiles and explicit scripted skip-onboarding runs retain their existing provisioning behavior.

Combine shortcut selection and actual dictation practice. Optional settings remain in the existing Settings sections opened on completion. Translation has one disclosure; account sign-in has product terms links and does not grant usage consent. Usage sharing is changed explicitly in Settings > Account; the removed onboarding checkbox no longer supplies consent. Existing account consent remains intact.

## Consequences

First-use flow depends only on offline dictation readiness. Downloads overlap permission setup after confirmation. Practice can be skipped, but unavailable resources or revoked permissions cannot be presented as a completed setup. Tests cover missing resources, confirmation, permission changes, live shortcut behavior and optional settings independently. Native OS checks remain necessary for actual editor insertion and permission dialogs.
