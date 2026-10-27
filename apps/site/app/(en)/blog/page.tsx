import { BlogPage, blogMetadata } from "@/pages/blog";
import { setPageLocale } from "@/shared/i18n/server";

export const metadata = blogMetadata("en");

export default function Page() {
  setPageLocale("en");
  return <BlogPage />;
}
