import type { ReactNode } from "react";
import { LocaleRootLayout, rootMetadata, rootViewport } from "@/app";

// Root layout for English, the default locale, served without a path prefix.
export const metadata = rootMetadata("en");
export const viewport = rootViewport;

export default function Layout({ children }: { children: ReactNode }) {
  return <LocaleRootLayout locale="en">{children}</LocaleRootLayout>;
}
