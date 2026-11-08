# Marketing site (`apps/site`)

The site at `arion.astrafoundry.si` is the one place in the repository where React is allowed ([ADR 016](../decisions/016-marketing-site.md)). It follows the same FSD rules as the desktop UI, with TypeScript slices; commands, configuration and assets are in the [site README](../../apps/site/README.md).

## Layout

| Path | Holds |
| --- | --- |
| `app/(en)/` | English routes, served without a prefix; its own root layout |
| `app/[locale]/` | `zh` and `ja` routes under `/zh/` and `/ja/`; `dynamicParams = false` |
| `app/sitemap.ts`, `app/robots.ts`, `app/global-not-found.tsx` | Static sitemap with hreflang alternates, robots, the site-wide 404 |
| `app/blog-assets/` | Static route writing files that blog posts name by project path ([blog](blog.md#media-sources)), blog author avatars and testimonial avatars |
| `pages/` | Empty on purpose: it stops Next from reading `src/pages` as the Pages Router |
| `src/app` | Document shell (`LocaleRootLayout`), providers, root metadata, global CSS |
| `src/pages` | `home`, `blog`, `blog-post`, `privacy`, `terms`, `usage-data`, `self-hosting`, `credits`, `not-found`: a page component and its metadata |
| `src/widgets` | `site-header`, `site-footer`, `hero`, `feature-tour`, `open-source`, `testimonials`, `download-cta`, `post-article`, `post-list` |
| `src/features` | `switch-locale`, `theme-switch`, `download-release` |
| `src/entities` | `post` (blog front matter rules and the build-time content reader), `testimonial` (quote rules and the build-time X profile reader) |
| `content/blog` | Blog posts, one Markdown file per locale ([blog](blog.md)) |
| `content/testimonials` | Home page testimonial quotes and their cached X profiles ([testimonials](#testimonials)) |
| `src/shared` | `config` (env, links, credits), `i18n`, `lib`, `ui`, and the temporary `beui` copy |

Route files only set the locale and render a page from `src/pages`. Imports point downward and cross slices through `index.ts`; ESLint enforces both. `src/shared/beui` is vendored and exempt from lint until `@astrafoundry/design` replaces it.

## Rules

- **Copy.** Every visible string is in `src/shared/i18n/messages/{en,zh,ja}.json`. English is the source. `scripts/check-messages.mjs` runs before every build and fails it on a missing or extra key, a changed value type, a changed `{placeholder}` set, or an empty or TODO value. Product names, licenses and keyboard keys are configuration in `src/shared/config`, not copy. Zero-width spaces (U+200B) in zh/ja headings mark where `TextReveal` may break a line.
- **Links.** Outbound URLs come from `siteConfig`. The only platform link is the footer's AstraFoundry website link, `NEXT_PUBLIC_PLATFORM_URL` itself; the site links to neither the account portal nor the hosted sync waitlist; components never spell a platform URL. Community channels (Discord, Feishu group, X) are `siteConfig.community`, listed in `communityChannels` order by the header's Community link target (`/#community`, under the download options) and the footer's Community column. Internal links use `AppLink`, which keeps the reader's locale.
- **Static only.** No request-time route handler, middleware, cookie or session read, and no request to the platform at build or run time. The only runtime request is `download-release` reading the center's public `/v1/releases/{stable,beta}` summaries in the browser. Its static HTML already links each target to the center's stable download; after loading, a target without a stable release offers the newest preview (beta). Each target is a card with its version and a SHA-256 button that copies the installer checksum (the full value is its tooltip); the checksum is not printed on the page. Every "Download" link elsewhere points to the download section. A blog `<embed-frame>` pointing at another site contacts it only after the reader presses its load button. The one route handler, `app/blog-assets`, is `force-static`: it writes repository files that posts reference, and blog authors' GitHub avatars, into the export at build time. Remote blog images are downloaded at build time only to read their size. Blog author profiles and avatars are read from the committed cache `apps/site/content/authors` ([blog](blog.md#authors)); refresh it with `pnpm blog-authors:update`. Testimonial authors' X names and avatars are read from the committed cache `apps/site/content/testimonials` ([testimonials](#testimonials)); refresh it with `pnpm testimonials:update`.
- **Download targets** are macOS Apple silicon and Windows x64; Intel macOS is not offered ([ADR 021](../decisions/021-apple-silicon-only-macos.md)).
- **Minimum macOS** is 14 in `src/shared/config/site.ts`, matching `apps/desktop/desktop.json` and the packaged native runtime minimum.
- **Product claims** follow the specs: offline after model download, no cloud recognition, optional explicitly consented usage aggregates, encrypted vocabulary, and sync described as in development until SYNC-01 ships ([003](../../specs/003-memory-sync.md), [005](../../specs/005-privacy.md), [ADR 015](../decisions/015-hosted-relay-account-authorization.md)). Change the copy in the same change set as the behavior it describes.
- **Illustrations** follow [ADR 007](../decisions/007-sunbeam-design-system.md): solid mock surfaces floating over the motion-blurred photograph, no glass, glow or gradients. A mock is a picture of the product: its wrapper has `role="img"` and an `aria-label` from the dictionary, and its content is `inert`. Scripted mocks start when scrolled into view (`useInViewOnce`) and drive their steps with `useStepTimeline`, which jumps to the final step under reduced motion. Mocks share `MockWindow` and `DictationPill` from `shared/ui` rather than redrawing window chrome or the pill. The hero's dictated sentence (`demo.sentence`) is the page language's line, the same sentence that locale's 0.1.1 dictation stage types (`apps/site/public/blog/arion-0-1-1/stage.en.html` and its `zh` and `ja` files). A chapter that demonstrates mixed Chinese and English, or Pinyin, keeps that Chinese text in every locale; its chrome still comes from the page's catalog. Blog `<embed-frame>` recordings are one HTML file per locale under `public/blog/<slug>/` (`stage.en.html`, `stage.zh.html`, `stage.ja.html`), and each translation of the post points at its own file.
- **Hydration.** The static HTML carries each animated element's initial state, and hydration does not patch attributes that differ on the client. A component must render the same element with the same initial props whether or not reduced motion is on, and express the difference through its animation targets and transitions (`MotionConfig reducedMotion="user"` skips transform motion).

## Feature tour

`feature-tour` is the home page's highlights section (anchor `#values`). Its chapters are `CHAPTERS` in `config/chapters.ts`; each id names its copy under `tour.<id>`, its mock in `ui/mocks/` and its backdrop crop `public/backdrops/tour-<id>-800.webp`. From `lg` up, a sticky stage on the left shows the active chapter's backdrop and mock, and turns a page (rise or fall with the scroll direction) when another chapter's copy crosses the middle of the viewport; inactive copy dims. Below `lg`, each chapter carries its own stage above its copy. A mock replays its timeline each time its chapter becomes active. The keyboard chapter's 中/英 badge flips with the same two 0.12 s halves as the native badge ([KBD-04](../../specs/007-ime-ranking.md)). Adding a chapter means a copy entry in all three dictionaries, a mock, a timeline in `MOCK_DELAYS` and a crop in `scripts/build-assets.mjs`.

## Open source

`open-source` (anchor `#open-source`) invites readers to change the code and contribute: its buttons open the repository and its Issues, beside the build-from-source commands. Self-hosting is reached from the header and footer, not from this section.

## Testimonials

`testimonials` sits between the open-source section and the download section (anchor `#testimonials`). Each entry in `content/testimonials/testimonials.json` is `handle` (X username without `@`), `lang` (tag of the quoted text), `text` (the post as written; a blank line starts a paragraph) optional `postUrl` (`https://x.com/<handle>/status/<id>`) and `translations`, which maps each site locale to the quote's text in that locale. Every locale whose `lang` tag differs from the quote's needs a translation; a locale without one shows the post as written. Translations keep the original's meaning and tone, and each card is marked with the language it shows. The section has a title and no lead. Display names and avatars live beside the quotes: `profiles.json` plus each `x-<handle>` avatar, written by `pnpm testimonials:update` (`make testimonials-update`) from FxTwitter's public profile endpoint (`api.fxtwitter.com/<handle>`, no key). Run it after adding or changing a quote and commit the result. The build reads only the cache and publishes avatars under `/blog-assets/`; a malformed entry, a handle missing from the cache or an avatar that does not match its recorded SHA-256 fails the build. A card follows X's display rules: avatar, display name, `@handle` and X logo beside the text, linking to the post or, without `postUrl`, the profile.

Cards alternate between two rows that drift in opposite directions; each track holds its cards twice and moves by one copy, and the copy is `aria-hidden` and `inert`. Hovering or focusing a row pauses it, and a button pauses both (WCAG 2.2.2). Under reduced motion the rows stand still, the copy and the button are hidden, and each row scrolls sideways.

## Blog

Posts are Markdown in `content/blog/<slug>/<locale>.md`, rendered at build time with GFM, trusted static HTML, Shiki highlighting, lazy media and the `<embed-frame>`, `<video-clip>` and `<dictation-pill>` embeds. Writing rules, embeds and the pipeline are in [blog](blog.md).

## Agent feedback skill

`public/feedback.md` is a Markdown procedure for coding agents, served at `/feedback.md`; the desktop app's **Report a problem** window asks the agent to fetch it ([agent-assisted feedback](../project/agent-feedback.md)). It is meant to be fetched directly: no page links to it, the sitemap leaves it out, and `public/_headers` serves it as `text/markdown; charset=utf-8` with `X-Robots-Tag: noindex` (robots.txt still allows it, because some agent fetchers obey robots.txt). It is English, tells the agent to answer in the person's language, and repeats the repository and community links from `siteConfig`; `tests/feedback-skill.test.ts` keeps those links, the headers and the no-link rule in check.

## Deployment

`apps/site/wrangler.jsonc` serves `apps/site/out` as Workers static assets with no Worker script, auto trailing slashes and `404.html` for unknown paths, on the custom domain `arion.astrafoundry.si`. See [deployment](../operations/deployment.md#marketing-site).

## Product terms

`/terms/` and its localized routes contain Arion’s product terms, separate from the AstraFoundry website terms. The desktop Settings account card links here before opening browser authentication. The footer and sitemap include the page; its links preserve the reader’s locale for the privacy page and point to the repository license for software rights.

## Usage data disclosure

`/usage-data/` and its localized routes explain the exact report fields, account linkage, exclusions, consent and stopping behavior in [PRV-10](../../specs/005-privacy.md). The onboarding optional checkbox links here before authentication; Privacy links here too. The page uses the shared document layout and is included in the sitemap.
