# Marketing site (`apps/site`)

The site at `arion.astrafoundry.si` is the one place in the repository where React is allowed ([ADR 016](../decisions/016-marketing-site.md)). It follows the same FSD rules as the desktop UI, with TypeScript slices; commands, configuration and assets are in the [site README](../../apps/site/README.md).

## Layout

| Path | Holds |
| --- | --- |
| `app/(en)/` | English routes, served without a prefix; its own root layout |
| `app/[locale]/` | `zh` and `ja` routes under `/zh/` and `/ja/`; `dynamicParams = false` |
| `app/sitemap.ts`, `app/robots.ts`, `app/global-not-found.tsx` | Static sitemap with hreflang alternates, robots, the site-wide 404 |
| `app/blog-assets/` | Static route writing files that blog posts name by project path ([blog](blog.md#media-sources)) |
| `pages/` | Empty on purpose: it stops Next from reading `src/pages` as the Pages Router |
| `src/app` | Document shell (`LocaleRootLayout`), providers, root metadata, global CSS |
| `src/pages` | `home`, `blog`, `blog-post`, `privacy`, `self-hosting`, `credits`, `not-found`: a page component and its metadata |
| `src/widgets` | `site-header`, `site-footer`, `hero`, `feature-tour`, `privacy-summary`, `open-source`, `download-cta`, `post-article`, `post-list` |
| `src/features` | `switch-locale`, `theme-switch`, `download-release` |
| `src/entities` | `post` (blog front matter rules and the build-time content reader) |
| `content/blog` | Blog posts, one Markdown file per locale ([blog](blog.md)) |
| `src/shared` | `config` (env, links, credits), `i18n`, `lib`, `ui`, and the temporary `beui` copy |

Route files only set the locale and render a page from `src/pages`. Imports point downward and cross slices through `index.ts`; ESLint enforces both. `src/shared/beui` is vendored and exempt from lint until `@astrafoundry/design` replaces it.

## Rules

- **Copy.** Every visible string is in `src/shared/i18n/messages/{en,zh,ja}.json`. English is the source. `scripts/check-messages.mjs` runs before every build and fails it on a missing or extra key, a changed value type, a changed `{placeholder}` set, or an empty or TODO value. Product names, licenses and keyboard keys are configuration in `src/shared/config`, not copy. Zero-width spaces (U+200B) in zh/ja headings mark where `TextReveal` may break a line.
- **Links.** Outbound URLs come from `siteConfig`. The platform links (account, hosted sync waitlist) are built from `NEXT_PUBLIC_PLATFORM_URL`; components never spell a platform URL. Internal links use `AppLink`, which keeps the reader's locale.
- **Static only.** No request-time route handler, middleware, cookie or session read, and no request to the platform at build or run time. The only runtime request is `download-release` reading the center's public `/v1/releases/{stable,beta}` summaries in the browser. Its static HTML already links each target to the center's stable download; after loading, a target without a stable release offers the newest preview (beta) with its version and SHA-256. Every "Download" link elsewhere points to the download section. A blog `<embed-frame>` pointing at another site contacts it only after the reader presses its load button. The one route handler, `app/blog-assets`, is `force-static`: it writes repository files that posts reference into the export at build time. Remote blog images are downloaded at build time only to read their size.
- **Download targets** are macOS Apple silicon and Windows x64; Intel macOS is not offered ([ADR 021](../decisions/021-apple-silicon-only-macos.md)).
- **Minimum macOS** is 14 in `src/shared/config/site.ts`, matching `apps/desktop/desktop.json` and the packaged native runtime minimum.
- **Product claims** follow the specs: offline after model download, no cloud recognition or telemetry, encrypted vocabulary, and sync described as in development until SYNC-01 ships ([003](../../specs/003-memory-sync.md), [005](../../specs/005-privacy.md), [ADR 015](../decisions/015-hosted-relay-account-authorization.md)). Change the copy in the same change set as the behavior it describes.
- **Illustrations** follow [ADR 007](../decisions/007-sunbeam-design-system.md): solid mock surfaces floating over the motion-blurred photograph, no glass, glow or gradients. A mock is a picture of the product: its wrapper has `role="img"` and an `aria-label` from the dictionary, and its content is `inert`. Scripted mocks start when scrolled into view (`useInViewOnce`) and drive their steps with `useStepTimeline`, which jumps to the final step under reduced motion. Mocks share `MockWindow` and `DictationPill` from `shared/ui` rather than redrawing window chrome or the pill.
- **Hydration.** The static HTML carries each animated element's initial state, and hydration does not patch attributes that differ on the client. A component must render the same element with the same initial props whether or not reduced motion is on, and express the difference through its animation targets and transitions (`MotionConfig reducedMotion="user"` skips transform motion).

## Feature tour

`feature-tour` is the home page's highlights section (anchor `#values`). Its chapters are `CHAPTERS` in `config/chapters.ts`; each id names its copy under `tour.<id>`, its mock in `ui/mocks/` and its backdrop crop `public/backdrops/tour-<id>-800.webp`. From `lg` up, a sticky stage on the left shows the active chapter's backdrop and mock, and turns a page (rise or fall with the scroll direction) when another chapter's copy crosses the middle of the viewport; inactive copy dims. Below `lg`, each chapter carries its own stage above its copy. A mock replays its timeline each time its chapter becomes active. The keyboard chapter's 中/英 badge flips with the same two 0.12 s halves as the native badge ([KBD-04](../../specs/007-ime-ranking.md)). Adding a chapter means a copy entry in all three dictionaries, a mock, a timeline in `MOCK_DELAYS` and a crop in `scripts/build-assets.mjs`.

## Blog

Posts are Markdown in `content/blog/<slug>/<locale>.md`, rendered at build time with GFM, trusted static HTML, Shiki highlighting, lazy media and the `<embed-frame>`, `<video-clip>` and `<dictation-pill>` embeds. Writing rules, embeds and the pipeline are in [blog](blog.md).

## Deployment

`apps/site/wrangler.jsonc` serves `apps/site/out` as Workers static assets with no Worker script, auto trailing slashes and `404.html` for unknown paths, on the custom domain `arion.astrafoundry.si`. See [deployment](../operations/deployment.md#marketing-site).
