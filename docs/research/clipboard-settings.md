# Clipboard retention settings

## Primary-source observations

| Project | Exposed controls | Evidence |
| --- | --- | --- |
| Maccy | A history-size number field and stepper, accepting 1–999. Current storage usage is displayed alongside it; the inspected pane does not expose an editable byte budget or age limit. | [StorageSettingsPane.swift](https://github.com/p0deje/Maccy/blob/master/Maccy/Settings/StorageSettingsPane.swift), `size`, `sizeFormatter`, `Stepper` and read-only `storageSize`. |
| Ditto | General options expose maximum saved copies and expiration age, each with a corresponding enable checkbox. | [OptionsGeneral.cpp](https://github.com/sabrogden/Ditto/blob/master/src/OptionsGeneral.cpp), `IDC_MAX_SAVED_COPIES`, `IDC_EXPIRE_AFTER`, `GetCheckForMaxEntries`, `GetCheckForExpiredEntries` and their setters. |
| EcoPaste | Preferences expose maximum history count, retention days and an independent cleanup interval in hours. The visible retention control fixes the unit to days; 0 disables age cleanup. Although the backend model supports other units, that does not establish a UI selector for them. | [Preference schema](https://github.com/EcoPasteHub/EcoPaste/blob/master/src/pages/Preference/config/preferenceSchema.ts), `history.retention`, `history.maxCount`, `history.cleanupIntervalHours`; [RetentionControl.tsx](https://github.com/EcoPasteHub/EcoPaste/blob/master/src/pages/Preference/components/settingControls/RetentionControl.tsx). |

These are source-code observations, not execution of installed competitor products. They establish that exposing retention choices is common in these projects, not that Arion needs all of their settings or defaults.

## Arion decision

Expose only ordinary-history count and retention age through preset selectors, below the enabled collection switch. Retain existing custom values when displaying saved preferences. Keep byte-budget and per-capture safeguards internal. Defaults remain 200 ordinary entries, 30 days, 100 MiB total payload and 10 MiB per capture. Pins bypass age/count expiry but consume the storage budget.

Use the dictation native key recorder and disclosure. The current simplified picker exposes no destructive management; cleanup is automatic, and pin/unpin remains available on hover. The comparison motivates the simplification; it does not copy either product's interface or broaden Arion's scope.

## EcoPaste activation and fullscreen behavior

Primary source inspection of EcoPaste's `master` branch establishes a different activation boundary from Arion's earlier caret-only picker:

- [position.rs](https://github.com/EcoPasteHub/EcoPaste/blob/master/src-tauri/src/window/position.rs) positions by **mouse cursor**, current-monitor center or remembered coordinates. It does not require a focused text field to open history. Inspected blob: `ec5fcb8b75bb2d9ab75d953b373d15c7e8afbefe`.
- [window/macos.rs](https://github.com/EcoPasteHub/EcoPaste/blob/master/src-tauri/src/window/macos.rs) uses a nonactivating NSPanel that can become key without becoming the main window; it configures fullscreen auxiliary/Space behavior and hides on resign-key. Inspected blob: `75e4d931e585a912edc960b9b4fb349ac59dc70c`.
- [keystroke/macos.rs](https://github.com/EcoPasteHub/EcoPaste/blob/master/src-tauri/src/keystroke/macos.rs) posts Command-V through CGEvent. Its permission requirement applies to automatic paste, not to finding caret geometry for panel activation.

These are source observations, not execution of EcoPaste or a guarantee for every editor. Arion retains MyGo native UI and its target/security validation, but separates history activation from caret support: prefer input geometry, otherwise open beside the mouse in copy-only mode. Enter, the return icon and row clicks paste into a verified captured input; otherwise they copy. The macOS panel must receive keyboard input and join the active fullscreen Space without activating Arion.
