import "server-only";
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import matter from "gray-matter";
import { type Locale, LOCALES } from "@/shared/i18n";
import { publicImageSize } from "@/shared/lib/server";
import {
  parseFrontmatter,
  type Post,
  POST_SLUG,
  PostError,
  type PostMeta,
  sortNewestFirst,
  translationMismatch,
} from "../model/post";

/** `content/blog/<slug>/<locale>.md`; images live in `public/blog/<slug>/`. */
const CONTENT_DIR = path.join(process.cwd(), "content", "blog");

function readPost(slug: string, locale: Locale): Post {
  const file = path.join(CONTENT_DIR, slug, `${locale}.md`);
  const source = path.relative(process.cwd(), file);
  const { data, content } = matter(readFileSync(file, "utf8"));
  const { cover, ...fields } = parseFrontmatter(data, source);
  return {
    ...fields,
    slug,
    locale,
    cover: cover ? { ...cover, ...publicImageSize(cover.src) } : undefined,
    body: content,
  };
}

/**
 * Every post exists in every site locale, like every other page, so the
 * language switch and hreflang links never point at a missing translation.
 */
function readAll(): Map<string, Record<Locale, Post>> {
  const posts = new Map<string, Record<Locale, Post>>();
  for (const entry of readdirSync(CONTENT_DIR, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const slug = entry.name;
    const where = path.relative(process.cwd(), path.join(CONTENT_DIR, slug));
    if (!POST_SLUG.test(slug)) throw new PostError(where, "slugs are lowercase-hyphenated");

    const expected = LOCALES.map((locale) => `${locale}.md`);
    const files = readdirSync(path.join(CONTENT_DIR, slug));
    const missing = expected.filter((name) => !files.includes(name));
    const extra = files.filter((name) => !expected.includes(name));
    if (missing.length) throw new PostError(where, `missing ${missing.join(", ")}`);
    if (extra.length) {
      throw new PostError(where, `unexpected ${extra.join(", ")} (images go in public/blog/${slug}/)`);
    }

    const translations = Object.fromEntries(
      LOCALES.map((locale) => [locale, readPost(slug, locale)]),
    ) as Record<Locale, Post>;
    const mismatch = translationMismatch(Object.values(translations));
    if (mismatch) throw new PostError(where, `translations disagree on "${mismatch}"`);
    posts.set(slug, translations);
  }
  return posts;
}

let cache: Map<string, Record<Locale, Post>> | undefined;
const allPosts = () => (cache ??= readAll());

export function postSlugs(): string[] {
  return [...allPosts().keys()];
}

/** One post in one locale; an unknown slug is a routing bug. */
export function getPost(slug: string, locale: Locale): Post {
  const post = allPosts().get(slug);
  if (!post) throw new Error(`No blog post "${slug}"`);
  return post[locale];
}

/** Every post in `locale`, newest first, without bodies. */
export function listPosts(locale: Locale): PostMeta[] {
  return sortNewestFirst(
    [...allPosts().values()].map(({ [locale]: post }) => ({
      slug: post.slug,
      locale: post.locale,
      title: post.title,
      summary: post.summary,
      date: post.date,
      category: post.category,
      cover: post.cover,
    })),
  );
}
