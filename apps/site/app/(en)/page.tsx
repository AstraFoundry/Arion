import { HomePage, homeMetadata } from "@/pages/home";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = homeMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <HomePage />;
}
