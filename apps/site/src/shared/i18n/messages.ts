import type { Locale } from "./config";
import en from "./messages/en.json";
import ja from "./messages/ja.json";
import zh from "./messages/zh.json";

/** The copy of one locale: `messages/<locale>.json`, shaped like English. */
export type Messages = typeof en;

const SITE_MESSAGES: Record<Locale, Messages> = { en, zh, ja };

export function loadMessages(locale: Locale): Messages {
  return SITE_MESSAGES[locale];
}
