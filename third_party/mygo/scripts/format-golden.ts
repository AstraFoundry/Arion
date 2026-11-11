// Writes i18n/testdata/format_icu.json: how ICU (Intl.NumberFormat,
// Intl.DateTimeFormat, Intl.DisplayNames, Intl.Locale) writes numbers,
// compact ones and amounts of money too, dates and times, ranges of them
// (formatRange), the
// first day of the week and the hour cycle, and
// the names of languages in themselves, in every locale package i18n has formatting
// data of, which TestFormatMatchesICU checks package i18n against. Run it
// again after updating CLDR, with a Node.js whose ICU has the same CLDR
// (`node -p process.versions.cldr`):
//
//   node scripts/format-golden.ts
//
// Node.js, not Bun: Bun on macOS uses the system's ICU, whose data Apple
// changes.
//
// Dates are in the Gregorian calendar (-u-ca-gregory), which package i18n
// formats in, and in UTC unless a case names a zone. The locales are those
// of i18n/testdata/format_locales.json, which `go generate ./i18n` writes.
import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";

const dir = join(import.meta.dirname, "..", "i18n", "testdata");
const { locales, autonyms } = JSON.parse(await readFile(join(dir, "format_locales.json"), "utf8")) as {
  locales: string[];
  autonyms: string[];
};

type NumberCase = { kind: "number"; value: string; options: Intl.NumberFormatOptions & { roundingMode?: string } };
type DateCase = { kind: "date"; time: string; zone?: string; options: Intl.DateTimeFormatOptions };
type NamesCase = { kind: "names"; times: string[]; options: Intl.DateTimeFormatOptions };
type RangeCase = { kind: "range"; time: string; end: string; options: Intl.DateTimeFormatOptions };
type Case = NumberCase | DateCase | NamesCase | RangeCase | { kind: "firstDay" | "hourCycle" };

const cases: Case[] = [];

// Numbers.
const decimals = ["0", "1", "-1", "1.5", "0.001234", "1234.5", "-1234567.891", "12345", "123456789", "1000000",
  "0.5", "2.5", "1.0005", "99999.9999", "123456789012345680000", "-0.0001"];
for (const value of decimals) cases.push({ kind: "number", value, options: {} });
for (const value of ["0", "0.256", "-0.5", "1.5", "12.345", "1234.56"]) cases.push({ kind: "number", value, options: { style: "percent" } });
for (const value of ["1", "1234.5", "-0.125"]) {
  cases.push({ kind: "number", value, options: { minimumFractionDigits: 2, maximumFractionDigits: 2 } });
}
for (const value of ["2.5", "3.5", "-2.5"]) {
  cases.push({ kind: "number", value, options: { maximumFractionDigits: 0, roundingMode: "halfEven" } });
}
cases.push({ kind: "number", value: "1234567.891", options: { useGrouping: false } });
cases.push({ kind: "number", value: "1234.5", options: { minimumFractionDigits: 3 } });
// Compact numbers, as a chart's axis labels them.
const compacts = ["0", "999.9", "1234", "-1234", "12345", "123456", "999999", "1234567", "-2500000", "1000000000",
  "123456789012", "1000000000000000", "0.001234"];
