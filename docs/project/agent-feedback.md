# Agent-assisted feedback

A person who hits a problem can hand it to the coding agent they already use (Codex, Claude Code and others). Arion gives them a one-line prompt; the agent fetches the feedback skill from the site and runs the conversation, the issue and, optionally, the fix. Arion itself sends nothing.

## Entry points

All of them open the same **Report a problem** window (`pages/feedback`, opened by `Actions.OpenFeedback`, owned by `bootstrap.openFeedback`):

| Where | Control |
| --- | --- |
| Menu bar panel | **?** icon button beside the readiness state (`widgets/statusheader`) |
| Settings › About | **Report a bug** row: the existing GitHub Issues link and an **Ask an agent** button |
| macOS input menu | **Report a problem…** under **Open Arion**, shown when LaunchServices resolves the app; opens `arion://feedback` |
| Windows language bar menu | **Report a problem…** after **Open Arion**; opens `arion://feedback` and reports the open failure like **Open Arion** |

`arion://feedback` (any case, an optional trailing `/` or query) opens the window instead of revealing Settings or onboarding; every other `arion:` URL keeps its [existing behavior](account-signin.md). The window opens whether or not onboarding is complete.

## Window

The layout follows onboarding: an illustration on the left and the content on the right.

- `widgets/agentdemo` is a looping, screen-recording-style mock (9.6 s): the pointer copies the prompt, pastes it into a coding agent's input (`config.Agent`), sends it, and the agent opens a GitHub issue. A camera zooms and pans toward the pointer's work. The timeline is a pure function of time (`frameAt`) with eased keyframes; with reduced motion it holds the frame with the issue opened, unzoomed. Its texts come from the copy catalogs; the paste keycap shows the platform shortcut.
- The right column has the title, one paragraph, the prompt in a monospace card with **Copy** (it reads **Copied** for 1.8 s after `CopyText` succeeds), and an **Open an issue yourself** link to the issue form.

The prompt (`FormatFeedbackPrompt`) names `config.FeedbackSkillURL`, the app version (or "development build" when unversioned) and the platform as `GOOS`/`GOARCH` (for example `macOS arm64`). It contains no paths, account data or logs.

## The feedback skill

`apps/site/public/feedback.md` is served at `https://arion.astrafoundry.si/feedback.md` ([site](../frontend/site.md#agent-feedback-skill)). It tells the agent to:

1. Talk in the person's language and in plain words, one or two questions at a time, and publish nothing without showing it and getting a yes.
2. Answer a plain question itself from the repository (README, `AGENTS.md` routes, specs and code) without filing anything, and move on to a report only if the answer reveals a problem or the person wants one.
3. Offer **report** or **report and fix** only if it is a current flagship coding model in an agent that can edit and run code; otherwise offer the report alone.
4. Diagnose the problem, collect the OS and architecture, compare the person's version with the newest published release (changelog and closed issues) and send them to Settings › About › Check for Updates when a newer release fixes it, and search existing issues before filing.
5. Read only the relevant window of the local logs (`<data directory>/logs/events.*.jsonl`, see [logging](../backend/logging.md)), redact anything personal, show the excerpt and attach it only with consent. Larger logs go through **Export logs…**, uploaded by the person.
6. File the issue with `gh` (or hand the person a draft for the web form), then offer to fix it. A fix reads `AGENTS.md`, keeps to the project rules and opens a pull request that references the issue with `Fixes #n`.
7. Finally invite the person to the Discord or Feishu group.

Changes to the procedure, the log location or the community links must update the skill in the same change set.

## Verification

`go test ./ui/widgets/agentdemo` checks the story order, continuous pointer and zoom movement, that the camera never uncovers the stage, and renders frames; `go test ./ui/app -run 'Feedback|AsksAnAgent'` covers both locales, copying, the issue link, reduced motion and the panel and About entries; `go test ./bootstrap -run FeedbackURL` covers the URL. `apps/site/tests/feedback-skill.test.ts` checks the skill's links, its headers and that no page links to it. The input-menu items need the native builds: the macOS item compiles with the IMK host, and the Windows language-bar contract test (`scripts/test-windows-language-bar.mjs`) runs only on Windows. `ARION_UI_SNAPSHOTS=<dir>` saves the window and recording frames for review.
