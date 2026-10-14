import { CreditsPage, creditsMetadata } from "@/pages/credits";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = creditsMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <CreditsPage />;
}
