import "server-only";
import { setRequestLocale } from "next-intl/server";
import { isLocale, type Locale } from "./config";

export { loadMessages } from "./messages";

/**
 * Sets the locale for the current render. Call it first in every layout and
 * page (static rendering has no request to read the locale from).
 */
export function setPageLocale(locale: Locale): void {
  setRequestLocale(locale);
}

/** Validates a `[locale]` route param; unknown values never reach a page. */
export function toLocale(value: string): Locale {
  if (!isLocale(value)) throw new Error(`Unknown locale "${value}"`);
  return value;
}
