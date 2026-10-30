import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function termsMetadata(locale: Locale): Metadata {
  const { termsPage } = loadMessages(locale);
  return pageMetadata(routes.terms, locale, termsPage.title, termsPage.lead);
}
