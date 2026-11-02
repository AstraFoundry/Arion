import { UsageDataPage, usageDataMetadata } from "@/pages/usage-data";
import { setPageLocale, toLocale } from "@/shared/i18n/server";

interface RouteProps {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({ params }: RouteProps) {
  return usageDataMetadata(toLocale((await params).locale));
}

export default async function Page({ params }: RouteProps) {
  setPageLocale(toLocale((await params).locale));
  return <UsageDataPage />;
}
