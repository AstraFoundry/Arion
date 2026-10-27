import "server-only";
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import matter from "gray-matter";
import { type Locale, LOCALES } from "@/shared/i18n";
import { resolveMedia } from "@/shared/lib/server";
import {
  parseFrontmatter,
  type Post,
  type PostCover,
  POST_SLUG,
  PostError,
  type PostMeta,
  sortNewestFirst,
  translationMismatch,
} from "../model/post";

/**
 * `content/blog/<slug>/<locale>.md`. Media may sit next to the Markdown, in
 * `public/`, anywhere else in the repository, or on another site.
 */
const CONTENT_DIR = path.join(process.cwd(), "content", "blog");

function coverOf(cover: { src: string; alt: string }, file: string, source: string): PostCover {
  try {
    const { url, file: local } = resolveMedia(cover.src, file);
    return { src: url, alt: cover.alt, file: local };
  } catch (error) {
    throw new PostError(source, `cover: ${(error as Error).message}`);
  }
}

function readPost(slug: string, locale: Locale): Post {
  const file = path.join(CONTENT_DIR, slug, `${locale}.md`);
  const source = path.relative(process.cwd(), file);
  const { data, content } = matter(readFileSync(file, "utf8"));
  const { cover, ...fields } = parseFrontmatter(data, source);
  return {
    ...fields,
    slug,
    locale,
    cover: cover ? coverOf(cover, file, source) : undefined,
    body: content,
    file,
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
    const markdown = readdirSync(path.join(CONTENT_DIR, slug)).filter((name) => name.endsWith(".md"));
    const missing = expected.filter((name) => !markdown.includes(name));
    const extra = markdown.filter((name) => !expected.includes(name));
    if (missing.length) throw new PostError(where, `missing ${missing.join(", ")}`);
    if (extra.length) {
      throw new PostError(where, `unexpected ${extra.join(", ")}; posts are ${expected.join(", ")}`);
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
