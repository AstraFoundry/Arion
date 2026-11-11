// Writes i18n/testdata/relative_time_icu.json.gz with Node's upstream ICU.
// Run after updating CLDR: node scripts/relative-time-golden.ts
// Bun on macOS uses Apple's ICU; use Node with CLDR matching i18n's.
import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { gzipSync } from "node:zlib";

const dir = join(import.meta.dirname, "..", "i18n", "testdata");
const { locales } = JSON.parse(await readFile(join(dir, "format_locales.json"), "utf8")) as { locales: string[] };
const units = ["second", "minute", "hour", "day", "week", "month", "quarter", "year"] as const;
const cases: { value: string; unit: number; style: number; auto: boolean }[] = [];
const styles = ["long", "short", "narrow"] as const;
for (const [style] of styles.entries()) {
  for (const auto of [false, true]) {
    for (const [unit] of units.entries()) {
      for (const value of ["-12345.6", "-1.2345", "-21", "-11", "-5", "-2", "-1", "-0.005", "-0.0049", "-0.0001", "-0", "0", "0.0001", "0.0049", "0.005", "0.5", "0.995", "1", "1.0001", "1.2345", "1.2355", "1.0049", "1.005", "2", "3", "5", "11", "21", "1234", "12345.6"]) {
        cases.push({ value, unit, style, auto });
      }
    }
  }
}
const rows: Record<string, string[]> = {};
// Extensions check the same digit selection as the number formatter;
// unknown and POSIX locales are covered on the Go side.
for (const locale of [...locales, "ar-u-nu-latn", "fa-u-nu-latn", "bn-u-nu-latn", "en-AU", "zh-TW"]) {
  const formatters = styles.flatMap((style) => [false, true].map((auto) =>
    new Intl.RelativeTimeFormat(locale, { style, numeric: auto ? "auto" : "always" })));
  rows[locale] = cases.map((c) => formatters[c.style * 2 + Number(c.auto)]!.format(Number(c.value), units[c.unit]!));
}
const golden = { icu: process.versions.icu, cldr: process.versions.cldr, cases, locales: rows };
// The full cross-product is compressed to keep the fixture small.
await writeFile(join(dir, "relative_time_icu.json.gz"), gzipSync(JSON.stringify(golden) + "\n", { level: 9 }));
console.log(`${Object.keys(rows).length} locales, ${cases.length} cases each`);
