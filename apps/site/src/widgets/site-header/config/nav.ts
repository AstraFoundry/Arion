import { anchor, routes, sections } from "@/shared/config";
import type { Messages } from "@/shared/i18n";

export function navItems(copy: Messages) {
  return [
    { href: routes.blog, label: copy.nav.blog },
    { href: routes.privacy, label: copy.nav.privacy },
    { href: routes.selfHosting, label: copy.nav.selfHosting },
    { href: anchor(sections.openSource), label: copy.nav.openSource },
    { href: anchor(sections.community), label: copy.nav.community },
  ];
}