for (const compactDisplay of ["short", "long"] as const) {
  for (const value of compacts) cases.push({ kind: "number", value, options: { notation: "compact", compactDisplay } });
}
cases.push({ kind: "number", value: "1234567", options: { notation: "compact", maximumFractionDigits: 2 } });
cases.push({ kind: "number", value: "1000", options: { notation: "compact", minimumFractionDigits: 2 } });
// Amounts of money, as a price, a total or a balance shows them: of
// currencies whose fraction digits CLDR and ISO 4217 agree on, as
// JavaScriptCore takes ISO 4217's where V8 takes CLDR's.
const amounts: [string, string[]][] = [
  ["USD", ["1842", "-1842", "0", "1234567.891", "0.005", "-0.001"]],
  ["EUR", ["1842", "-1842.5"]],
  ["JPY", ["1842.5", "-1842"]],
  ["GBP", ["1842"]],
  ["CAD", ["1842"]],
  ["INR", ["1234567"]],
  ["BHD", ["1842.5"]],
  ["CHF", ["-1842.53"]],
  ["XYZ", ["1842"]],
];
for (const [currency, values] of amounts) {
  for (const value of values) cases.push({ kind: "number", value, options: { style: "currency", currency } });
}
for (const currency of ["USD", "CAD", "GBP"]) {
  cases.push({ kind: "number", value: "1842", options: { style: "currency", currency, currencyDisplay: "narrowSymbol" } });
}
for (const value of ["1842", "-1842"]) {
  cases.push({ kind: "number", value, options: { style: "currency", currency: "EUR", currencyDisplay: "code" } });
}
for (const value of ["1", "2", "3", "11", "100", "1.5", "-1842", "1000000"]) {
  cases.push({ kind: "number", value, options: { style: "currency", currency: "USD", currencyDisplay: "name" } });
}
cases.push({ kind: "number", value: "1", options: { style: "currency", currency: "EUR", currencyDisplay: "name" } });
for (const currency of ["USD", "EUR", "JPY"]) {
  cases.push({ kind: "number", value: "-1842", options: { style: "currency", currency, currencySign: "accounting" } });
}
cases.push({ kind: "number", value: "-1842", options: { style: "currency", currency: "EUR", currencyDisplay: "code", currencySign: "accounting" } });
cases.push({ kind: "number", value: "-1842", options: { style: "currency", currency: "USD", currencyDisplay: "name", currencySign: "accounting" } });
cases.push({ kind: "number", value: "1842.5", options: { style: "currency", currency: "USD", maximumFractionDigits: 0 } });
cases.push({ kind: "number", value: "1842.56", options: { style: "currency", currency: "USD", maximumFractionDigits: 1 } });
cases.push({ kind: "number", value: "1842.5", options: { style: "currency", currency: "USD", minimumFractionDigits: 3 } });
cases.push({ kind: "number", value: "1842.5", options: { style: "currency", currency: "JPY", minimumFractionDigits: 1 } });
cases.push({ kind: "number", value: "1234567", options: { style: "currency", currency: "USD", useGrouping: false } });
// Compact amounts, as a chart of money labels its axis.
for (const value of ["999", "1234", "4200000", "-4200000", "1234567890", "1000000000000000"]) {
  cases.push({ kind: "number", value, options: { style: "currency", currency: "USD", notation: "compact" } });
}
cases.push({ kind: "number", value: "4200000", options: { style: "currency", currency: "USD", notation: "compact", compactDisplay: "long" } });
cases.push({ kind: "number", value: "123456789", options: { style: "currency", currency: "JPY", notation: "compact" } });
cases.push({ kind: "number", value: "4200000", options: { style: "currency", currency: "EUR", notation: "compact", currencyDisplay: "code" } });
cases.push({ kind: "number", value: "-123", options: { style: "currency", currency: "EUR", notation: "compact", currencySign: "accounting" } });
for (const compactDisplay of ["short", "long"] as const) {
  for (const value of ["1000000", "4200000"]) {
    cases.push({ kind: "number", value, options: { style: "currency", currency: "EUR", notation: "compact", compactDisplay, currencyDisplay: "name" } });
  }
}

