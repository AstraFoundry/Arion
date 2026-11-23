# ADR 053: Rime typing layouts in the mobile keyboards

Status: Accepted for iOS. Android follows with the same engine and layouts. Amends [ADR 051](051-native-mobile-keyboards.md), whose keyboards were voice only.

## Context

A voice keyboard still needs typing for names, codes and quiet places; switching to another keyboard breaks the flow. Typeless 2.11.0 ships Rime in its keyboards with full-keyboard and 9-key Pinyin as a setting ([research](../research/typeless-mobile-typing.md)). The desktop already pins librime 1.17.0, the `pinyin_simp` dictionary and an encrypted learning store.

## Decision

- **Engine.** The phones use the desktop's pinned librime and its correction patch. Its dependencies are built from pinned sources as static libraries; schemas are prebuilt on the build machine. The phone never deploys or downloads typing data.
- **Layouts.** Full-keyboard Pinyin (`arion_pinyin`), 9-key Pinyin (`arion_pinyin_9key`, digits derived from the same dictionary, learning shared) and English inserted as typed. The keyboard's 中/EN key toggles Chinese and English only; which Chinese layout is a setting in the app.
- **Switching.** The switch key is the bottom-left key of the voice and typing panels, which share one height, so its place is learned once. The keyboard reopens in the last mode.
- **Learning.** Rime's user dictionary goes through the desktop's encrypted database with a POSIX file provider shared by macOS, iOS and Android. The iOS key is AES-256-GCM in the keychain, device-only, and never recreated over existing learning.

## Consequences

- The keyboard extension grows by about 9 MB of code and 1.9 MB of data; the iOS build needs `pnpm mobile:rime` first.
- No sentence model, English completion or expandable candidate grid yet, all of which Typeless has.
- The desktop's macOS learning adapter now uses the shared POSIX provider.

## Alternatives considered

- **System or third-party typing SDKs.** None are offline, embeddable and license-compatible with our pinned data.
- **A 中/EN/9-key cycle.** Rejected by the product owner: the toggle must be stable, and the layout is a preference.

## References

- [012 Mobile input](../../specs/012-mobile-input.md), MOB-15
- [Mobile apps](../project/mobile.md)
