# 028: Separate Windows keyboard and voice host admission

## Status

Accepted.

## Context

The TSF service rejected every elevated host before opening its local keyboard
runtime. With UAC disabled, ordinary desktop applications run elevated and the
input source appeared selectable but Pinyin and mode switching never started.
The restriction originated in the voice pipe's process trust policy.

## Decision

Allow local keyboard conversion in medium- and high-integrity x64 desktop
hosts, including elevated applications. Reject low-integrity, AppContainer,
system-integrity and secure-mode hosts. Keep field safety, owned compositions,
verified program files and encrypted learning unchanged (KBD-01, KBD-04).

Voice IPC retains unelevated medium-integrity peers, equal user/session/integrity
and protected-module validation. Elevated keyboard hosts do not connect to the
voice pipe and their Voice input menu action remains disabled. No automatic
process elevation, UIAccess grant or UAC setting change is introduced.

New keyboard profile directories and mode files explicitly name the current
user as owner. Elevated tokens otherwise default to the Administrators group,
which fails the existing user-ownership validator. Existing untrusted paths are
refused; ownership and ACL checks are not relaxed or silently repaired.

## Consequences

Local typing can work when the desktop app cannot establish voice transport.
UAC-disabled desktops can use Pinyin, mode switching and local learning without
changing system settings. Supporting elevated voice delivery remains a separate
trust-boundary decision.

Real token tests and an owned ACP fixture verify elevated keyboard activation,
Chinese commits, Shift/menu switching, private mode persistence, protected-field
refusal and disabled voice transport (KBD-A11). Manual key dispatch does not
establish installed system dispatch or third-party editor acceptance.

## Alternatives considered

- Require users to enable UAC: leaves local keyboard compatibility dependent on
  a machine setting unrelated to in-process Pinyin conversion.
- Relax the shared voice token policy: would expand cross-process text access
  when only local keyboard compatibility is required.
- Run the whole app elevated: does not fix the original host rejection and
  unnecessarily expands permissions.

## References

- [IME requirements](../../specs/007-ime-ranking.md).
- [Windows input method](../operations/windows-input-method.md).
- [TSF in-process registration](https://learn.microsoft.com/en-us/windows/win32/tsf/text-service-registration).
