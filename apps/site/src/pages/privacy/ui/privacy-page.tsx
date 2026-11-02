import { useCopy } from "@/shared/i18n";
import { routes } from "@/shared/config";
import { AppLink, DocumentPage } from "@/shared/ui";

export function PrivacyPage() {
  const { privacyPage, usageDataPage } = useCopy();
  return (
    <DocumentPage
      title={privacyPage.title}
      lead={privacyPage.lead}
      sections={privacyPage.sections}
    >
      <nav className="mt-12 text-sm" aria-label={privacyPage.title}>
        <AppLink href={routes.usageData}>{usageDataPage.title}</AppLink>
      </nav>
    </DocumentPage>
  );
}
