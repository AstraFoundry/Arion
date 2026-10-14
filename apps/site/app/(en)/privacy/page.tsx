import { PrivacyPage, privacyMetadata } from "@/pages/privacy";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = privacyMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <PrivacyPage />;
}
