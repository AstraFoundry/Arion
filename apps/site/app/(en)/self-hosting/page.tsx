import { SelfHostingPage, selfHostingMetadata } from "@/pages/self-hosting";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = selfHostingMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <SelfHostingPage />;
}
