import { routes, sections, anchor, siteConfig } from "@/shared/config";
import type { Messages } from "@/shared/i18n";

export interface FooterColumn {
  title: string;
  links: { label: string; href: string }[];
}

export function footerColumns(copy: Messages): FooterColumn[] {
  return [
    {
      title: copy.footer.product,
      links: [
        { label: copy.common.download, href: siteConfig.releasesUrl },
        { label: copy.nav.privacy, href: routes.privacy },
        { label: copy.nav.selfHosting, href: routes.selfHosting },
        { label: copy.nav.openSource, href: anchor(sections.openSource) },
      ],
    },
    {
      title: copy.footer.project,
      links: [
        { label: copy.footer.source, href: siteConfig.repositoryUrl },
        { label: copy.footer.issues, href: siteConfig.issuesUrl },
        { label: copy.footer.license, href: siteConfig.licenseUrl },
        { label: copy.footer.credits, href: routes.credits },
      ],
    },
    {
      title: copy.footer.astraFoundry,
      links: [
        { label: copy.common.account, href: siteConfig.accountUrl },
        { label: copy.common.hostedSyncWaitlist, href: siteConfig.hostedSyncWaitlistUrl },
      ],
    },
  ];
}
