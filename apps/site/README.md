# Arion site

The marketing site for `arion.astrafoundry.si`: a Next.js static export (React 19, Tailwind v4, Motion, next-intl) in English, Simplified Chinese and Japanese, served by Cloudflare Workers static assets. It is not part of the desktop app ([ADR 016](../../docs/decisions/016-marketing-site.md)). Structure, routing and copy rules: [site conventions](../../docs/frontend/site.md).

## Commands

Run from the repository root with the pinned pnpm (`make install` first):

| Command | Does |
| --- | --- |
| `pnpm site:check` / `make site-check` | Dictionary completeness, typecheck, lint, unit tests, static export |
| `pnpm site:build` / `make build site` | Dictionary check and static export into `apps/site/out` |
| `pnpm testimonials:update` / `make testimonials-update` | Refreshes the cached X names and avatars in `content/testimonials` (commit the result) |
| `pnpm site:assets` / `make site-assets` | Regenerates `public/backdrops`, `public/brand` and `public/og.jpg` from repository sources |
| `pnpm site:deploy` / `make site-deploy` | `wrangler deploy`; Make builds first, pnpm deploys the existing export (needs Cloudflare credentials in the shell; none are stored here) |

`pnpm --filter @arion/site dev` starts the development server on `127.0.0.1`.

## Blog

Posts are Markdown in `content/blog/<slug>/{en,zh,ja}.md`, with their media in `public/blog/<slug>/`. Front matter, embeds and lazy media are described in the [blog guide](../../docs/frontend/blog.md).

## Configuration

Three build-time variables, inlined into the bundle (`src/shared/config/env.ts`). The build fails if any is missing or is not a bare http(s) origin.

| Variable | Production value | Used for |
| --- | --- | --- |
| `NEXT_PUBLIC_SITE_URL` | `https://arion.astrafoundry.si` | Canonical URLs, hreflang, sitemap, robots, Open Graph |
| `NEXT_PUBLIC_PLATFORM_URL` | `https://astrafoundry.si` | The footer's AstraFoundry website link |
| `NEXT_PUBLIC_CENTER_URL` | `https://api.arion.astrafoundry.si` | Download buttons (`/v1/downloads/<channel>/<target>`) and the browser read of the channel summaries (`/v1/releases/{stable,beta}`) |

`.env.production` holds the production values; `.env.example` documents them. Override them in the environment for a preview build. The site only links to the platform; it never calls it and never reads a session. The download section is the one runtime request: the browser reads the center's public channel summaries to show versions and checksums ([services/updates](../../services/updates/README.md)).

## Images

`scripts/build-assets.mjs` derives every image from files already in the repository; nothing is fetched or hotlinked:

- `public/backdrops/hillside-*.webp`, the hero photograph, from `assets/backdrops/misty-hillside.jpg`.
- `public/backdrops/coastal-*.webp`, the one photograph behind every feature-tour card, from `assets/backdrops/coastal-blur.jpg`.
- `public/og.jpg` from `assets/backdrops/pink-motion.jpg`.
- `public/blog/<slug>/cover.webp`, a 4:3 cover for each post, from the same directory.
- Every photograph is credited on `/credits/`. These files are separate from the desktop app's backdrop.
- `public/brand/*.png`, and the mark composited onto `public/og.jpg`, from `apps/desktop/resources/icon.png`; `public/brand/feishu.png` is copied from `apps/desktop/ui/shared/assets/feishu.png` (source and terms in that directory's README).

## Design system

The site's tokens, font stacks, `Button`/`LinkButton`, `Container`, `Reveal`, the language controls, the demo timeline hooks and the beUI motion pieces (`TextReveal`, `ThemeToggle`) come from [`@astrafoundry/design`](https://github.com/AstraFoundry/design) (Sunbeam), pinned in `package.json`. `src/app/styles/globals.css` imports `@astrafoundry/design/styles.css` and adds only the site's own rules; `src/app/fonts.ts` loads the Latin faces as the `--font-latin-*` variables the package's stacks read. The package is installed from GitHub Packages, which needs a token: see [operations](../../docs/operations/README.md#github-packages).

beUI is MIT licensed (Saurabh Chauhan); the typefaces are OFL. Both are credited on `/credits/`.
