import { useCopy } from "@/shared/i18n";
import { routes, siteConfig } from "@/shared/config";
import { AppLink, DocumentPage } from "@/shared/ui";

export function TermsPage() {
  const { termsPage, nav, footer } = useCopy();
  return (
    <DocumentPage
      title={termsPage.title}
      lead={termsPage.lead}
      sections={termsPage.sections}
    >
      <nav className="mt-12 flex flex-wrap gap-6 text-sm" aria-label={termsPage.title}>
        <AppLink href={routes.privacy}>{nav.privacy}</AppLink>
        <AppLink href={siteConfig.licenseUrl}>{footer.license}</AppLink>
      </nav>
    </DocumentPage>
  );
}
