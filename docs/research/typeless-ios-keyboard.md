# Typeless iOS keyboard

## Target and scope

App Store IPA `Typeless-2.11.0.ipa`, SHA-256 `07a252b50b15a7d22ca387c00634b04f889e61f15a96649c20381ed785cd83eb`: app `com.typeless.mobile` 2.11.0 (build 6), keyboard extension `com.typeless.mobile.keyboard`, widget extension `com.typeless.mobile.dynamicisland`. The archive was extracted into the private evaluation workspace and read with `plutil` and `strings`; nothing was executed. Everything below is **static evidence** of what the bundle contains, not verified runtime behavior.

## Observations

- **Architecture.** Both the app and the keyboard run React Native: the keyboard extension carries its own Hermes bytecode bundle (`main.jsbundle`, version 96, 4.47 MB) and React Native support bundles.
- **Keyboard declaration.** `NSExtensionPointIdentifier = com.apple.keyboard-service`, principal class `keyboard.KeyboardViewController`, `RequestsOpenAccess = true`, `PrimaryLanguage = mis`, `IsASCIICapable = false`.
- **Recording in the app.** The app declares `UIBackgroundModes` `audio`, `NSSupportsLiveActivities` and frequent Live Activity updates, and a widget extension named `dynamicisland`. With the URL scheme `typeless`, this is consistent with the common pattern of recording in the containing app while a Live Activity shows the session; the IPA alone does not show the switching sequence.
- **Keyboard gestures.** Keyboard strings include “Tap to speak”, “Release to finish”, “Release to cancel”, “Release to translate”, “Swipe to translate”, “Swipe here to translate”, “Release to edit”, “Speak to edit”, “Speak without switching keyboards” and “Swipe to switch keyboards”. Together they indicate both tap and hold dictation, with release targets for cancel, translate and voice editing.
- **Permissions and tracking.** `NSMicrophoneUsageDescription` is present; so is `NSUserTrackingUsageDescription` for marketing attribution, with Adjust, Firebase and Sentry bundles.

## Arion design choices

These are Arion decisions informed by, not copied from, the evidence ([ADR 051](../decisions/051-native-mobile-keyboards.md)):

- Arion records inside the keyboard through WebKit instead of the containing app, so dictation does not leave the editor.
- Tap and hold both dictate; holding offers cancel and translate as release targets ([012](../../specs/012-mobile-input.md) MOB-01). Voice editing of a selection is not implemented.
- No tracking or attribution SDKs.

## Limitations

The Hermes bundle was searched for strings only, not decompiled; the Live Activity, audio session and inter-process messaging were not traced. The broader REA analysis of this IPA, and of other competitors, was started in the private workspace and did not finish.
