# Rime installation and Arion input-method updates

This is a read-only source comparison, not execution evidence for either
upstream installer. Reviewed revisions are Weasel
`82b50c9e9b9ee0bd31357db6f25ddc804546c7f7` and Squirrel
`46962d2efae5be70389c778d0a6c52f669072827`.

## Official upstream mechanisms

Weasel's [NSIS installer](https://github.com/rime/weasel/blob/82b50c9e9b9ee0bd31357db6f25ddc804546c7f7/output/install.nsi)
requests administrator execution. Its upgrade branch stops the old server,
invokes the old setup's unregister operation, stages the replacement, invokes
setup again, and deploys the schemas. It preserves user files separately from
program code and supports silent installation. Its
[native setup](https://github.com/rime/weasel/blob/82b50c9e9b9ee0bd31357db6f25ddc804546c7f7/WeaselSetup/imesetup.cpp)
renames locked old DLLs to `.old.N` and schedules their deletion after reboot.
This demonstrates automatic installer-owned maintenance with an OS privilege
boundary; it does not establish that DLLs can be overwritten while loaded.

Squirrel's [postinstall script](https://github.com/rime/squirrel/blob/46962d2efae5be70389c778d0a6c52f669072827/scripts/postinstall)
stops the previous host, registers the input source, builds shared data, then
enables/selects it as the console user. Its
[Makefile](https://github.com/rime/squirrel/blob/46962d2efae5be70389c778d0a6c52f669072827/Makefile)
packages the bundle and installer separately and uses `/Library/Input Methods`
for native installation. Updating input-method components belongs to the
application/install lifecycle rather than a separate routine user task.

## Arion policy and platform adapters

Both adapters expose `UpdateAvailable` and `UpdateAutomatically`. Bootstrap
maintains an existing enabled source when no owned dictation or key hold is
active. macOS and direct pre-loader Windows registrations repeat the unused-source
check after staging; stable Windows loaders can publish a core while selected.
Absent or deliberately disabled sources are not installed/enabled in the
background. Deferral is pending work, while a real failure stops automatic
retries and retains a retry action. A retry can finish ordinary-user setup
without another administrator request when machine registration already matches.

macOS retains its signed per-user bundle transaction, controlled host stop,
registration/readiness observation and rollback. Windows retains the protected
`Program Files/Arion/InputMethod/<SHA256>` layout. A new helper validates the
complete bundled keyboard closure, serializes machine maintenance, stages a
complete independent version, compares the expected previous registration,
and publishes the core pointer through its fixed loader. Direct registration
migration also switches the COM path/current profile. Legacy `0409` is retired only after
the new `0804` Arion profile exists. Registration/category failures restore the
captured prior COM path, profile descriptions and managed-category membership.
An atomically published protected intent containing only old/new hashes marks
an interrupted handover as incomplete, so matching DLL bytes alone cannot hide
unfinished registration. Corrupt intents fail observation.

Old version directories remain intact. A loaded old COM factory and its
dependencies can continue in an existing editor. A stable facade's later
activation can resolve the published core after prior references are released;
new editor processes use the current registration. The updater does not kill editors,
rewrite loaded DLLs, delete learning data, change other TIPs, or force a selected
source away. Retained old versions are a deliberate safety cost; this change
does not introduce a background garbage collector for them.

Voice connection compatibility is separate from installation currency. Windows
verifies the named-pipe peer and actual loaded protected module before accepting
its wire protocol; a compatible retained component can remain connected after
an application update. Windows protocol 2 requires the complete keyboard closure
and native private/password/PIN-scope refusal. Installed-browser testing exposed
the missing private-scope guard in older hosts, so protocols 0 and 1 are refused
for voice instead of treating that known defect as backward compatibility.
A newer application's bundle hash alone is not
a reason to reject an older compatible client or ask the user to reopen an
editor. This does not upgrade the code of a loaded voice-only DLL or repair an
unsafe older acquisition implementation in place; that one-time migration still
requires native component reload. Arion now separates a fixed COM bootstrap
from its versioned core. Its stable processor facade releases the core delegate
on deactivation, so retaining the outer service object does not alone prevent
the next activation from resolving the new core. This is an independent
implementation informed by the shipped-loader evidence, not copied proprietary
code or a claim of transferring active composition between versions. Ordinary
core publication preserves the registered profile and changes only its protected
`CorePath`; direct pre-loader migration retains the unused-source boundary.

Machine registration and protected-code publication still need UAC. Arion's
per-user application installer has no administrator grant to reuse. Therefore
a Windows IME upgrade can request one OS authorization after the application
update; it does not require uninstall/reinstall clicks and never loops on a
denied request. This is automatic maintenance, not an assertion of completely
silent privileged installation. No persistent elevated service is introduced.

## Validation boundary

`test-windows-ime-registration.mjs` includes callback-only phase/failure/rollback
tests, actual atomic intent-file tests and independent read-only system queries.
Go TSF/bootstrap tests cover one machine transaction followed by ordinary-user
enablement, current-build no-op, legacy migration, selection races, disabled and
absent sources, cancellation, denial and explicit retry. These checks do not
perform installation, registration or UAC. Actual upgrade authorization,
rollback after native API failure, already-open editor behavior and source
persistence require separate installed-system acceptance. macOS native tests
must run on a reachable macOS host.
