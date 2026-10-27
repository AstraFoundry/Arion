import { postSlugs } from "@/entities/post";
import { BlogPostPage, blogPostMetadata } from "@/pages/blog-post";
import { setPageLocale } from "@/shared/i18n/server";

interface RouteProps {
  params: Promise<{ slug: string }>;
}

// One page per directory in content/blog; nothing else exists.
export const dynamicParams = false;

export function generateStaticParams() {
  return postSlugs().map((slug) => ({ slug }));
}

export async function generateMetadata({ params }: RouteProps) {
  return blogPostMetadata((await params).slug, "en");
}

export default async function Page({ params }: RouteProps) {
  setPageLocale("en");
  return <BlogPostPage slug={(await params).slug} />;
}
