import { TermsPage, termsMetadata } from "@/pages/terms";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = termsMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <TermsPage />;
}
