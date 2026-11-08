---
name: write-blog-post
description: Use when writing or revising a post for the Arion site blog (apps/site/content/blog), including release announcements and changelog posts that cover a range of versions, product explainers and engineering essays. Covers gathering facts from tags, the desktop changelog, specs, ADRs, issues and copy catalogs; the voice; English, Simplified Chinese and Japanese versions; animated mock recordings (stage embeds) and real UI renders; credits; and the checks a post must pass.
---

# Write an Arion blog post

The blog's format and build rules live in [`docs/frontend/blog.md`](../../../docs/frontend/blog.md). Read it first: front matter keys, media sources, embeds and the build failures are defined there and are not repeated here. This skill is the writing method on top of those rules, distilled from the published posts in `apps/site/content/blog/`. Read at least one of them in full before starting; `arion-0-1-1` and `arion-0-1-4` are the release models, `building-like-an-experiment` the engineering model.

## 1. Collect facts before writing

Every sentence in a post must trace to a source in the repository or on GitHub. Write nothing from memory or from a commit subject alone.

| Need | Source |
| --- | --- |
| What shipped in a version range | `git fetch --tags`, then `git log --oneline vA..vB`; the matching sections of `apps/desktop/CHANGELOG.md` |
| Whether a version is public | GitHub releases (draft or published); a tag alone only stages a draft |
| How a change behaves, its limits | The commit's diff to `specs/`, `docs/decisions/`, `docs/project/` and `docs/backend/`; then the current text of those documents |
| Exact UI labels | `apps/desktop/ui/shared/copy/locales/en.json` and `zh-CN.json`; provider names and other fixed labels in `ui/shared/config` |
| Who reported or asked for something | Issues and pull requests linked from the commits, or issues opened in the release window |
| What is still open | `docs/operations/release-status.md`, the changelog's closing paragraph, spec acceptance rows marked open |

Rules for the collection:

- Exclude commits after the last tag in the range; they are unreleased, even when they sit on `main`.
- Exclude changes that a user of the product cannot notice (CI, site-only work, docs, refactors) unless the post is about them.
- When a document says something is unmeasured or unaccepted, the post says so too. Never turn a configured budget into a measured latency, or a test pass into broad acceptance.
- Keep a short fact sheet in `.workspace/drafts/` while writing: each claim, its source path, and any number with its owning document.

## 2. Shape the post

**Release and changelog posts** (`category: announcement`):

