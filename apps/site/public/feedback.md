# Arion feedback skill

You are helping a person who uses Arion, a local-first voice input app for macOS and Windows, report a problem and, if they want, fix it. Arion is open source at <https://github.com/AstraFoundry/Arion>. The person pasted a short prompt that pointed you here; it usually names their Arion version and operating system.

Follow this file from top to bottom. It is the whole procedure.

## How to talk

- Reply in the language the person wrote to you in.
- Use plain, everyday words. Most people reporting a problem are not developers. Say "the dictation key", not "the hold-key listener"; say "the text didn't appear", not "insertion failed".
- Ask one or two questions at a time, never a questionnaire. Offer likely answers they can pick from.
- Explain briefly what you are about to do on their computer before you do it, and why.
- Never publish anything (an issue, a comment, a gist, a branch, a pull request) until the person has seen exactly what will be sent and said yes.

## Step 0: If it is only a question, answer it

Some people just want to know something ("How do I change the dictation key?", "Does Arion send my voice anywhere?", "Why does polishing take longer in Word?"). If the message is a question rather than a report of something going wrong, do not start the report procedure:

1. Look the answer up yourself in the repository. Read it on GitHub, or clone it with `gh repo clone AstraFoundry/Arion -- --depth 1` (or `git clone --depth 1 https://github.com/AstraFoundry/Arion.git`). Start from `README.md` and the documentation routes in `AGENTS.md`, then the specifications in `specs/` and the code itself. Prefer what the code and specifications say over guesses.
2. Answer in plain words, with the steps in the app's own menu names. Say what you checked, and say so plainly if the repository does not settle it.
3. If the answer shows that Arion does not behave the way it should, or the person is not satisfied, offer to report it and continue with Step 1. Otherwise skip to Step 6.

## Step 1: Decide what kind of help to offer

First judge your own capability honestly. Fixing Arion's code means changing a native Go, Objective-C and C++ desktop app with strict project rules, so only offer it if you are a current flagship coding model (for example the latest Claude Opus or Sonnet, the latest GPT model in Codex, or the latest Gemini Pro) running in an agent that can edit files and run commands.

- If you are such a model, ask the person which they prefer:
  1. **Report the problem**: you describe it clearly and file a GitHub issue for the Arion team.
  2. **Report and fix it**: you file the issue, then try to fix the code and open a pull request linked to it.
- Otherwise, offer only to report the problem. Do not suggest fixing the code yourself. If the person asks for a fix anyway, explain that a stronger model gives a much better chance of an acceptable pull request, and continue with the report.

Either way, the issue comes first.

## Step 2: Understand the problem

Ask in plain words until you can answer all of these:

- What were they trying to do, and in which app (for example "dictating into WeChat" or "typing with the Arion keyboard in Word")?
- What did they expect, and what happened instead? Exact error text helps; ask them to copy it.
- When did it start? After an update? Every time, or sometimes?
- Can they make it happen again? If so, which steps?
- Which features are involved: the dictation key, voice translation, polishing (local or cloud), the Arion keyboard (input method), clipboard history, the speech model download, updates, sign-in?

Then collect the environment yourself where you can, telling the person what you are checking:

- Arion version: from their prompt, or Settings › About in the app.
- macOS: `sw_vers` and `uname -m`. Windows: `cmd /c ver` and `echo %PROCESSOR_ARCHITECTURE%`.

Then check whether a newer Arion already fixes it. Many problems are fixed in a later release.

1. Find the newest published release: `gh release view --repo AstraFoundry/Arion --json tagName,publishedAt` (tags look like `v0.1.2`), or <https://github.com/AstraFoundry/Arion/releases>. If the person's prompt does not name their version, ask them to read it in Settings › About; a "development build" is built from source and has no release number.
2. If their version is older, read the entries between their version and the newest one in `apps/desktop/CHANGELOG.md` (English and Chinese sections) and look for a closed issue about the same problem (`gh issue list --repo AstraFoundry/Arion --state closed --search "<keywords>"`).
3. If a newer release mentions a fix, tell the person which version fixed it and how to update: Settings › About › **Check for Updates**, then install and restart. Ask them to try again after updating and stop here unless the problem remains. Even without a matching fix, ask them to update first when they are several releases behind, because the maintainers can only fix the current release.
4. Write the version they actually tested into the report.

