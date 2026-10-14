import type { Metadata, Viewport } from "next";
import { routes, siteConfig } from "@/shared/config";
import { type Locale, LOCALE_TAGS, localeAlternates, LOCALES } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

/** Open Graph locale codes use an underscore (`zh_CN`). */
const ogLocale = (locale: Locale) => LOCALE_TAGS[locale].replace("-", "_");

export function rootMetadata(locale: Locale): Metadata {
  const { meta } = loadMessages(locale);
  return {
    metadataBase: new URL(siteConfig.url),
    title: { default: meta.title, template: meta.titleTemplate },
    description: meta.description,
    applicationName: siteConfig.name,
    openGraph: {
      type: "website",
      siteName: siteConfig.name,
      locale: ogLocale(locale),
      alternateLocale: LOCALES.filter((each) => each !== locale).map(ogLocale),
      title: meta.title,
      description: meta.description,
      images: [{ url: "/og.jpg", width: 1200, height: 630, alt: meta.ogAlt }],
    },
    twitter: {
      card: "summary_large_image",
      title: meta.title,
      description: meta.description,
      images: ["/og.jpg"],
    },
    icons: {
      icon: [{ url: "/brand/favicon-64.png", type: "image/png", sizes: "64x64" }],
      apple: [{ url: "/brand/apple-touch-icon.png", sizes: "180x180" }],
    },
    alternates: localeAlternates(routes.home, locale),
  };
}

export const rootViewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#F9F7F4" },
    { media: "(prefers-color-scheme: dark)", color: "#090807" },
  ],
};
