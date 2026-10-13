# UX reference: Typeless and Doubao IME

Scope: onboarding, menu bar presence, settings structure, the floating dictation surface, model management and insertion strategy. This complements the pipeline investigations in [Typeless](typeless.md) and [Doubao](doubao.md). It records **observed shipped-artifact evidence** and **Arion design choices** separately; nothing here is copied code or copy.

## Evidence

| Target | Method | Evidence |
| --- | --- | --- |
| Typeless 2.8.1 `app.asar` (SHA-256 `eb9bcc84…c80b`) | REA `analyze_javascript_application` (record-and-continue) | Application graph `ev_dd28054f9b74345ad14dfa2269d7ce1ec66e4ce543da17da050a3d60cec9adcd` |
| Typeless onboarding surface | REA `trace_application_feature`, seed `onboarding` | `ev_9c42a0d9540c1ace96bd8e1b96b2b6e57010d020f1ee30e8e9b9a12677bb85ab`: a separate `onboarding.html` renderer, two renderer modules, onboarding copy-config and progress routes |
| Typeless renderer windows and copy keys | Inert reading of the extracted bundle | Renderer entries `floating-bar.html`, `hub.html`, `onboarding.html`, `sidebar.html`, `interactive-card.html`, `login.html`; English locale key namespaces |
| Doubao IME 1.0.2 `Info.plist` | REA `inspect_plist` | `ev_6429b2b9888383020d5a2a363a70917fbb568aea466a42413a6e11e52613d0a0`: background-only IMK server, separate settings and installer bundle IDs, microphone and Bluetooth usage strings |
| Doubao settings and IME binaries | `strings` over the Mach-O files | Type names only; no call graph, ordering or runtime behavior is claimed from them |
| Doubao IME main executable (arm64) | REA 6.4.0 + Ghidra 12.1.4 decompilation | Five functions: paste gate, clipboard handoff, host-commit resolution, fn long-press profile ([ledger](evidence/doubao-decompiled.json), [findings](doubao.md#decompiled-insertion-and-shortcut-paths)) |

No app was launched, no account or user data was read, and no network traffic was observed. Locale strings and type names show what surfaces exist, not how often or under which conditions they appear.

## Findings

### Onboarding

- **Typeless (observed copy keys):** three phases (`sign up`, `set up`, `experience it`). Setup asks for microphone then Accessibility, each with a one-line purpose and a privacy explanation; a **hotkey test** ("do you see the button turning blue while pressing?") and a **microphone test** ("do you see the bars moving while you speak?") each offer "yes, continue" or "change". A guided practice dictates sample messages into an in-window textbox before a final "Speak, don't type" page. Account creation, acquisition surveys, pricing and invite codes are part of its flow.
- **Doubao (observed type names):** `OnboardingWindow`/`OnboardingManager`, `MicrophonePermissionPage`, `TrySpeakingPage`, a `GlobalVoiceGuideModal` with `GlobalVoiceGuideStep`s, and a Lottie-driven `VoiceGuideCarousel` with a transition direction type.
- **Arion choice:** Welcome → Microphone (with a live level meter as the test) → Accessibility (auto-advances when granted in System Settings) → Model (shows the automatically chosen bundle, why, and download progress) → Shortcut (press-to-verify) → Try it (a practice text area) → Done. No account, survey or pricing steps: Arion is local-first and has none of those.

### Menu bar and windows

- **Typeless:** a tray menu with home, history, settings, microphone selection, "add word to dictionary", update status, version and quit; a separate frameless floating bar window for status.
- **Doubao:** the input method is background-only; settings is a separate UI-element app; an `ASRPanelController`/`BaseCapsulePanel` family and a `MicrophoneMeterModel` back the voice panel.
- **Arion choice:** a menu bar icon whose click opens a native popover panel (status, start/finish button with the shortcut, last dictation with copy, active model and its state) and whose right-click opens a short menu. The dictation pill lives in its own transparent, click-through, always-on-top window that never takes focus.

### Settings

- **Typeless:** sections for general (appearance, launch at login, dock icon, hardware acceleration), audio (microphone, interaction sounds, muting other audio), keyboard shortcuts with conflict and reserved-key validation, languages, personalization and history sync.
- **Doubao:** a sidebar (`SettingsSidebarButton`, `SettingsTab`) of grouped sections (`SettingsSection`, `SettingSwitch`, single/multi-select groups), a shortcut recorder with conflict detection (`InputSourceShortcutConflictDetector`), feature highlights, and an offline-model area with `DisableOfflineVoiceModal`.
- **Arion choice:** sidebar sections General (permissions, launch at login, sounds), Dictation (shortcut recorder rejecting modifier-less keys and taken shortcuts, an optional hold-to-talk key modeled on Doubao's separate fn long-press profile (`DD-shortcut-fn`), insertion method, filler removal), Microphone (device picker and live meter), Model, Vocabulary (encrypted approved replacements), Privacy and About.

### Floating surface and motion

- **Typeless:** the floating bar computes bar heights from Web Audio `getByteFrequencyData` and uses short cubic-bezier transitions; its alerts cover permissions, device problems, network failures, session limits, cancelled transcripts and "press again to finish" hints.
- **Doubao:** capsule panels, Lottie animations, banner animation state and progress-fill capsule buttons.
- **Arion choice:** a dark capsule with a pulsing live dot, input-level bars eased by a spring, the streaming draft (tail-truncated), the shortcut to finish; a three-dot wave while transcribing; a drawn check for verified insertion; a shake-in warning with a title and a next step for failures. Springs keep velocity so the pill resizes smoothly as text streams; reduced motion is honored by MyGo.

### Model management

- **Doubao (observed type names):** an offline model manager with manifest, active layout, file store, path resolver, storage health/repair and download stages and failure reasons.
- **Arion choice:** the catalog pins every artifact's URL, SHA-256, size, member digests and licenses; downloads resume, are size-bounded, verified and published atomically, and the previous verified install stays until a replacement verifies. The **default bundle is chosen automatically** from measured evaluations ([006 Models](../../specs/006-models.md)); the user can still pick another evaluated bundle.

### Insertion

- **Typeless:** an insert-text IPC to a native helper, accessibility context and paste-related surfaces; its permission copy says Accessibility lets it "paste text into any textbox".
- **Doubao:** IMK `insertText`, synthetic Command-V diagnostics and clipboard-restoration guards, delayed-paste cancellation and target ownership rejection diagnostics. Decompiled: Command-V is posted only while session, generation and commit sequence still match and nothing was written yet; the clipboard is handed back only if `changeCount` is unchanged; Excel waits for mouse release; a host that already committed gets zero writes (`DD-paste-gate`, `DD-paste-handoff`, `DD-commit-strategy`).
- **Arion choice:** see [ADR 003](../decisions/003-insertion-routes.md): one route chosen at capture time (Accessibility for native editors, Unicode keystrokes for Chromium/Electron editors, or user-selected paste with guarded clipboard restore), focus revalidation before the single commit, and no automatic second attempt.

## Limits

Static strings cannot show default settings, ordering, timing, retry policy or which branch a session takes. Doubao behavior claims are limited to the decompiled functions in the ledger; everything else rests on type names.
