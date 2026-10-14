import { NextIntlClientProvider } from "next-intl";
import type { ReactNode } from "react";
import { type Locale, LOCALE_TAGS, useCopy } from "@/shared/i18n";
import { setPageLocale } from "@/shared/i18n/server";
import { SiteFooter } from "@/widgets/site-footer";
import { SiteHeader } from "@/widgets/site-header";
import { fontVariables } from "../fonts";
import { RootProviders } from "../providers/root-providers";

function Shell({ children }: { children: ReactNode }) {
  const copy = useCopy();
  return (
    <RootProviders>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[60] focus:rounded-full focus:bg-card focus:px-4 focus:py-2 focus:text-sm focus:shadow-3"
      >
        {copy.common.skipToContent}
      </a>
      <SiteHeader />
      <main id="main">{children}</main>
      <SiteFooter />
    </RootProviders>
  );
}

/**
 * The document shell for one locale. English (`app/(en)`) and the prefixed
 * locales (`app/[locale]`) each have a root layout that renders this, so
 * `<html lang>` is right in the static HTML.
 */
export function LocaleRootLayout({ locale, children }: { locale: Locale; children: ReactNode }) {
  setPageLocale(locale);
  return (
    <html lang={LOCALE_TAGS[locale]} className={fontVariables} suppressHydrationWarning>
      <body>
        <NextIntlClientProvider>
          <Shell>{children}</Shell>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
