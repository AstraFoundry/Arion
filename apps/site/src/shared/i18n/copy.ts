import { useLocale as useNextIntlLocale, useMessages } from "next-intl";
import type { Locale } from "./config";
import type { Messages } from "./messages";

/**
 * The current locale's copy. Works in client components and in non-async
 * server components; async server code uses `loadMessages(locale)`.
 */
export function useCopy(): Messages {
  return useMessages();
}

export function useLocale(): Locale {
  return useNextIntlLocale();
}
