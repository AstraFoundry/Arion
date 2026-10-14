import type { MetadataRoute } from "next";
import { routes, siteConfig } from "@/shared/config";
import { LOCALE_TAGS, LOCALES, localizePath } from "@/shared/i18n";

export const dynamic = "force-static";

type Entry = Omit<MetadataRoute.Sitemap[number], "url" | "alternates">;

/** One entry per page and locale, each listing its translations. */
export default function sitemap(): MetadataRoute.Sitemap {
  const url = (path: string) => new URL(path, siteConfig.url).toString();
  const localized = (path: string, entry: Entry): MetadataRoute.Sitemap =>
    LOCALES.map((locale) => ({
      ...entry,
      url: url(localizePath(path, locale)),
      alternates: {
        languages: Object.fromEntries(
          LOCALES.map((each) => [LOCALE_TAGS[each], url(localizePath(path, each))]),
        ),
      },
    }));

  return [
    ...localized(routes.home, { changeFrequency: "weekly", priority: 1 }),
    ...localized(routes.privacy, { changeFrequency: "monthly", priority: 0.8 }),
    ...localized(routes.selfHosting, { changeFrequency: "monthly", priority: 0.7 }),
    ...localized(routes.credits, { changeFrequency: "yearly", priority: 0.3 }),
  ];
}
