import { getRequestConfig } from "next-intl/server";
import { isLocale } from "./config";
import { loadMessages } from "./messages";

/**
 * next-intl request configuration. Every layout and page sets the locale with
 * `setRequestLocale` (see `server.ts`) before anything reads copy; a missing
 * or unknown locale is a routing bug, so it fails the build.
 */
export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  if (!isLocale(requested)) {
    throw new Error(
      `No locale was set for this render (got ${String(requested)}); call setPageLocale first`,
    );
  }
  return {
    locale: requested,
    messages: loadMessages(requested),
    // Copy is plain text with `{name}` placeholders filled by `format()`.
    onError: (error) => {
      throw error;
    },
  };
});
