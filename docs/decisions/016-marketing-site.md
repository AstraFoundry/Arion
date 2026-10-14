# ADR 016: Marketing site in `apps/site`

## Status

Accepted. Clarifies the scope of [ADR 001](001-desktop-native-boundaries.md).

## Context

Arion needs a public product site at `arion.astrafoundry.si`: product pages, downloads and links to the AstraFoundry account and hosted-sync waitlist. AstraFoundry's sites share one visual system, published as `@astrafoundry/design` (Sunbeam tokens and beUI motion components, React). `lyra-landing` already ships this kind of site as a Next.js static export on Cloudflare Workers.

[ADR 001](001-desktop-native-boundaries.md) and the repository rules forbid React and webviews. That rule protects the desktop app, which is a MyGo native shell; it was never about a website.

## Decision

- The marketing site lives in `apps/site`. It is not part of the desktop shell, does not ship inside the app and shares no code with it.
- React with a Next.js static export (`output: "export"`) is allowed there. ADR 001's no-React/no-webview rule constrains the desktop app only.
- The site consumes `@astrafoundry/design` from GitHub Packages. Until the package is published, it may vendor the Sunbeam tokens and beUI components from `lyra-landing`; that copy is temporary and is replaced by the package.
- It follows FSD with TypeScript slices exposing `index.ts`, and its copy lives in en, zh and ja message catalogs. The desktop interface languages (English and Simplified Chinese, [i18n](../frontend/i18n.md)) are unchanged.
- It is deployed like `lyra-landing`: Cloudflare Workers static assets.
- It stays static. It never reads sessions or cookies and never calls account APIs; sign-in and waitlist actions link to `astrafoundry.si`.

## Consequences

- The repository gains a Node/React build next to the Go modules; it is isolated to `apps/site` and does not affect desktop builds or checks.
- Design tokens exist twice: in `@astrafoundry/design` for the web and in `apps/desktop/ui/shared/theme` for the native app ([ADR 007](007-sunbeam-design-system.md)). Both follow Sunbeam; changes are mirrored by hand.
- Until the package is published, the vendored copy can drift from `lyra-landing`.
- A static site has no request-time failure modes and no user data.

## Alternatives considered

- **A separate repository.** Splits release notes, download links and docs from the product they describe.
- **Server-rendered Next.js.** No request-time data exists; a server adds cost and attack surface.
- **A non-React static generator.** The shared design package and beUI demos are React.
- **Pages on `astrafoundry.si` only.** The platform shows a product summary and links out; the full product site is Arion's own.

## References

- [ADR 001](001-desktop-native-boundaries.md), [ADR 007](007-sunbeam-design-system.md).
- `lyra-landing` ADR 001 (static Next.js with vendored beUI) and ADR 002 (Cloudflare Worker hosting).
