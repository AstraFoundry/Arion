# Competitor text insertion

Scope: installed macOS Chatterfly voice-result delivery and Doubao IME commit paths. This is shipped-artifact/static evidence, not a live Lark or Codex acceptance test. The [ledger](evidence/competitor-text-insertion.json) distinguishes actual REA Evidence IDs from supplemental byte-verified LLVM observations. Competitor implementation code is not an Arion dependency.

## Targets and identity

- Chatterfly `1.0.8.13459` / build `13459`, installed in `/Library/Input Methods/Chatterfly.app`: universal SHA-256 `595e502f74e41e1b28904affa9093974f6cbc197a3756a74a007ef562f9ee767`; arm64 SHA-256 `3406894676fd6b9028795858907d1821f54977aa0f9ec5716d42eba12848fe21`. This differs from the older installer payload: older function addresses cannot establish this version's behavior.
- DoubaoIme `1.0.2 (1)`, installed in `/Library/Input Methods/DoubaoIme.app`: universal SHA-256 `4b3836dcb8a3e06b01b4c95390c757b71f623c1f7c4141c059e49284f7f6442d`; arm64 SHA-256 `266f49f6bcb62e43c5170620d4b86e2e4c4ef873bcce624ef1a90b9c46244983`. Current REA Mach-O identity matches the retained native decompilation and keyboard-chain evidence.

REA `6.4.0`, Ghidra `12.1.4`, OpenJDK 21 and the existing installed LLVM tools were used. Full records and expanded code stay in `.workspace/research/insertion-comparison/`. No competitor was launched or attached; no editor text was sent, user audio recorded or traffic intercepted.

Fresh REA metadata records bind the installed artifacts (Chatterfly Mach-O `ev_c20f00868862c08f514a8440bd2e1927ffb146145953921eca23ce5d31f5ef47`, plist `ev_e26a35ae14f63e007da0f3e64f99958cb423a76c8fc29140b62d7c1f2536b7f1`; Doubao Mach-O `ev_66516fc55867516031ba4c3c21cfcfd36c135cf6b7e3d1a9cbc94dbd80198f0e`). These records establish identity/metadata, not delivery control flow. New native dossiers did not complete: Doubao's Ghidra analysis produced address-space/decompiler errors; Chatterfly remained in startup analysis and was stopped. Owned analyzer processes/resources were cleaned up; native `close_binary` did not complete, while the separate Chatterfly metadata session closed normally. Chatterfly control flow below comes from current-byte LLVM observations (all 15 function interval hashes independently rechecked); Doubao's decompiled claims reuse matching retained REA records. No fresh native decompilation is claimed.

## Delivery comparison

| Question | Chatterfly current installed version | Doubao retained matching version |
| --- | --- | --- |
| Ordinary own-IME commit | Effective IMK sender receives `insertText:replacementRange:` | Shared insertion routine reaches client `insertText:replacementRange:`; keyboard caller explicitly supplies `isASR = false` |
| Voice while another input method is active | Explicit `commitVoiceTextViaSystemPaste:` path | Complete external-IME policy not recovered |
| Compatibility paste | Preselected built-in Chromium browser IDs plus configurable extra bundle IDs | Guarded delayed Command-V transaction exists; full direct-versus-paste predicate remains unresolved |
| AX role | Editable focus and post-paste value/text-marker verification in traced methods | Context, target/implementation classification and ownership surfaces; AX setter imports alone do not establish selected-text insertion |
| Unicode text injection | Not present in the traced commit methods; Command-V is an actual V-key shortcut | No Unicode text injection established in the traced paths; bounded negative findings are not a whole-program absence claim |
| Uncertain delivery | Readback has success/missing/unknown states; missing is checked twice before showing the owned result panel | Posted paste reports unknown host acknowledgement; host-commit resolution can select zero writes |

## Chatterfly: mode, preselected paste and readback

The current machine-code chain is `SGMacSpeechService.commitFinalSpeechText` (`0x1003ce898`) → runtime `commitVoiceText:` (`0x1002c9e5c`) → `SGInputHelper.commitVoiceText:sender:` (`0x1002e2c50`). The first method checks own-IME state and editable focus; unsupported/no-focus cases show a result panel. Another active IME selects system paste instead of pretending to own that editor's IMK client.

The helper resolves a missing sender from the current controller's effective sender and requires nonempty text and a sender. Normal own-IME delivery calls `insertText:replacementRange:`. Paste selection (`0x1002e2e54`) tests nine built-in browser bundle IDs, then `speech_commit_text_config.commit_via_paste_bundles`. The built-in list contains Chrome variants, Chromium, Edge variants, Brave and Vivaldi. It does not itself establish Lark/Codex compatibility; current configured extras were not read.

