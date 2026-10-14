import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function privacyMetadata(locale: Locale): Metadata {
  const { privacyPage } = loadMessages(locale);
  return pageMetadata(routes.privacy, locale, privacyPage.title, privacyPage.lead);
}
