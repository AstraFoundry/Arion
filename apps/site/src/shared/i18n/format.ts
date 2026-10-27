import { type Locale, LOCALE_TAGS } from "./config";

/** Replaces `{name}` placeholders in a dictionary template. */
export function format(
  template: string,
  values: Record<string, string | number>,
): string {
  return template.replace(/\{(\w+)\}/g, (match, key: string) =>
    key in values ? String(values[key]) : match,
  );
}

/**
 * A calendar date (`YYYY-MM-DD`) in the locale's medium form: "Oct 27, 2026",
 * "2026年10月27日". Read and written in UTC so the day never shifts.
 */
export function formatDate(isoDate: string, locale: Locale): string {
  return new Intl.DateTimeFormat(LOCALE_TAGS[locale], {
    year: "numeric",
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${isoDate}T00:00:00Z`));
}

/** Names joined the locale's way: "Ada, Lena and Kai", "Ada、Lena和Kai". */
export function formatList(items: readonly string[], locale: Locale): string {
  return new Intl.ListFormat(LOCALE_TAGS[locale], { type: "conjunction" }).format(items);
}
