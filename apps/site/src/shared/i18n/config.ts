export const LOCALES = ["en", "zh", "ja"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && (LOCALES as readonly string[]).includes(value);
}

/** Locales served under a path prefix (`/zh/`, `/ja/`); the default has none. */
export const PREFIXED_LOCALES = LOCALES.filter((locale) => locale !== DEFAULT_LOCALE);

/** BCP 47 tags for `<html lang>`, `hreflang` and `Intl` formatting. */
export const LOCALE_TAGS: Record<Locale, string> = {
  en: "en",
  zh: "zh-CN",
  ja: "ja",
};

/**
 * Each language's name in that language. They read the same on every page,
 * so they are configuration rather than per-locale copy.
 */
export const LOCALE_NAMES: Record<Locale, string> = {
  en: "English",
  zh: "简体中文",
  ja: "日本語",
};

/**
 * Adds the locale prefix to a root-relative path: `/privacy/` → `/zh/privacy/`.
 * External URLs, in-page anchors and default-locale paths are returned as is.
 */
export function localizePath(path: string, locale: Locale): string {
  if (locale === DEFAULT_LOCALE || !path.startsWith("/") || path.startsWith("//")) {
    return path;
  }
  return path === "/" ? `/${locale}/` : `/${locale}${path}`;
}

/** Splits a pathname into its locale and the locale-free path. */
export function parseLocalePath(pathname: string): { locale: Locale; path: string } {
  const [, first] = pathname.split("/");
  if (isLocale(first) && first !== DEFAULT_LOCALE) {
    const rest = pathname.slice(first.length + 1);
    return { locale: first, path: rest === "" ? "/" : rest };
  }
  return { locale: DEFAULT_LOCALE, path: pathname };
}

/**
 * `alternates` metadata for a page that exists in every locale: its canonical
 * URL in `locale`, and one `hreflang` entry per locale (English is x-default).
 */
export function localeAlternates(path: string, locale: Locale) {
  const languages: Record<string, string> = {};
  for (const each of LOCALES) languages[LOCALE_TAGS[each]] = localizePath(path, each);
  languages["x-default"] = path;
  return { canonical: localizePath(path, locale), languages };
}

/** Page metadata for a route that exists in every locale. */
export function pageMetadata(path: string, locale: Locale, title: string, description: string) {
  return {
    title,
    description,
    alternates: localeAlternates(path, locale),
    openGraph: { title, description, url: localizePath(path, locale) },
  };
}
