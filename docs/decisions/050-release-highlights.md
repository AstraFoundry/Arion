# ADR 050: Release highlights after an update

Status: Accepted.

## Context

An update restart opens Settings ([DSK-14](../../specs/001-desktop-input.md)), but nothing there says what changed. The changelog is read on GitHub, mixes fixes with features, and covers one version while a person who updates rarely may cross several. New behavior that is off by default, such as selection actions ([011](../../specs/011-selection-actions.md)), stays unknown.

## Decision

- `apps/desktop/ui/entities/whatsnew/releases.json` is the catalog: one entry per release, newest first, with highlight slide IDs or a `skip` reason. Slides cover what a person can see or do; fixes and internal changes never get a slide. The catalog may list a release before `desktop.json` is bumped to it.
- `Preferences.WhatsNewSeen` stores the version whose highlights the profile last finished or skipped. Completing onboarding records the running version, so a new install sees nothing. A profile written before the field existed has seen nothing.
- A launch whose profile has completed onboarding gets the slides of every release after the seen version up to the running one, newest first, at most six. A downgrade, a development build or a seen version at or after the running one gets none. Scripted sessions get none.
- The slides show as a dialog over the Settings window: a screen-recording style mock (the shared `ui/shared/screencast` lens and pointer) on the photograph, then the version, title and text, then the slide dots and Skip, Previous, Next or Done. Skip, Done, Escape and a click outside dismiss it and record the running version. A failed save leaves the old version, so the next launch shows the slides again. Settings › About reopens the newest release's slides.
- Release preflight (`scripts/release.mjs check`) refuses a version without its catalog entry. Skipping highlights is allowed, but it is a recorded reason, not an omission.

## Consequences

- Every release decides its highlights, and a person crossing several versions catches up in one dialog.
- Each slide needs a scene and copy in both catalogs in `ui/features/whatsnew`; the feature's tests fail for a listed slide without them.
- Highlights describe the running version's behavior. A slide about a feature that is later removed should be dropped from the catalog.

## Alternatives considered

- Rendering `CHANGELOG.md` in the app: it lists fixes and reads as release notes, not as a demonstration.
- A separate window: Settings already opens after an update restart, and a separate window would compete with it for focus.
- Recorded video: it would not follow the interface language or appearance, and would grow the package.
