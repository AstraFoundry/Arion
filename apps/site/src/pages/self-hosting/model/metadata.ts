import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function selfHostingMetadata(locale: Locale): Metadata {
  const { selfHostingPage } = loadMessages(locale);
  return pageMetadata(routes.selfHosting, locale, selfHostingPage.title, selfHostingPage.lead);
}
