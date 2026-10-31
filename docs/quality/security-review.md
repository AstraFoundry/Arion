# Security review

A defensive review of the repository at `2c15ee5c` and the fixes made on top of it. It covers the shipped desktop app, the update center, the sync relay, the marketing site, release tooling and CI. `benchmarks/` source snapshots and the unpatched parts of `third_party/mygo` outside the updater were not reviewed.

Severity is the reviewer's estimate of real exploitability and impact: **High** (remote or cross-user compromise), **Medium** (a realistic local or server-side attack with meaningful impact), **Low** (requires an already strong position, or limited impact), **Info** (hardening or policy notes). Fixed findings name their commit subject; open findings need an owner decision because the fix changes a format, a platform integration or a release process.

## Findings

| ID | Severity | Status | Area | Location |
| --- | --- | --- | --- | --- |
| SEC-01 | Medium | Fixed | Sync relay pre-auth memory growth | `services/sync/interfaces/http/server.go` (`allow`, `limiter`) |
| SEC-02 | Medium | Fixed | Reachable Go standard-library and pgx vulnerabilities | `go.work`, module `go.mod` files, `deploy/*/Dockerfile*`, `.github/workflows/ci.yml` |
| SEC-03 | Medium | Fixed (native run pending) | Dictation recorded by clipboard history | `apps/desktop/infrastructure/macos/paste.m`, `apps/desktop/infrastructure/windows/windows.go` |
| SEC-04 | Low | Fixed | Unpinned third-party action in secret-bearing jobs | `.github/workflows/ci.yml`, `.github/workflows/benchmark.yml` |
| SEC-05 | Low | Fixed | Next.js advisories in the site toolchain | `apps/site/package.json` |
| SEC-06 | Medium | Open | Update version is not signed (rollback) | `third_party/mygo/updater.go:177`, `third_party/mygo/internal/update/update.go:105` |
| SEC-07 | Medium | Open | macOS credentials readable through `/usr/bin/security` | `apps/desktop/infrastructure/keystore/item.go:61`, `keychain_darwin.go:40` |
| SEC-08 | Medium | Open | macOS IME socket accepts any same-user peer | `apps/desktop/infrastructure/imebridge/ime.go:270`, `apps/desktop/infrastructure/macos/ime.go:332` |
| SEC-09 | Medium | Open | Windows elevates a helper from a user-writable directory | `apps/desktop/infrastructure/tsf/ime_windows.go:105`, `:573`; `third_party/mygo/cmd/mygo/nsis.go:329` |
| SEC-10 | Low | Open | Release builds honour the E2E control socket | `apps/desktop/main.go:37`, `apps/desktop/bootstrap/e2e.go:195` |
| SEC-11 | Low | Open | Update extraction follows in-archive symlinks | `third_party/mygo/internal/update/update.go:276` |
| SEC-12 | Low | Open | Forged callback ends the ChatGPT sign-in | `apps/desktop/internal/oauthloopback/oauthloopback.go:56`, `apps/desktop/infrastructure/cloudpolish/chatgpt.go:22` |
| SEC-13 | Low | Open | Update center signs downloads for unpublished files | `services/updates/application/service.go:141` |
| SEC-14 | Low | Open | `undici` advisories in site deployment tooling | `apps/site` → `wrangler` → `miniflare` → `undici` |
| SEC-15 | Low | Open | Static site sends no security headers | `apps/site` (no `public/_headers`) |
| SEC-16 | Info | Open | First-party actions pinned to major tags | `.github/workflows/*.yml` |
| SEC-17 | Info | Open | Cloud polishing reuses another client's OAuth ID | `apps/desktop/infrastructure/cloudpolish/chatgpt.go:21` |

## Fixed

### SEC-01 Sync relay rate limiter grew without bound