- Slug `arion-<newest version with dots as hyphens>` (`arion-0-1-4`), even when the post covers several versions. Title `Arion 0.1.1: our first release` for one version, `Arion 0.1.2 to 0.1.4: what changed` for a range.
- Opening: what shipped, for which platforms, in one paragraph; then a one-line upgrade instruction using the real control (Settings › About › **Check for Updates**). A first release also says what the product is.
- One `##` section per user-visible change, ordered by what users will notice most within each version, versions oldest first. Put the version in the heading when the post covers a range: `## Cloud polishing with compatible endpoints (0.1.3)`.
- Each section follows the same order: what the user sees or can now do, then how it works in one or two concrete sentences, then the boundary (what it does not do, what stays on the device, which platform is unchanged).
- A bug fix names the symptom the way a user would have met it, then the cause, then the fix. Credit the reporter inline: `[@login](https://github.com/login) [traced it](https://github.com/AstraFoundry/Arion/issues/N)`.
- End with `## What isn't there yet` (open limits from step 1; link to an earlier post's list instead of repeating it), a pointer to the full changelog and the support channels, then credits and footnotes.

**Product and engineering posts** open on a concrete scene or result, not on a thesis, and keep the same evidence rules. An engineering post states method, numbers and their limits, and links the private evidence by path when it lives in Arion-evaluation.

## 3. Voice

- Plain, specific and calm. Write for someone who uses the product, not for someone who built it. Name keys, settings and menu items exactly as the UI shows them, in bold in English and Japanese, in 「」 in Chinese.
- Short paragraphs; one idea each. Bullets only for parallel items (two fixes, the steps an agent takes, open limits).
- Numbers carry units and come from the owning document (`−42 dBFS`, `20 ms`, `about 2.5 GB`). Do not invent counts, percentages or comparisons.
- No hype words, no "not just X, it's Y", no rhetorical questions, no emoji, no exclamation marks, no section kickers. No em-dash chains; use a full stop or a colon.
- Mechanism detail goes in a footnote when it would slow the main text. Footnotes link specs and ADRs with full `https://github.com/AstraFoundry/Arion/blob/main/...` URLs and cite acceptance IDs in code (`SPH-A12`).
- Links to site pages use site URLs (`/blog/arion-0-1-1/`, `/privacy/`, `/#download`); `pnpm check` rejects a link to a route that does not exist. Never write the string `feedback.md` in a post's Markdown, not even as code or inside another URL such as `docs/project/agent-feedback.md`: the site serves it as an agent skill, and `apps/site/tests/feedback-skill.test.ts` fails on any file under `app`, `src` or `content` containing it. Describe it as the feedback skill on the website, and link the pull request instead of that document. A mock recording under `public/` may show the app's prompt text verbatim, since it is inert text, not a link.

## 4. Three locales

Write `en.md` first, then `zh.md` and `ja.md` from the facts, not as literal translations. All three must agree on `date`, `authors`, `category`, `image` and `draft`; words, tags and `updated` may differ.

- The desktop UI exists in English and Simplified Chinese only. `zh.md` uses the `zh-CN.json` labels; `ja.md` uses the English labels in bold. Fixed names that the app does not translate (provider names such as **OpenAI compatible**) stay in English in every locale.
- Headings change wording per language, so an anchor link to another post must use that locale's heading slug (`/blog/arion-0-1-1/#还没做到的事`, `#まだできていないこと`, `#what-isnt-there-yet`).
- Embed labels and captions are translated per locale (`<github-people label="贡献者" ...>`).
- Match the register of the existing posts: Chinese is direct and conversational (「你」), Japanese uses です/ます and さん for people.

## 5. Media

Show each change moving. A user-visible change is a sequence (press, speak, click, type, connect, paste), and a still image of one state cannot show it. The release posts illustrate every section that has a UI with a mock recording; static images are for the cover and for a state that has no sequence. A change without UI gets no invented diagram.

### Mock recordings (the default)

- One `<embed-frame>` per change, all scenes of a post in one engine: `apps/site/public/blog/<slug>/stage.js`, `stage.css` and `stage.<locale>.html`, the hash picking the scene (`stage.en.html#compatible`). Start from the latest post's stage (`arion-0-1-4`, which follows `arion-0-1-1`): it already has the wallpaper and rounded recording, the eased zoom camera, the smoothed cursor with click ripples, the keystroke overlay, the dictation pill, chat, settings, menu bar panel, feedback window and agent chat. Keep the files self-contained per post and delete pieces the post does not use; eslint (`pnpm --filter site exec eslint public/blog/<slug>`) flags unused functions.
- Script each scene as the user would do it, in order: move to the control, click, type or press keys (shown in the overlay), the app responds, the camera zooms in on the action and back out. End on the result, and pass `hold()` there so reduced motion shows the finished frame. Use the annotation card for the one fact the picture cannot show (the request that is now sent, the window that is now skipped).
- Draw Arion's surfaces after the real ones: render them first with the UI tests (see below) and copy layout, order and labels. Labels come from `apps/desktop/ui/shared/copy/locales/*.json`; generate `stage.<locale>.html` from the catalogs with a small script instead of retyping strings. The desktop UI has English and Simplified Chinese only, so `stage.ja.html` localizes the other apps' content (chat, agent) and keeps Arion's labels in English.
- Each locale has a poster: a 1280 × 800 screenshot of the scene's first frame (about 350 ms after load), saved as WebP under `apps/site/content/blog/<slug>/posters/<scene>.<locale>.webp`.
- Captions say what happens and end with "A mock recording" (「模拟录屏」, 「模擬録画です」), adding that no sound is captured or no provider is contacted when the scene could suggest otherwise. Never present a mock as a capture of real use.
- Before publishing, capture frames across each scene's timeline in every locale, in light and dark, and with reduced motion (Playwright with `reducedMotion: 'reduce'` and `colorScheme: 'dark'`), and look at them: camera framing, overlaps, text overflow, the final frame. Then scroll the built post to each embed and confirm it loads and plays in its sandboxed frame.
- `<dictation-pill>` is an inert, single-state illustration; prefer a scene when the point is a change over time. Its drafts are a short phrase (about five words); a long draft is truncated from the start.

### Real UI renders (the cover and reference)

- The desktop UI tests render windows headlessly on any OS: `ARION_UI_SNAPSHOTS=<dir> go test ./ui/app -run <Name>` in `apps/desktop`. The tests in `apps/desktop/ui/app/*_test.go` already drive most screens into interesting states (`TestCloudCompatibleEndpoint`, `TestMenuBarPanel`, the About tests); copy their setup into a throwaway test that sets the version being announced, `tt.SetScale(2)` and the locale, then delete it. If the render shows host-specific text (the platform in a prompt), set it to the platform a reader runs, never leave `linux amd64` in a picture, and revert any temporary source change before committing.
- Crop to the part that shows the change. Keep the alpha channel when converting: panels and popovers render with transparent rounded corners, which turn black if flattened to RGB.
- Convert to WebP (quality around 88) and keep it next to the Markdown (`./cover.webp`). The cover must be the same file in all locales; a locale-specific image goes in the body as a captioned figure.
- Quote UI strings in the text, alt text and captions exactly as the catalog has them, and compare against the render: the English and Chinese strings for the same key often differ in wording (`CloudBaseURLLabel` is "Base URL" and 「接口地址」).
- Alt text describes what the image shows, including the text a reader needs; the caption says why it is there.

## 6. Credits

- `authors` is the writer's GitHub login. Contributors are the commit authors in the range; issue reporters are the people whose issues the release closed or answered.
- `<github-people>` and `<voice-quote>` read `apps/site/content/authors`. A new login needs `pnpm blog-authors:update` (reaches `api.github.com`; set `GITHUB_TOKEN`, and `NODE_USE_ENV_PROXY=1` behind a proxy), then commit the cache. When the API is unreachable, credit new people with inline profile links and leave them out of `<github-people>` rather than hand-writing cache entries; say so in the hand-off.

## 7. Check before hand-off

1. `pnpm check` from the repository root (links, Markdown, structure).
2. `NODE_USE_ENV_PROXY=1 pnpm --filter site build` (front matter, translations, media, embeds, authors) and `pnpm --filter site test` (content rules such as the feedback skill test).
3. Serve `apps/site/out`, take full-page screenshots of the post in every locale and of `/blog/`, and read them: cover, contents list, figures, credits, footnotes.
4. Reread each claim against the fact sheet. Report checks you could not run and why.

Commit as `docs(site): add <slug> blog post`, and any skill or docs change separately or in the same change set as required by `CLAUDE.md`.
