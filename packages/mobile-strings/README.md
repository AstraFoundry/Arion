# Mobile strings

`strings.json` holds every user-facing string of the iOS and Android apps, in English (the default) and Simplified Chinese. Keys are dotted lowerCamel segments; placeholders are `{name}` and may move between languages. `pnpm mobile:generate` writes the iOS String Catalog and Swift accessors and the Android string resources; `pnpm mobile:check` fails when they differ. See [mobile apps](../../docs/project/mobile.md).