The limiter runs before request authentication and stored every distinct `X-Arion-Device-Id` header value in a map that was never pruned. An unauthenticated client could rotate large header values (up to the server's header limit) and exhaust relay memory. The relay now rejects a device header that is not a valid relay ID (`domain.ValidID`, at most 128 characters) as unauthorized, and forgets identities idle for a full one-minute window. Tests: `TestLimiterForgetsIdleIdentities`, `TestInvalidDeviceHeaderIsRejectedBeforeLimiting`. Commit: `fix(security): bound the sync relay rate limiter`.

The limiter remains per process and keyed by the claimed device ID, so a client that knows another device's random ID can still use that device's budget; per-IP limiting belongs at the reverse proxy.

### SEC-02 Vulnerable Go toolchain and pgx

`govulncheck` reported reachable go1.27.1 standard-library vulnerabilities in the sync relay and the updates service (`net/http`, HTTP/2, `crypto/tls`, `crypto/x509`; fixed in go1.27.2/go1.27.3), 14 go1.27.1 standard-library vulnerabilities at module level for the desktop on darwin/arm64 and windows/amd64, and a reachable `github.com/jackc/pgx/v5` vulnerability (GO-2026-5004, fixed in v5.9.2) in the relay. `go.work` and the desktop, sync and updates modules now select `toolchain go1.27.3` (the language version stays `go 1.27.1`); native CI and both service images build with go1.27.3; pgx is v5.9.2. After the change `govulncheck` reports no vulnerabilities for those modules. Commit: `build(security): build with go1.27.3 and pgx v5.9.2`.

### SEC-03 Paste transactions reached clipboard history

Shortcut-mode delivery places dictated text on the system clipboard for the paste and then restores the previous contents. Clipboard managers, Windows clipboard history and Windows cloud clipboard sync recorded that text; with cloud clipboard enabled, dictation left the device outside every documented network surface ([005](../../specs/005-privacy.md) PRV-01). The macOS transaction item now carries `org.nspasteboard.TransientType`; Windows paste adds the registered `ExcludeClipboardContentFromMonitorProcessing` format, and a paste whose exclusion cannot be set restores the previous text and fails. The explicit `copied` outcome is unchanged. The macOS native pasteboard test asserts the marker; `TestTransientClipboardTextIsExcludedFromHistory` checks the Windows format. Both need their native hosts (`go test ./apps/desktop/infrastructure/macos -run TestNativePasteboardTransaction` on macOS, `go test ./apps/desktop/infrastructure/windows -run TestTransientClipboardTextIsExcludedFromHistory` on Windows). Commit: `fix(security): keep paste transactions out of clipboard history`.

### SEC-04 Third-party action on a movable tag

`pnpm/action-setup@v4` ran in the native job before the step that receives the update signing key and Apple credentials during tagged releases. A moved tag could run arbitrary code in that job. It is pinned to the v4.3.0 commit. Commit: `ci(security): pin pnpm action-setup to a commit`.

### SEC-05 Next.js advisories

`pnpm audit` reported critical and high Next.js advisories for 16.0.0–16.4.0 (image optimization and `use cache` remote code execution, server action denial of service). The production site is a static export, so only the local development server was exposed. Next.js and `eslint-config-next` are now 16.4.3; `pnpm site:check` passes. Commit: `fix(security): upgrade the site to next 16.4.3`.

## Open

### SEC-06 The update signature does not bind the version

The manifest's Ed25519 signature covers only the archive's SHA-256. `Version`, the target and the delta's `From` are unsigned. Whoever controls the center, the release bucket or the admin publication step can serve any previously signed archive under a higher version string, and the app installs it: a rollback to a release with a known vulnerability, or repeated reinstallation of an old build. [ADR 019](../decisions/019-center-update-channels.md) states that these parties "cannot make an app install unsigned code"; that holds, but they can still choose which signed code. Proposed fix: sign a canonical statement of version, target, artifact kind and digest, verify it before preparation, and confirm the staged bundle's version equals the signed version. This changes the release signing format and needs a key-compatibility plan for installed versions.

### SEC-07 macOS credentials stored through `/usr/bin/security`

The hosted-sync refresh token and cloud polishing credentials (API key or ChatGPT refresh token) are written with `security add-generic-password -w <secret>` without an access list. The item's default ACL trusts the creating program, `/usr/bin/security`, so any process of the same user can read it with `security find-generic-password -w` without a Keychain prompt. The secret is also an argument of the child process while it is written (already documented in `keystore/item.go`). Proposed fix: move these items to the SecItem API used by `infrastructure/secrets` (Data Protection keychain, this-device accessibility, app access group) and migrate existing items once. Needs macOS validation and a migration decision.

### SEC-08 macOS IME socket has no peer authentication

`~/Library/Application Support/Arion/ime.sock` is mode 0600, so only the same user connects, but the macOS bridge does not call `RequireClientVerification`; Windows verifies the pipe client's process and loaded module. Any same-user process can send `hello`/`activate`, become the IME client and send `dictate`. Arion then starts microphone capture under its own microphone permission; while the Arion input source is selected, the session commits the recognized text to that client instead of the real input method. The overlay pill makes capture visible. Proposed fix: verify the peer's audit token (`LOCAL_PEERTOKEN`) against the installed `ArionInput` code signature and announced build hash before accepting `activate` or `dictate`, as on Windows.

### SEC-09 Elevated Windows helper runs from a user-writable directory

The per-user installer places the app in `%LOCALAPPDATA%\Programs\Arion`. Input-method install, repair, update and removal run `ime\arion-ime.exe` from there with `ShellExecuteEx` `runas`. Code running as the same user at medium integrity can replace that executable; the next UAC approval runs it as administrator, where it can change the machine-wide COM registration. The helper's own checks protect the copied DLLs, not the helper. Proposed fix: perform elevated work only from a protected location (for example a signed helper staged under `%ProgramFiles%\Arion\InputMethod` during the first approved install), or verify the helper's Authenticode signature and hash from an already protected component before elevation.

### SEC-10 E2E control socket in release builds

`main.go` passes `ARION_E2E_CONTROL`, `ARION_DATA_DIR` and `ARION_OFFLINE` from the environment in every build. With a control socket, a same-user process that launches the installed app can replay fixture audio and commit recognized text into the focused application through Arion's Accessibility trust; the socket is created with the default umask. Packaged E2E deliberately drives the shipped app, so removing the harness from releases is a test-strategy decision. Proposed fix: a release-time build tag that excludes the harness, or an explicit opt-in that release signing rejects, plus a 0600 socket in a private directory.

### SEC-11 Update extraction can escape through earlier symlinks

`ExtractArchive` checks names and link targets lexically. A symlink `a → .` followed by `a/b → ..` passes those checks, after which an entry `a/b/x` resolves outside the staging directory. Exploitation requires a validly signed archive, whose signer can already ship code, so this is defense in depth. Proposed fix: extract through `os.OpenRoot` or reject entries whose parent path traverses a symlink; record it as a maintained MyGo patch in [third_party](../../third_party/README.md).

### SEC-12 First callback wins on the fixed ChatGPT sign-in port

The ChatGPT sign-in must listen on `127.0.0.1:1455` (the provider's registered redirect) and delivers only the first callback. Any web page open during sign-in can request `http://localhost:1455/auth/callback?state=x`, which fails the attempt with a state mismatch. State and PKCE prevent token theft; the impact is a sign-in denial. Proposed fix: ignore callbacks whose state does not match and keep waiting until the deadline.

### SEC-13 Downloads of unpublished release files

`FileURL` presigns any flat name under `files/` in the release bucket, including staged releases not yet published and withdrawn releases. Names are predictable from versions. The files are signed builds, and apps install only what a channel manifest offers, so the exposure is early or continued public access to builds that staff have not published. Proposed fix: presign only files referenced by a published channel manifest or index.

### SEC-14 `undici` advisories in deployment tooling

`wrangler` → `miniflare` pins `undici` 7.28.0, which has eight advisories (TLS options dropped in `Agent`, cross-origin redirect cache poisoning and several denial-of-service issues; fixed in 7.30.1). It runs only in local development and `site:deploy`. The latest `miniflare` at review time still pinned 7.28.0. Proposed fix: upgrade when `miniflare` ships a patched `undici`, or add a verified `pnpm` override.

### SEC-15 Site security headers

The static export has no `_headers` file, so Cloudflare serves it without `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy` or `frame-ancestors`. The site has no forms or sessions, and JSON-LD output escapes `<`. Proposed fix: add `apps/site/public/_headers` with those headers and verify them on the deployed origin.

### SEC-16 First-party actions on major tags

`actions/checkout`, `actions/setup-node`, `actions/setup-go`, `actions/upload-artifact` and `actions/download-artifact` use major tags. They are GitHub-owned; pinning commits with automated update PRs would remove the remaining tag-move risk in release jobs.

### SEC-17 Shared OAuth client ID

Cloud polishing signs in to ChatGPT with the public OAuth client ID registered for the Codex CLI ([ADR 022](../decisions/022-cloud-polishing.md)). It is not a secret and PKCE protects the flow, but the provider attributes these sign-ins to another application. This is a provider-policy decision for the owners.

## Reviewed without findings

- **Model provisioning** (`infrastructure/provision`): catalog-pinned SHA-256 and size, bounded downloads with byte-range resume, member-by-member extraction into a random 0700 staging directory, atomic publish with the previous version retained. The `hf-mirror.com` fallback is protected by the same digests.
- **Update verification** (`third_party/mygo`): manifest and archive size limits, Ed25519 verification before extraction, staged-record revalidation from the signed artifact, symlink and type checks on recovery paths.
- **Local transports**: the IME socket directory is 0700 and the socket 0600 with lock-guarded stale-socket recovery; the Windows pipe uses a current-user DACL, `FILE_FLAG_FIRST_PIPE_INSTANCE`, rejects remote clients and verifies the peer; the GGUF runtime listens on a Unix socket in a private `MkdirTemp` directory (current-user DACL on Windows) with no TCP listener; the OAuth loopback binds 127.0.0.1 only and answers with a no-script CSP.
- **Command execution**: fixed executables with argument vectors, no shell interpolation; `browser.Open` accepts only absolute http(s) URLs without user info; product-notification URLs are host allow-listed; the `arion://` scheme only reveals a window; the Windows context reader passes the window handle through the environment, not the script.
- **Privacy**: cloud polishing runs only when `PolishEngine` is cloud, with no automatic fallback between engines; product code logs only content-free codes; native logging found prints no text.
- **Secrets**: no credentials in tracked files or reachable history (AWS documentation example keys in two S3 signing tests only); `.env` files are ignored except committed public site configuration; release signing uses an ephemeral keychain and keeps passwords out of error messages.
- **CI**: no `pull_request_target`; dispatch inputs reach shell steps only through `env`; default `contents: read` with `contents: write` only on the draft-release job; release secrets are passed only when `inputs.release` is set. After that review, direct CI (`pull_request` and `main`) grants `actions: write` only on the job that cancels the current run when the portable job fails. Release and signed-macOS callers still grant the reusable workflow `contents: read` only, so that cancel request is not authorized there.
- **Sync relay**: Ed25519 request proofs, bounded bodies, strict JSON decoding, OIDC/SET verification with an asymmetric algorithm allow-list, issuer/audience checks and JTI replay protection; services run as non-root distroless images.

## Method

- `govulncheck` v1.9.0 (built with go1.27.1, then go1.27.3): symbol scans of `packages/core`, `models`, `services/sync` and `services/updates`; module scans of `apps/desktop` with `GOOS=darwin GOARCH=arm64` and `GOOS=windows GOARCH=amd64`, because the desktop needs cgo and native SDKs for a symbol scan.
- `pnpm audit` on the frozen lockfile.
- Pattern scans of tracked files and history for private keys and AWS, GitHub, OpenAI-style, Hugging Face, GitLab and Slack tokens.
- Manual review of the areas above; native macOS and Windows behavior was not executed in this review.
