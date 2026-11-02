import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function usageDataMetadata(locale: Locale): Metadata {
  const { usageDataPage } = loadMessages(locale);
  return pageMetadata(routes.usageData, locale, usageDataPage.title, usageDataPage.lead);
}