The selected own-IME paste helper (`0x1002e3384`) checks Accessibility permission, creates a single-space marked range, obtains/clears that range, then invokes its clipboard helper. Missing permission or a clipboard-write failure takes a direct IMK insertion branch. That fallback is an observed competitor choice, not an Arion requirement.

Clipboard delivery (`0x1002e36a0`) snapshots items, writes text, then `sg_postCommandV` (`0x1002e3cfc`) posts virtual key `9` with Command flags. It schedules restoration after 250 ms. The scheduling block and restore helper were inspected together: that bounded chain restores unconditionally, with no observed `changeCount` guard. This does not establish the safety of every clipboard operation elsewhere in the app.

After the non-own-IME system-paste route, verification (`0x1003b9304`, `0x1003b9624`) checks the expected foreground app and reads the focused field's AX value/text-marker content. Presence yields success; unreadable/inconclusive content yields unknown. Readable editable content missing the text triggers a second observation (350 ms, then 450 ms), then a same-session result panel. It does not blindly reinsert. These checks do not prove that a matching substring was newly inserted or that insertion occurred exactly once; the inspected own-IME direct route is not verified by this caller.

## Doubao: final commit as a transaction

Matching retained REA decompilation provides stronger evidence for the paste transaction than diagnostic strings alone:

- `DD-paste-gate`, `ev_7813c9cf5cfd974f88db927e1d3ec7e857a490b646f2b4d8968d44ded8cdb348`, entry `0x10017dce4`: delayed Command-V requires the same session/generation/sequence, a paste in flight, no posted Command-V and zero previous final-text writes. Target and clipboard failures have distinct exits. Excel can wait for mouse release. Posted paste is logged with unknown host acknowledgement.
- `DD-paste-handoff`, `ev_f83fcb7dbd1d806506760b9dcd8270e70e79ed45279a4066d91fe105c5465bac`, entry `0x100176dd8`: handoff cancels pending restoration and compares current pasteboard `changeCount` with its owned count; intervening changes prevent that handoff. This is the inspected handoff path, not proof of every restore branch.
- `DD-commit-strategy`, `ev_3af76376fc6472a824bed1678d3f5b5a2975d70d8a2733140195b4be57bbb3c7`, entry `0x1001ab16c`: host resolution distinguishes `hostCommitted`, `imeCommitRequired`, `ambiguous` and `contextInvalid`. An already committed original can select `skipSystemCommitted` with zero writes.

The same function reads implementation names `web`, `qt`, `nsTextView`, `nsTextField`, `customMultiline`, `customDrawn` and `unknown`. This establishes classification state, not the algorithm that assigns it or a complete routing table. The [byte-verified shared insertion chain](doubao-keyboard-commit.md) reaches `insertText:replacementRange:` at `0x10018eaf4` from routine `0x10018dbac`. Complete ASR coordinator routing between direct commit and paste remains unresolved.

## Implications for Arion

The transferable distinction is between an owned IMK text client, external editor delivery and acknowledgement. IMK `insertText` is not an `AXSelectedText` attribute write. Chatterfly's non-own-IME system-paste branch is the closest observed mode comparison to Arion's global shortcut dictation.

Neither traced implementation establishes Unicode keyboard injection as the general solution for every web field. Arion now uses the separate ownership decision in [ADR 003](../decisions/003-insertion-routes.md): IMK for its own client and guarded system paste for macOS shortcut delivery. Actual rich-editor acceptance remains necessary.

A delivery design should explicitly select an applicable route before writing, preserve the target/client and generation, prevent duplicate writes, protect clipboard ownership and keep uncertain delivery actionable. The approved macOS system-paste policy is documented separately in [ADR 003](../decisions/003-insertion-routes.md); this static investigation does not establish runtime compatibility.

## Coverage and remaining unknowns

Answered statically: current Chatterfly own-IME versus system-paste branch, built-in/configurable paste policy, actual Command-V API calls and readback outcome handling; matching Doubao IMK insertion surface, delayed-paste gates, handoff ownership and already-committed suppression.

Partially answered: Doubao complete direct/paste routing and text-implementation classification; Chatterfly active runtime configuration and all direct-route acknowledgement paths. Unobserved: both products' actual execution in Lark/Codex, current permissions, renderer acceptance, selection fidelity and end-to-end exactly-once behavior. Shipped strings, static calls and import absence must not be presented as a successful host test.
