import { postSlugs } from "@/entities/post";
import { BlogPostPage, blogPostMetadata } from "@/pages/blog-post";
import { setPageLocale, toLocale } from "@/shared/i18n/server";

interface RouteProps {
  params: Promise<{ locale: string; slug: string }>;
}

// Every post exists in every locale (content/blog/<slug>/<locale>.md).
export const dynamicParams = false;

export function generateStaticParams() {
  return postSlugs().map((slug) => ({ slug }));
}

export async function generateMetadata({ params }: RouteProps) {
  const { locale, slug } = await params;
  return blogPostMetadata(slug, toLocale(locale));
}

export default async function Page({ params }: RouteProps) {
  const { locale, slug } = await params;
  setPageLocale(toLocale(locale));
  return <BlogPostPage slug={slug} />;
}
