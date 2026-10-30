import { getAuthors, lastModified, type Post } from "@/entities/post";
import { blogPostRoute, siteConfig } from "@/shared/config";
import { LOCALE_TAGS, localizePath } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";

const absolute = (url: string) => new URL(url, siteConfig.url).toString();

/** schema.org `BlogPosting` for a post, as search engines read it from JSON-LD. */
export async function blogPostingData(post: Post) {
  const { blog } = loadMessages(post.locale);
  const authors = await getAuthors(post.authors);
  const url = absolute(localizePath(blogPostRoute(post.slug), post.locale));
  return {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    headline: post.title,
    description: post.description,
    url,
    mainEntityOfPage: url,
    inLanguage: LOCALE_TAGS[post.locale],
    datePublished: post.date,
    dateModified: lastModified(post),
    articleSection: blog.categories[post.category],
    ...(post.tags.length > 0 ? { keywords: post.tags } : {}),
    ...(authors.length > 0
      ? {
          author: authors.map((author) => ({
            "@type": "Person",
            name: author.name,
            alternateName: author.login,
            url: author.profileUrl,
            image: absolute(author.avatar),
          })),
        }
      : {}),
    ...(post.image ? { image: absolute(post.image.src) } : {}),
    publisher: {
      "@type": "Organization",
      name: siteConfig.organization,
      url: siteConfig.organizationUrl,
    },
  };
}

/** JSON for an inline `<script>`: `<` is escaped so text can never close the tag. */
export const jsonLd = (data: object) => JSON.stringify(data).replace(/</g, "\\u003c");
