import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, localeAlternates } from "@/shared/i18n";

/** The home page uses the site title and description from the root layout. */
export function homeMetadata(locale: Locale): Metadata {
  return { alternates: localeAlternates(routes.home, locale) };
}