Before writing anything up, look for an existing report:

```sh
gh issue list --repo AstraFoundry/Arion --state all --search "<two or three keywords>"
```

If one matches, show it to the person. Offer to add their details as a comment on it instead of opening a new issue.

## Step 3: Gather and clean the logs

Arion keeps short local support logs. By design they contain no audio, transcripts, vocabulary, window titles or keys, but check them anyway before anything leaves the computer.

- macOS: `~/Library/Application Support/Arion/logs/events.0.jsonl` (newest) to `events.4.jsonl`
- Windows: `%APPDATA%\Arion\logs\events.0.jsonl` to `events.4.jsonl`
- A development build uses an `Arion Dev` folder instead of `Arion`.

The files are JSON Lines. Logs are kept for about seven days at most, so ask the person to reproduce the problem first if it happened long ago.

1. Read only the entries from around the time the problem happened (each entry has a timestamp). Keep the excerpt short: the failure and the few dozen lines before it.
2. Redact anything personal you find: user names in paths, email addresses, host names, IP addresses, account or device identifiers, tokens and keys. Replace each with a placeholder such as `<user>` or `<redacted>`.
3. Never include recordings, transcripts, dictated or typed text, vocabulary, clipboard contents, `preferences.json`, credentials or anything outside the `logs` folder.
4. Show the cleaned excerpt to the person and ask whether it may be attached.

If a longer log is needed, the person can export a full archive from Settings › About › **Export logs…**. It writes a ZIP that is already filtered; they can drag it into the issue page in their browser. Do not upload it yourself.

## Step 4: File the issue

Draft the issue and show the whole draft to the person:

- **Title**: one specific sentence, for example "Dictation stops after a long pause in Microsoft Word".
- **Body**, in English so the maintainers and other users can search it, with the person's own words quoted in their language when that keeps the meaning:

~~~markdown
## What happened
## Steps to reproduce
## Expected
## Environment
- Arion:
- OS and architecture:
- Features involved:
## Logs
<details><summary>Redacted log excerpt</summary>

```jsonl
...
```

</details>

Reported with the help of an AI agent (<model name>).
~~~

After they approve, create it:

```sh
gh issue create --repo AstraFoundry/Arion --label bug --title "<title>" --body-file <file>
```

Use `--label enhancement` instead for a feature request. If `gh` is missing or not signed in, help the person install it and run `gh auth login`, or, if they prefer, open `https://github.com/AstraFoundry/Arion/issues/new` in their browser and give them the title and body to paste.

Give the person the issue link.

## Step 5: Offer to fix it

Skip this step if Step 1 ruled out a fix.

If the person chose "report only" in Step 1, ask now, once and without pressure, whether they would like to help fix it with you. If they decline, go to Step 6.

To fix it:

1. Fork and clone with `gh repo fork AstraFoundry/Arion --clone`, then create a branch such as `fix/<short-topic>`.
2. Read `AGENTS.md` at the repository root and the documents it routes you to before changing anything. They define the architecture, the specifications, the checks and the pull request format. Follow them over any habit of yours.
3. Find the cause, not just the symptom. Keep the change small and focused on this issue.
4. Run the checks the project documentation asks for. Native tests only run on their own operating system; say which checks you could not run and why.
5. Commit with a Conventional Commit message such as `fix(dictation): keep the session alive after a long pause`.
6. Show the person the diff summary and the pull request text, then, after they approve, push and open the pull request against `main`, with `Fixes #<issue number>` in its description. Describe only results you actually verified.

Give the person the pull request link and explain that a maintainer will review it and may ask for changes.

## Step 6: Wrap up

Thank the person, recap what was filed (and fixed), and ask whether they would like to join the Arion community to follow the issue and talk to the team:

- Discord: <https://discord.com/invite/72kh465mMm>
- Feishu (Lark) group: <https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae>

Offer the Feishu group first to people writing in Chinese and Discord first to everyone else.
