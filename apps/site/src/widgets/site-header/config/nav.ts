import { anchor, routes, sections, siteConfig } from "@/shared/config";
import type { Messages } from "@/shared/i18n";

export function navItems(copy: Messages) {
  return [
    { href: routes.privacy, label: copy.nav.privacy },
    { href: routes.selfHosting, label: copy.nav.selfHosting },
    { href: anchor(sections.openSource), label: copy.nav.openSource },
    { href: siteConfig.hostedSyncWaitlistUrl, label: copy.common.hostedSyncWaitlist },
  ];
}
