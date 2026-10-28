import type { Metadata } from "next";
import { routes } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function blogMetadata(locale: Locale): Metadata {
  const { blog } = loadMessages(locale);
  return pageMetadata(routes.blog, locale, blog.title, blog.lead);
}
