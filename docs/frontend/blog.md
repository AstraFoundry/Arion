# Site blog (`apps/site`)

The blog lives at `/blog/` (and `/zh/blog/`, `/ja/blog/`) on the marketing site. Posts are Markdown files in the repository, rendered to static HTML at build time like every other page ([site conventions](site.md)). The layout follows long-form publishing practice (and Anthropic's articles): a ~65 character reading column, media that breaks out wider, captions and footnotes in the sans face, and an inline numbered contents list.

## Writing a post

```
apps/site/content/blog/<slug>/en.md     # one file per site locale; all three are required
apps/site/content/blog/<slug>/zh.md
apps/site/content/blog/<slug>/ja.md
apps/site/content/blog/<slug>/…         # optional: images and videos next to the Markdown
apps/site/public/blog/<slug>/…          # embedded HTML, and media you prefer to serve from public/
```

The slug is lowercase words joined by hyphens and becomes the URL. Each file starts with front matter:

```yaml
---
title: Where your dictation goes
summary: One or two sentences; the lead under the title, the index summary and the meta description.
date: 2026-10-27            # YYYY-MM-DD, UTC
category: product           # announcement | product | engineering (labels in blog.categories)
cover: ./cover.webp         # optional; with coverAlt. Any media source (see Media sources)
coverAlt: What the cover shows
---
```

Every post exists in every locale, like every other page, so the language switch and hreflang links never point at a missing translation. Translations must agree on `date`, `category` and `cover`. The build fails on a missing translation, an unknown front matter key, a bad date or category, or a cover that does not resolve ([`entities/post`](../../apps/site/src/entities/post)).

Links inside a post use site URLs (`/privacy/`, `/blog/<slug>/…`); they keep the reader's locale, and the repository check (`pnpm check`) fails when one names no English route, post or file in `public/`.

The index lists posts newest first: the newest as the lead, the rest as rows. A post ends with up to three other posts.

## Content

Markdown is GitHub-flavored: tables, footnotes (`[^note]`), task lists, strikethrough and autolinks. Fenced code is highlighted at build time with Shiki (GitHub light and dark themes) and gets a copy button; an unknown language fails the build. `##` headings get slug ids and a hover anchor; three or more `##` headings add a contents list.

### Media sources

Every image, `cover`, and `<video-clip>` `src` or `poster` may be written three ways:

| Written as | Example | Resolves to |
| --- | --- | --- |
| Remote URL (an image host) | `https://img.example.com/a.webp` | Used as is. The build downloads it once to read its size; an unreachable URL fails the build. Pages request it with `referrerpolicy="no-referrer"`. |
| Site path | `/blog/<slug>/a.webp`, `/og.jpg` | A file in `apps/site/public/`. |
| Project path | `./a.webp`, `../../../../desktop/ui/shared/assets/backdrop.jpg` | Relative to the Markdown file, to any file inside the repository. Published as `/blog-assets/<content hash>/<name>` by a static route (`app/blog-assets`), so the URL changes whenever the bytes do. Allowed types: avif, gif, jpg, png, svg, webp, mp4, webm. |

A missing file, a path that leaves the repository, `//host/…` and `data:` URLs fail the build. Remote images make the build depend on the image host being reachable, and each reader's browser contacts that host when the image nears the viewport; prefer project paths for images that matter. Building with remote images behind an HTTP proxy needs `NODE_USE_ENV_PROXY=1`.

### Images

Images are lazy. An image alone in a paragraph becomes a figure in the wide column, and its Markdown title becomes the caption:

```md
![Alt text describing the image](/blog/<slug>/figure.webp "Caption shown under the figure")
```

The build sets every image's intrinsic size so the page never jumps; stating both `width` and `height` in HTML skips the measurement, and stating only one fails the build. Every image needs alt text. A figure never upscales: a small image keeps its size, a tall one is capped at the viewport height, and the caption is as wide as the image. Images fade in once decoded.

Raw HTML is allowed and trusted (posts are reviewed like code), but it must stay static: `<script>`, `<style>`, `<iframe>`, `<object>`, `<embed>`, `<video>` and `on*` attributes fail the build. `<kbd>`, `<sup>`, `<details>` and similar elements render as written; `<aside>` renders as a side note.

### Embeds

Write each on its own line (a block embed inside a sentence fails the build).

| Element | Attributes | Behavior |
| --- | --- | --- |
| `<embed-frame>` | `src`, `title` (required), `ratio` (`16/9`) or `height` (px), `caption` | Interactive HTML in an iframe, loaded lazily. A file under `public/blog/<slug>/` runs sandboxed (`allow-scripts`, opaque origin). Another site is not contacted until the reader presses the "Load content from {host}" button. |
| `<video-clip>` | `src`, `poster`, `width`/`height` (read from the poster when omitted), `caption`, `autoplay`, `loop` | `preload="none"`: only the poster loads up front. `autoplay` makes a muted loop that plays while half in view and pauses off screen; under reduced motion it shows controls instead. |
| `<dictation-pill>` | `state` (`listening` with `draft`, `inserted`, `learned` with `term`), `label` (required), `caption` | The site's live `DictationPill` on a stage, as a labelled, inert illustration. |

Embedded HTML files are post content, not site code: keep them self-contained, language-neutral or one file per locale, and follow the system color scheme. The mapping from element to component is `widgets/post-article/lib/components.ts`; validation is `lib/rehype-post.ts`. A new embed needs both, plus a row here.

## Code

| Path | Holds |
| --- | --- |
| `src/entities/post` | Front matter validation, translation checks, sorting (`model`); reading `content/blog` at build time (`api`, server-only) |
| `src/widgets/post-article` | The Markdown pipeline (unified: remark-gfm → rehype-raw → validation → figures → media resolution → media sizes → slugs → Shiki), the element-to-component map, embeds, and `prose.css` |
| `src/widgets/post-list` | The index lead and post rows |
| `src/pages/blog`, `src/pages/blog-post` | Page composition and metadata (Open Graph `article` with the cover) |
| `src/shared/ui/dateline.tsx` | Date and label line |
| `src/shared/lib/server.ts` | `resolveMedia` (remote, site and project paths), `mediaSize` (local read or one build-time download) |
| `app/blog-assets/[hash]/[name]/route.ts` | Static route that writes each project-path file into the export |

All Markdown work happens during `next build`; the browser receives HTML plus the small client components (lazy image, copy button, video, embed facade). The sitemap lists every post in every locale.
