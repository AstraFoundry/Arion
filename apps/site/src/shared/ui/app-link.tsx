import Link from "next/link";
import type { ComponentProps } from "react";
import { localizePath, useLocale } from "@/shared/i18n";

type AppLinkProps = Omit<ComponentProps<typeof Link>, "href"> & { href: string };

/**
 * `next/link` for site routes: root-relative paths (`/docs/`, `/#faq`) get
 * the current locale's prefix, so links stay in the reader's language.
 * External URLs pass through unchanged.
 */
export function AppLink({ href, ...rest }: AppLinkProps) {
  const locale = useLocale();
  return <Link href={localizePath(href, locale)} {...rest} />;
}
