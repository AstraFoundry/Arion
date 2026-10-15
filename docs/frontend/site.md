# Marketing site (`apps/site`)

The site at `arion.astrafoundry.si` is the one place in the repository where React is allowed ([ADR 016](../decisions/016-marketing-site.md)). It follows the same FSD rules as the desktop UI, with TypeScript slices; commands, configuration and assets are in the [site README](../../apps/site/README.md).

## Layout

| Path | Holds |
| --- | --- |
| `app/(en)/` | English routes, served without a prefix; its own root layout |
| `app/[locale]/` | `zh` and `ja` routes under `/zh/` and `/ja/`; `dynamicParams = false` |
| `app/sitemap.ts`, `app/robots.ts`, `app/global-not-found.tsx` | Static sitemap with hreflang alternates, robots, the site-wide 404 |
| `pages/` | Empty on purpose: it stops Next from reading `src/pages` as the Pages Router |
| `src/app` | Document shell (`LocaleRootLayout`), providers, root metadata, global CSS |
| `src/pages` | `home`, `privacy`, `self-hosting`, `credits`, `not-found`: a page component and its metadata |
| `src/widgets` | `site-header`, `site-footer`, `hero`, `values`, `privacy-summary`, `open-source`, `download-cta` |
| `src/features` | `switch-locale`, `theme-switch` |
| `src/shared` | `config` (env, links, credits), `i18n`, `lib`, `ui`, and the temporary `beui` copy |

Route files only set the locale and render a page from `src/pages`. Imports point downward and cross slices through `index.ts`; ESLint enforces both. `src/shared/beui` is vendored and exempt from lint until `@astrafoundry/design` replaces it.

## Rules

- **Copy.** Every visible string is in `src/shared/i18n/messages/{en,zh,ja}.json`. English is the source. `scripts/check-messages.mjs` runs before every build and fails it on a missing or extra key, a changed value type, a changed `{placeholder}` set, or an empty or TODO value. Product names, licenses and keyboard keys are configuration in `src/shared/config`, not copy. Zero-width spaces (U+200B) in zh/ja headings mark where `TextReveal` may break a line.
- **Links.** Outbound URLs come from `siteConfig`. The platform links (account, hosted sync waitlist) are built from `NEXT_PUBLIC_PLATFORM_URL`; components never spell a platform URL. Internal links use `AppLink`, which keeps the reader's locale.
- **Static only.** No route handler, middleware, cookie or session read, and no request to the platform at build or run time.
- **Minimum macOS** is 14 in `src/shared/config/site.ts`, matching `apps/desktop/mygo.json` and the packaged native runtime minimum.
- **Product claims** follow the specs: offline after model download, no cloud recognition or telemetry, encrypted vocabulary, and sync described as in development until SYNC-01 ships ([003](../../specs/003-memory-sync.md), [005](../../specs/005-privacy.md), [ADR 015](../decisions/015-hosted-relay-account-authorization.md)). Change the copy in the same change set as the behavior it describes.
- **Illustrations** follow [ADR 007](../decisions/007-sunbeam-design-system.md): solid mock surfaces floating over the motion-blurred photograph, no glass, glow or gradients. A mock is a picture of the product: its wrapper has `role="img"` and an `aria-label` from the dictionary, and its content is `inert`. Scripted mocks start when scrolled into view (`useInViewOnce`) and drive their steps with `useStepTimeline`, which jumps to the final step under reduced motion.
- **Hydration.** The static HTML carries each animated element's initial state, and hydration does not patch attributes that differ on the client. A component must render the same element with the same initial props whether or not reduced motion is on, and express the difference through its animation targets and transitions (`MotionConfig reducedMotion="user"` skips transform motion).

## Deployment

`apps/site/wrangler.jsonc` serves `apps/site/out` as Workers static assets with no Worker script, auto trailing slashes and `404.html` for unknown paths, on the custom domain `arion.astrafoundry.si`. See [deployment](../operations/deployment.md#marketing-site).
