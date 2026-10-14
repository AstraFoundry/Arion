import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function creditsMetadata(locale: Locale): Metadata {
  const { creditsPage } = loadMessages(locale);
  return pageMetadata(routes.credits, locale, creditsPage.title, creditsPage.lead);
}
