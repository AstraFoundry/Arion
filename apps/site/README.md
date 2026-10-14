# Arion site

The marketing site for `arion.astrafoundry.si`: a Next.js static export (React 19, Tailwind v4, Motion, next-intl) in English, Simplified Chinese and Japanese, served by Cloudflare Workers static assets. It is not part of the desktop app ([ADR 016](../../docs/decisions/016-marketing-site.md)). Structure, routing and copy rules: [site conventions](../../docs/frontend/site.md).

## Commands

Run from the repository root with the pinned pnpm (`make install` first):

| Command | Does |
| --- | --- |
| `pnpm site:check` / `make site-check` | Dictionary completeness, typecheck, lint, unit tests, static export |
| `pnpm site:build` / `make site-build` | Dictionary check and static export into `apps/site/out` |
| `pnpm site:assets` / `make site-assets` | Regenerates `public/backdrops`, `public/brand` and `public/og.jpg` from repository sources |
| `pnpm site:deploy` / `make site-deploy` | Builds, then `wrangler deploy` (needs Cloudflare credentials in the shell; none are stored here) |

`pnpm --filter @arion/site dev` starts the development server on `127.0.0.1`.

## Configuration

Two build-time variables, inlined into the bundle (`src/shared/config/env.ts`). The build fails if either is missing or is not a bare http(s) origin.

| Variable | Production value | Used for |
| --- | --- | --- |
| `NEXT_PUBLIC_SITE_URL` | `https://arion.astrafoundry.si` | Canonical URLs, hreflang, sitemap, robots, Open Graph |
| `NEXT_PUBLIC_PLATFORM_URL` | `https://astrafoundry.si` | The AstraFoundry account link (`/account`) and the hosted sync waitlist link (`/products/arion/waitlist/hosted-sync`) |

`.env.production` holds the production values; `.env.example` documents them. Override them in the environment for a preview build. The site only links to the platform; it never calls it and never reads a session.

## Images

`scripts/build-assets.mjs` derives every image from files already in the repository; nothing is fetched or hotlinked:

- `public/backdrops/*.webp` from `apps/desktop/ui/shared/assets/backdrop.jpg` ("Flowers falling" by Roma Kaiuk, Unsplash License; credited on `/credits/`). The source is 800×1000, so the wide crops are upscaled; the photograph is motion-blurred, which hides it.
- `public/brand/*.png` and `public/og.jpg` from `apps/desktop/resources/icon.png`.

## Temporary vendored design code

`@astrafoundry/design` is not published yet. Until it is, the site carries a minimal copy of the Sunbeam tokens and beUI components from `lyra-landing`. Each piece is replaced by the package and must not grow in the meantime:

| Path | Source |
| --- | --- |
| `src/app/styles/sunbeam.css` | `lyra-landing` Sunbeam tokens, without the canvas and info tokens |
| `src/shared/beui/components/motion/text-reveal.tsx` | beUI text animation; local change: U+200B separates units, and reduced motion settles the full final state |
| `src/shared/beui/components/motion/theme-toggle.tsx`, `action-swap.tsx` | beUI, unchanged |
| `src/shared/beui/lib/ease.ts`, `utils.ts` | beUI helpers |
| `src/app/fonts.ts` | `lyra-landing` font setup (Fraunces, IBM Plex Sans, JetBrains Mono) |

beUI is MIT licensed (Saurabh Chauhan); the typefaces are OFL. Both are credited on `/credits/`.
