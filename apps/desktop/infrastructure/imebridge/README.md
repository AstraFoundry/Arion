# Native client compatibility

The common bridge binds composition to a single admitted connection, activation,
session, revision and acknowledgement. Windows configures a client verifier;
macOS retains its existing protocol-0 connection contract without this Windows
verification policy.

Windows obtains the pipe client PID from `GetNamedPipeClientProcessId`, opens and
holds the process handle, and requires the same user SID/session and ordinary
unelevated medium integrity. A live handle is checked again when Capture creates
the capability. Windows enumerates that process's loaded ArionTSF module and
checks its exact protected Program Files version directory, DLL SHA, privileged
ownership and write permissions, absence of redirecting ancestors/hardlinks, and
the complete adjacent keyboard manifest closure for protocol 2. The current app
payload is independently verified. A compatible retained module may differ from
the current package's DLL SHA.

Only protocol 2 satisfies the Windows voice safety contract: native composition
must refuse both TSF `IS_PASSWORD` and `IS_PRIVATE` text stores. Installed-browser
testing found that Chromium password stores can expose `IS_PRIVATE`; older
protocol-0/1 components did not reliably refuse these stores. Both protocols
are rejected with `CONTRACT_UNSUPPORTED` and the dedicated protocol diagnostic,
including all formerly reviewed legacy hashes. Unknown protocols also fail.
Future protocol-2 modules can remain compatible across different version hashes
after the same complete provenance checks. Failed identity/closure proof uses
the artifact diagnostic. The retained f9 artifact is only a file/module proof
fixture, not an accepted voice client or proof of protocol-2 native behavior.
An administrator can replace protected code; this policy does not claim to
protect a compromised same-user editor process or prove which of its modules
authored each message. It creates no broker or persistent privileged service.

Verification runs before admission. A rejected client cannot replace a valid
client, and a slowly verified earlier hello cannot evict a newer admitted hello.
Capture checks the proof and writes begin under the same mutex, then checks the
capability again after acknowledgement. An already pending verification can be
awaited for at most one second, respecting caller cancellation. Focus activation
wakes this wait. Hashing remains per connection; no cache bypass is implemented.

Run `node scripts/test-windows-ime-client.mjs --require-existing` with Go and the
Windows C compiler on PATH for race tests, a real isolated pipe/client-process
proof, low-integrity duplicate-token rejection, a read-only retained f9 closure
and loaded-module proof with legacy protocol rejection, and a macOS ARM64
compile-only gate. Without the flag a
missing retained fixture is explicitly skipped. The runner writes source hashes,
commands and raw Go JSON to `.workspace/tests/windows-ime-client/run-*/report.json`.
It never installs/registers TSF or opens a user's keyboard profile.

Before protocol-0/1 admission was withdrawn, October 24 ordinary-process
retained-f9 checks observed 368–487 ms for the complete
pipe/token/module/closure verifier and 370–570 ms for the closure alone. A
separate race-instrumented run took about 1.11 seconds per verifier. Those
historical timings do not establish current protocol-2 acceptance. They are
integration observations under development
background activity, not an idle latency benchmark or installed-browser
acceptance. Per-connection hashing can be perceptible. The current conservative
policy rejects a process with multiple simultaneously loaded ArionTSF modules;
that upgrade coexistence case has not been accepted in this change. macOS was
cross-compiled here, not executed.

Windows API references: [pipe client PID](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-getnamedpipeclientprocessid),
[process handles](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-openprocess),
and [module enumeration](https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-module32firstw).
