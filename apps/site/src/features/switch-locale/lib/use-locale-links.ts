"use client";

import { usePathname } from "next/navigation";
import {
  type Locale,
  LOCALE_NAMES,
  LOCALE_TAGS,
  LOCALES,
  localizePath,
  parseLocalePath,
  useLocale,
} from "@/shared/i18n";

export interface LocaleLink {
  locale: Locale;
  name: string;
  tag: string;
  href: string;
  current: boolean;
}

/** The current page in every locale; every page exists in all of them. */
export function useLocaleLinks(): LocaleLink[] {
  const current = useLocale();
  const pathname = usePathname();
  if (pathname === null)
    throw new Error("useLocaleLinks needs the App Router's pathname");
  const { path } = parseLocalePath(pathname);
  return LOCALES.map((locale) => ({
    locale,
    name: LOCALE_NAMES[locale],
    tag: LOCALE_TAGS[locale],
    href: localizePath(path, locale),
    current: locale === current,
  }));
}
