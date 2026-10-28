import type { Metadata } from "next";
import { getPost } from "@/entities/post";
import { blogPostRoute, siteConfig, socialImage } from "@/shared/config";
import { type Locale, LOCALE_TAGS, LOCALES, pageMetadata } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

/** Open Graph locale codes use an underscore (`zh_CN`). */
const ogLocale = (locale: Locale) => LOCALE_TAGS[locale].replace("-", "_");

/**
 * A post's head: description, canonical and hreflang links, Open Graph
 * `article` (published and modified times, authors, section, tags) and a
 * large Twitter card, with the post's image or the site preview.
 */
export function blogPostMetadata(slug: string, locale: Locale): Metadata {
  const post = getPost(slug, locale);
  const { blog, meta } = loadMessages(locale);
  const base = pageMetadata(blogPostRoute(slug), locale, post.title, post.description);
  const image = post.image
    ? { url: post.image.src, alt: post.image.alt }
    : { ...socialImage, alt: meta.ogAlt };

  return {
    ...base,
    ...(post.authors.length > 0 ? { authors: post.authors.map((name) => ({ name })) } : {}),
    ...(post.tags.length > 0 ? { keywords: post.tags } : {}),
    openGraph: {
      ...base.openGraph,
      type: "article",
      siteName: siteConfig.name,
      locale: ogLocale(locale),
      alternateLocale: LOCALES.filter((each) => each !== locale).map(ogLocale),
      publishedTime: post.date,
      modifiedTime: post.updated,
      authors: post.authors,
      section: blog.categories[post.category],
      tags: post.tags,
      images: [image],
    },
    twitter: {
      card: "summary_large_image",
      title: post.title,
      description: post.description,
      images: [image],
    },
  };
}