// Dates and times.
const times = ["2026-01-05T13:07:09Z", "2025-12-29T00:07:09Z", "2024-07-14T12:00:00Z", "1999-10-31T06:30:05Z"];
const styles = ["full", "long", "medium", "short"] as const;
for (const time of times.slice(0, 2)) {
  for (const s of styles) cases.push({ kind: "date", time, options: { dateStyle: s } });
}
for (const time of times) {
  for (const s of styles) cases.push({ kind: "date", time, options: { timeStyle: s } });
}
for (const s of styles) cases.push({ kind: "date", time: times[0]!, options: { dateStyle: s, timeStyle: s } });
cases.push({ kind: "date", time: times[0]!, options: { dateStyle: "full", timeStyle: "short" } });
cases.push({ kind: "date", time: times[0]!, options: { dateStyle: "short", timeStyle: "medium" } });
for (const hourCycle of ["h12", "h23"] as const) {
  for (const time of [times[0]!, times[1]!]) {
    for (const s of ["short", "medium"] as const) cases.push({ kind: "date", time, options: { timeStyle: s, hourCycle } });
    cases.push({ kind: "date", time, options: { hour: "numeric", minute: "2-digit", hourCycle } });
    cases.push({ kind: "date", time, options: { hour: "numeric", minute: "2-digit", second: "2-digit", hourCycle } });
  }
}
// The skeletons the controls and messages use, as Intl's options.
const skeletons: Intl.DateTimeFormatOptions[] = [
  { year: "numeric", month: "long" }, // yMMMM, a calendar's title
  { year: "numeric", month: "short" }, // yMMM
  { month: "short", day: "numeric" }, // MMMd
  { month: "long", day: "numeric" }, // MMMMd
  { year: "numeric", month: "numeric", day: "numeric" }, // yMd
  { year: "numeric", month: "short", day: "numeric" }, // yMMMd
  { year: "numeric", month: "long", day: "numeric", weekday: "long" }, // yMMMMEEEEd
  { year: "numeric", month: "short", day: "numeric", weekday: "short" }, // yMMMEEEd
  { year: "numeric", month: "2-digit", day: "2-digit" }, // yMMdd
  { year: "numeric", month: "numeric" }, // yM
  { month: "short", day: "numeric", weekday: "short" }, // MMMEEEd
  { day: "numeric", weekday: "short" }, // EEEd
  { weekday: "long" }, // EEEE
  { month: "long" }, // MMMM
  { year: "numeric" }, // y
  { day: "numeric" }, // d
  { hour: "numeric", minute: "2-digit" }, // jmm
  { hour: "numeric", minute: "2-digit", second: "2-digit" }, // jmmss
  { hour: "numeric" }, // j
  { minute: "2-digit", second: "2-digit" }, // mmss
  { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }, // yMMMdjmm
  { year: "numeric", month: "long", day: "numeric", hour: "numeric", minute: "2-digit" }, // yMMMMdjmm
  { hour: "numeric", minute: "2-digit", timeZoneName: "short" }, // jmmz
];
for (const options of skeletons) {
  for (const time of times) cases.push({ kind: "date", time, options });
}
// Offsets from GMT.
for (const zone of ["Etc/GMT-8", "Etc/GMT+5"]) {
  for (const s of ["full", "long"] as const) cases.push({ kind: "date", time: times[0]!, zone, options: { timeStyle: s } });
}

// Ranges, as a date-range picker labels them: from a time to one of the
// same minute, hour, half of the day, day, month and year, and others.
const rangeStart = "2026-01-05T09:30:00Z";
const rangeEnds = ["2026-01-05T09:30:40Z", "2026-01-05T09:45:00Z", "2026-01-05T11:00:00Z", "2026-01-05T13:07:09Z",
  "2026-01-09T10:00:00Z", "2026-02-09T10:00:00Z", "2027-03-02T10:00:00Z"];
const rangeOptions: Intl.DateTimeFormatOptions[] = [
  { dateStyle: "full" }, { dateStyle: "medium" }, { dateStyle: "short" }, { timeStyle: "short" },
  { dateStyle: "medium", timeStyle: "short" },
  { year: "numeric", month: "short", day: "numeric" }, // yMMMd
  { year: "numeric", month: "long" }, // yMMMM
  { month: "short", day: "numeric", weekday: "short" }, // MMMEEEd
  { hour: "numeric", minute: "2-digit" }, // jmm
  { hour: "numeric", minute: "2-digit", hourCycle: "h23" },
  { hour: "numeric", minute: "2-digit", second: "2-digit" }, // jmmss
  { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }, // yMMMdjmm
];
for (const options of rangeOptions) {
  for (const end of rangeEnds) cases.push({ kind: "range", time: rangeStart, end, options });
}

