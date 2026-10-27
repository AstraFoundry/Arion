import type { Metadata } from "next";
import { getPost } from "@/entities/post";
import { blogPostRoute, siteConfig } from "@/shared/config";
import { type Locale, pageMetadata } from "@/shared/i18n";

export function blogPostMetadata(slug: string, locale: Locale): Metadata {
  const post = getPost(slug, locale);
  const base = pageMetadata(blogPostRoute(slug), locale, post.title, post.summary);
  return {
    ...base,
    openGraph: {
      ...base.openGraph,
      type: "article",
      publishedTime: post.date,
      siteName: siteConfig.name,
      ...(post.cover
        ? {
            images: [
              {
                url: post.cover.src,
                width: post.cover.width,
                height: post.cover.height,
                alt: post.cover.alt,
              },
            ],
          }
        : {}),
    },
  };
}
