# Vocabulary UI evidence

These synthetic fixtures cover MEM-A22/A24. `manifest.json` maps original capture paths to archived bytes and SHA-256 identities. It retains the two sandbox-blocked attempts separately from the successful unrestricted native run; stderr from those blocked attempts was returned in tool history and is not available as an archived byte stream.

## Reproduce

From the repository root, set a new output directory for each run:

```sh
ARION_UI_SNAPSHOTS=/private/tmp/arion-hotword-ui GOCACHE=/private/tmp/arion-hotword-go-cache go test ./apps/desktop/ui/app -run 'TestStandaloneHotwordFormAndRemoval|TestVocabulary' -count=1
ARION_NATIVE_E2E=1 ARION_E2E_OUTPUT=/private/tmp/arion-hotword-native GOCACHE=/private/tmp/arion-hotword-go-cache go -C apps/desktop test -tags arion_e2e -run '^TestVocabularyNative' -count=1 -timeout=120s -v ./e2e
```

The native command requires an unrestricted interactive macOS session. It hosts the production composition root and windows in an isolated temporary profile, invokes production add/remove actions, checks independent record identities, captures both locales and confirms removing a hotword retains its alias. The native tests passed in 10.637 seconds. The native action test uses the normal OS credential-backed encryption path and only writes synthetic vocabulary in that temporary profile; it does not change the user's vocabulary file.

Headless tests dispatch form and removal clicks through MyGo's tester, cover both locales and retain light/dark hotword form images. The full `ui/app` suite separately passed in 213.818 seconds. These frames are rendered test artifacts, not desktop screenshots.

Native `CapturePage` images establish actual window lifecycle and Content rendering. Native material transparency can appear black in these captures; they are not OS compositor screenshots. The native test invokes production actions directly and does not dispatch physical mouse or keyboard input to the native controls. Headless interaction tests supply the control-binding evidence. No Windows native execution, microphone, ASR accuracy, hotword benefit or false-bias result is established here.

## PR image hosting

PNG bytes are kept on the image-only branch `claude/pr-screenshots-vocabulary-hotwords` under `pr-screenshots/`, rather than on the feature branch. `manifest.json` records each URL, branch path and original SHA-256; the native log remains in this archive. Fetch that branch and use `git show origin/claude/pr-screenshots-vocabulary-hotwords:pr-screenshots/<filename>` to recover an image, then verify its recorded digest. The image branch may be deleted after merge per repository policy; retain a local copy first if continued image evidence is required.
