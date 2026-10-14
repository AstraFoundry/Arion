import type { ReactNode } from "react";
import { LocaleRootLayout, rootMetadata, rootViewport } from "@/app";
import { PREFIXED_LOCALES } from "@/shared/i18n";
import { toLocale } from "@/shared/i18n/server";

interface LayoutProps {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}

// Root layout for the prefixed locales: /zh/…, /ja/….
export const dynamicParams = false;
export const viewport = rootViewport;

export function generateStaticParams() {
  return PREFIXED_LOCALES.map((locale) => ({ locale }));
}

export async function generateMetadata({ params }: LayoutProps) {
  return rootMetadata(toLocale((await params).locale));
}

export default async function Layout({ children, params }: LayoutProps) {
  const locale = toLocale((await params).locale);
  return <LocaleRootLayout locale={locale}>{children}</LocaleRootLayout>;
}
