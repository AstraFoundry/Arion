import { useCopy } from "@/shared/i18n";
import { routes, siteConfig } from "@/shared/config";
import { AppLink, DocumentPage } from "@/shared/ui";

export function UsageDataPage() {
  const { usageDataPage, nav, footer } = useCopy();
  return (
    <DocumentPage
      title={usageDataPage.title}
      lead={usageDataPage.lead}
      sections={usageDataPage.sections}
    >
      <nav className="mt-12 flex flex-wrap gap-6 text-sm" aria-label={usageDataPage.title}>
        <AppLink href={routes.privacy}>{nav.privacy}</AppLink>
        <AppLink href={siteConfig.licenseUrl}>{footer.license}</AppLink>
      </nav>
    </DocumentPage>
  );
}
