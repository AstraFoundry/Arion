import type { Metadata } from "next";
import type { Locale } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

export function notFoundMetadata(locale: Locale): Metadata {
  const { notFound } = loadMessages(locale);
  return { title: notFound.title, description: notFound.body, robots: { index: false } };
}
