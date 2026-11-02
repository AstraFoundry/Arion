import { UsageDataPage, usageDataMetadata } from "@/pages/usage-data";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = usageDataMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <UsageDataPage />;
}