// The names of months and weekdays, as dates of every month and weekday
// write them: Intl gives no list of them but its patterns'.
const monthTimes = Array.from({ length: 12 }, (_, m) => new Date(Date.UTC(2023, m, 15)).toISOString());
const dayTimes = Array.from({ length: 7 }, (_, d) => new Date(Date.UTC(2023, 0, 1 + d)).toISOString());
for (const month of ["long", "short", "narrow"] as const) {
  cases.push({ kind: "names", times: monthTimes, options: { month } });
  cases.push({ kind: "names", times: monthTimes, options: { month, day: "numeric" } });
}
for (const weekday of ["long", "short", "narrow"] as const) {
  cases.push({ kind: "names", times: dayTimes, options: { weekday } });
}
cases.push({ kind: "names", times: dayTimes, options: { weekday: "long", year: "numeric", month: "long", day: "numeric" } });
cases.push({ kind: "firstDay" }, { kind: "hourCycle" });

const hourCycles: Record<string, number> = { h11: 12, h12: 12, h23: 24, h24: 24 };

const out: Record<string, string[]> = {};
for (const locale of locales) {
  const row: string[] = [];
  for (const c of cases) {
    switch (c.kind) {
      case "number":
        row.push(new Intl.NumberFormat(locale, c.options).format(c.value as unknown as number));
        break;
      case "date":
        row.push(new Intl.DateTimeFormat(`${locale}-u-ca-gregory`, { timeZone: c.zone ?? "UTC", ...c.options }).format(new Date(c.time)));
        break;
      case "range":
        row.push(new Intl.DateTimeFormat(`${locale}-u-ca-gregory`, { timeZone: "UTC", ...c.options }).formatRange(new Date(c.time), new Date(c.end)));
        break;
      case "names": {
        const f = new Intl.DateTimeFormat(`${locale}-u-ca-gregory`, { timeZone: "UTC", ...c.options });
        row.push(c.times.map((t) => f.format(new Date(t))).join("|"));
        break;
      }
      case "firstDay": {
        const info = (new Intl.Locale(locale) as Intl.Locale & { getWeekInfo(): { firstDay: number } }).getWeekInfo();
        row.push(String(info.firstDay % 7));
        break;
      }
      case "hourCycle":
        row.push(String(hourCycles[new Intl.DateTimeFormat(locale, { hour: "numeric" }).resolvedOptions().hourCycle!]));
        break;
    }
  }
  out[locale] = row;
}
const names_: Record<string, string> = {};
for (const tag of autonyms) names_[tag] = new Intl.DisplayNames([tag], { type: "language" }).of(tag) ?? "";

const versions = process.versions as Record<string, string | undefined>;
const icu = `${versions.icu ?? "unknown"} (CLDR ${versions.cldr ?? "unknown"})`;
const lines = (m: Record<string, unknown>) =>
  "{\n" + Object.entries(m).map(([k, v]) => `  ${JSON.stringify(k)}: ${JSON.stringify(v)}`).join(",\n") + "\n }";
const file = join(dir, "format_icu.json");
await writeFile(
  file,
  `{\n "icu": ${JSON.stringify(icu)},\n "cases": [\n${cases.map((c) => "  " + JSON.stringify(c)).join(",\n")}\n ],\n "locales": ${lines(out)},\n "autonyms": ${lines(names_)}\n}\n`,
);
console.log(`wrote ${file}: ICU ${icu}, ${locales.length} locales, ${cases.length} cases`);
