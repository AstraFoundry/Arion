import type { Locale } from "@/shared/i18n";

/** Categories a post can file under; each has a label in `blog.categories`. */
export const POST_CATEGORIES = ["announcement", "product", "engineering"] as const;
export type PostCategory = (typeof POST_CATEGORIES)[number];

/** A post directory name: lowercase words joined by hyphens. */
export const POST_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export interface PostCover {
  /** The URL pages use: remote, a `public/` path, or a published project file. */
  src: string;
  alt: string;
  /** The local file behind `src`; absent for a remote image. */
  file?: string;
}

export interface PostMeta {
  slug: string;
  locale: Locale;
  title: string;
  summary: string;
  /** Publication day, `YYYY-MM-DD`. */
  date: string;
  category: PostCategory;
  cover?: PostCover;
}

export interface Post extends PostMeta {
  /** The Markdown body after the front matter. */
  body: string;
  /** Absolute path of the Markdown file; relative media paths start here. */
  file: string;
}

/** The front matter fields as written, before the cover is resolved. */
export interface PostFrontmatter {
  title: string;
  summary: string;
  date: string;
  category: PostCategory;
  cover?: { src: string; alt: string };
}

export class PostError extends Error {
  constructor(source: string, message: string) {
    super(`${source}: ${message}`);
    this.name = "PostError";
  }
}

const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;

function text(data: Record<string, unknown>, key: string, source: string): string {
  const value = data[key];
  if (typeof value !== "string" || value.trim() === "") {
    throw new PostError(source, `front matter "${key}" must be a non-empty string`);
  }
  return value.trim();
}

/** YAML reads an unquoted `2026-10-27` as a Date; both forms name a UTC day. */
function isoDate(value: unknown, source: string): string {
  const day = value instanceof Date ? value.toISOString().slice(0, 10) : value;
  if (typeof day !== "string" || !ISO_DATE.test(day) || Number.isNaN(Date.parse(day))) {
    throw new PostError(source, `front matter "date" must be a YYYY-MM-DD date`);
  }
  return day;
}

/** Validates a post's front matter; every problem fails the build. */
export function parseFrontmatter(data: unknown, source: string): PostFrontmatter {
  if (!data || typeof data !== "object") throw new PostError(source, "missing front matter");
  const fields = data as Record<string, unknown>;
  const known = new Set(["title", "summary", "date", "category", "cover", "coverAlt"]);
  for (const key of Object.keys(fields)) {
    if (!known.has(key)) throw new PostError(source, `unknown front matter "${key}"`);
  }

  const category = fields.category;
  if (!POST_CATEGORIES.includes(category as PostCategory)) {
    throw new PostError(
      source,
      `front matter "category" must be one of ${POST_CATEGORIES.join(", ")}`,
    );
  }

  const parsed: PostFrontmatter = {
    title: text(fields, "title", source),
    summary: text(fields, "summary", source),
    date: isoDate(fields.date, source),
    category: category as PostCategory,
  };
  if (fields.cover !== undefined || fields.coverAlt !== undefined) {
    parsed.cover = { src: text(fields, "cover", source), alt: text(fields, "coverAlt", source) };
  }
  return parsed;
}

/**
 * Translations of one post share its date, category and cover image; only
 * the words differ. Returns the first field that disagrees, if any.
 */
export function translationMismatch(posts: readonly PostMeta[]): string | undefined {
  const [first, ...rest] = posts;
  for (const post of rest) {
    if (post.date !== first.date) return "date";
    if (post.category !== first.category) return "category";
    if (post.cover?.src !== first.cover?.src) return "cover";
  }
  return undefined;
}

/** Newest first; posts from the same day keep slug order. */
export function sortNewestFirst<T extends PostMeta>(posts: readonly T[]): T[] {
  return [...posts].sort((a, b) => b.date.localeCompare(a.date) || a.slug.localeCompare(b.slug));
}
