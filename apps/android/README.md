# Arion for Android

The Arion app and its voice input method, in Kotlin and Jetpack Compose ([ADR 051](../../docs/decisions/051-native-mobile-keyboards.md), [012 Mobile input](../../specs/012-mobile-input.md)). Layout, flow and verification status are in [mobile apps](../../docs/project/mobile.md).

```bash
pnpm android:test
```

Runs the `core` tests and builds the debug app. Modules: `core` (domain, no Android dependency), `design` (Sunbeam for Compose and the generated string resources), `app` (activity, input method, recognizer and model adapters). Do not edit `design/src/main/kotlin/.../SunbeamTokens.kt` or `design/src/main/res/values*/strings.xml`; `pnpm mobile:generate` writes them.
