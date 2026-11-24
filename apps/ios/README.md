# Arion for iOS

The Arion app and its voice keyboard extension, in UIKit ([ADR 051](../../docs/decisions/051-native-mobile-keyboards.md), [012 Mobile input](../../specs/012-mobile-input.md)). Layout, flow and verification status are in [mobile apps](../../docs/project/mobile.md).

```bash
pnpm ios:test
```

Generates `Arion.xcodeproj` from `project.yml` with XcodeGen and runs the ArionKit tests on the simulator. Open the generated project to run the app. Do not edit `ArionKit/Sources/ArionKit/Design/SunbeamTokens.swift`, `Localization/L10n.swift` or `Resources/Localizable.xcstrings`; `pnpm mobile:generate` writes them.

The keyboard's microphone capture follows [stephancill/ios-keyboard-dictation](https://github.com/stephancill/ios-keyboard-dictation) (MIT).
