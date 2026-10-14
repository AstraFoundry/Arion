import { PrivacyPage, privacyMetadata } from "@/pages/privacy";
import { setPageLocale, toLocale } from "@/shared/i18n/server";

interface RouteProps {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({ params }: RouteProps) {
  return privacyMetadata(toLocale((await params).locale));
}

export default async function Page({ params }: RouteProps) {
  setPageLocale(toLocale((await params).locale));
  return <PrivacyPage />;
}
