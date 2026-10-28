import { BlogPage, blogMetadata } from "@/pages/blog";
import { setPageLocale, toLocale } from "@/shared/i18n/server";

interface RouteProps {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({ params }: RouteProps) {
  return blogMetadata(toLocale((await params).locale));
}

export default async function Page({ params }: RouteProps) {
  setPageLocale(toLocale((await params).locale));
  return <BlogPage />;
}
