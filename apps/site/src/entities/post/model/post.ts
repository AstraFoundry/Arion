import type { Locale } from "@/shared/i18n";

/** Categories a post can file under; each has a label in `blog.categories`. */
export const POST_CATEGORIES = ["announcement", "product", "engineering"] as const;
export type PostCategory = (typeof POST_CATEGORIES)[number];

/** A post directory name: lowercase words joined by hyphens. */
export const POST_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export interface PostImage {
  /** The URL pages use: remote, a `public/` path, or a published project file. */
  src: string;
  alt: string;
  /** The local file behind `src`; absent for a remote image. */
  file?: string;
}

/**
 * A post's metadata. Field names follow common front matter conventions and
 * map one to one onto Open Graph `article` and schema.org `BlogPosting`.
 */
export interface PostMeta {
  slug: string;
  locale: Locale;
  title: string;
  description: string;
  /** Publication day, `YYYY-MM-DD`. */
  date: string;
  /** Day of the last substantive revision, `YYYY-MM-DD`; never before `date`. */
  updated?: string;
  authors: string[];
  tags: string[];
  category: PostCategory;
  /** The cover: shown above the post and used as its social preview. */
  image?: PostImage;
  /** Drafts are built in development only. */
  draft: boolean;
}

export interface Post extends PostMeta {
  /** The Markdown body after the front matter. */
  body: string;
  /** Absolute path of the Markdown file; relative media paths start here. */
  file: string;
}

/** The front matter as written, before the image is resolved. */
export interface PostFrontmatter {
  title: string;
  description: string;
  date: string;
  updated?: string;
  authors: string[];
  tags: string[];
  category: PostCategory;
  image?: { src: string; alt: string };
  draft: boolean;
}

/** Every front matter key a post may use. */
export const FRONTMATTER_KEYS = [
  "title",
  "description",
  "date",
  "updated",
  "authors",
  "tags",
  "category",
  "image",
  "imageAlt",
  "draft",
] as const;

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
function isoDate(value: unknown, key: string, source: string): string {
  const day = value instanceof Date ? value.toISOString().slice(0, 10) : value;
  if (typeof day !== "string" || !ISO_DATE.test(day) || Number.isNaN(Date.parse(day))) {
    throw new PostError(source, `front matter "${key}" must be a YYYY-MM-DD date`);
  }
  return day;
}

/** An optional list of distinct, non-empty strings (`tags: [a, b]`). */
function list(data: Record<string, unknown>, key: string, source: string): string[] {
  const value = data[key];
  if (value === undefined) return [];
  if (!Array.isArray(value) || value.some((item) => typeof item !== "string" || item.trim() === "")) {
    throw new PostError(source, `front matter "${key}" must be a list of non-empty strings`);
  }
  const items = value.map((item: string) => item.trim());
  if (new Set(items).size !== items.length) {
    throw new PostError(source, `front matter "${key}" lists an entry twice`);
  }
  return items;
}

/** Validates a post's front matter; every problem fails the build. */
export function parseFrontmatter(data: unknown, source: string): PostFrontmatter {
  if (!data || typeof data !== "object") throw new PostError(source, "missing front matter");
  const fields = data as Record<string, unknown>;
  const known = new Set<string>(FRONTMATTER_KEYS);
  for (const key of Object.keys(fields)) {
    if (!known.has(key)) {
      throw new PostError(
        source,
        `unknown front matter "${key}"; posts use ${FRONTMATTER_KEYS.join(", ")}`,
      );
    }
  }

  const category = fields.category;
  if (!POST_CATEGORIES.includes(category as PostCategory)) {
    throw new PostError(
      source,
      `front matter "category" must be one of ${POST_CATEGORIES.join(", ")}`,
    );
  }
  if (fields.draft !== undefined && typeof fields.draft !== "boolean") {
    throw new PostError(source, `front matter "draft" must be true or false`);
  }

  const parsed: PostFrontmatter = {
    title: text(fields, "title", source),
    description: text(fields, "description", source),
    date: isoDate(fields.date, "date", source),
    authors: list(fields, "authors", source),
    tags: list(fields, "tags", source),
    category: category as PostCategory,
    draft: fields.draft === true,
  };
  if (fields.updated !== undefined) {
    parsed.updated = isoDate(fields.updated, "updated", source);
    if (parsed.updated < parsed.date) {
      throw new PostError(source, `front matter "updated" is before "date"`);
    }
  }
  if (fields.image !== undefined || fields.imageAlt !== undefined) {
    parsed.image = { src: text(fields, "image", source), alt: text(fields, "imageAlt", source) };
  }
  return parsed;
}

/**
 * Translations of one post share its publication date, category, cover image
 * and draft state; the words, authors' spelling, tags and revision date may
 * differ. Returns the first field that disagrees, if any.
 */
export function translationMismatch(posts: readonly PostMeta[]): string | undefined {
  const [first, ...rest] = posts;
  for (const post of rest) {
    if (post.date !== first.date) return "date";
    if (post.category !== first.category) return "category";
    if (post.image?.src !== first.image?.src) return "image";
    if (post.draft !== first.draft) return "draft";
  }
  return undefined;
}

/** The day a post last changed: its revision date, or its publication date. */
export const lastModified = (post: PostMeta) => post.updated ?? post.date;

/** Newest first; posts from the same day keep slug order. */
export function sortNewestFirst<T extends PostMeta>(posts: readonly T[]): T[] {
  return [...posts].sort((a, b) => b.date.localeCompare(a.date) || a.slug.localeCompare(b.slug));
}
