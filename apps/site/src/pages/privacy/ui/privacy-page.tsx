import { useCopy } from "@/shared/i18n";
import { DocumentPage } from "@/shared/ui";

export function PrivacyPage() {
  const { privacyPage } = useCopy();
  return (
    <DocumentPage
      title={privacyPage.title}
      lead={privacyPage.lead}
      sections={privacyPage.sections}
    />
  );
}
