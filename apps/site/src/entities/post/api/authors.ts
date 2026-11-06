import "server-only";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";
import { assetUrl, PROJECT_ASSET_TYPES } from "@/shared/lib/server";
import type { PostAuthor } from "../model/post";

/**
 * Profiles committed by `pnpm blog-authors:update`. The site build reads
 * this directory and does not call GitHub.
 */
const DEFAULT_CACHE_DIR = path.join(process.cwd(), "content", "authors");

interface CachedAuthor {
  login: string;
  name: string;
  profileUrl: string;
  avatar: string;
  avatarSHA256: string;
}

interface ResolvedAuthor {
  author: PostAuthor;
  avatar: Uint8Array;
}

let cacheDir = DEFAULT_CACHE_DIR;
let cached: Map<string, CachedAuthor> | undefined;

/** Vitest points the reader at a fixture. The site build uses `content/authors`. */
export function setAuthorCacheDirectory(dir: string): void {
  cacheDir = dir;
  cached = undefined;
  resolved.clear();
}

function missingFile(error: unknown): boolean {
  return typeof error === "object" && error !== null && "code" in error && error.code === "ENOENT";
}

function authorCache(): Map<string, CachedAuthor> {
  if (cached) return cached;
  const file = path.join(cacheDir, "authors.json");
  const where = path.relative(process.cwd(), file);
  let parsed: { authors?: CachedAuthor[] };
  try {
    parsed = JSON.parse(readFileSync(file, "utf8")) as { authors?: CachedAuthor[] };
  } catch (error) {
    throw new Error(
      missingFile(error)
        ? `Blog author cache ${where} is missing. Run pnpm blog-authors:update and commit it.`
        : `Blog author cache ${where}: ${(error as Error).message}`,
    );
  }
  if (!Array.isArray(parsed.authors)) throw new Error(`Blog author cache ${where} has no authors list`);
  const map = new Map<string, CachedAuthor>();
  for (const author of parsed.authors) {
    if (!author?.login) throw new Error(`Blog author cache ${where} has an entry with no login`);
    const key = author.login.toLowerCase();
    if (map.has(key)) throw new Error(`Blog author cache lists "${author.login}" twice`);
    map.set(key, author);
  }
  cached = map;
  return map;
}

function resolveAuthor(login: string): ResolvedAuthor {
  const author = authorCache().get(login.toLowerCase());
  if (!author) {
    throw new Error(
      `Blog author "${login}" is not in the committed cache. Run pnpm blog-authors:update and commit content/authors.`,
    );
  }
  if (!author.name || !author.profileUrl || !author.avatar || !author.avatarSHA256) {
    throw new Error(`Blog author "${login}" has an incomplete cache entry`);
  }
  if (path.basename(author.avatar) !== author.avatar) {
    throw new Error(`Blog author "${login}": avatar "${author.avatar}" must be a file name in the cache`);
  }
  const extension = path.extname(author.avatar).toLowerCase();
  const type = PROJECT_ASSET_TYPES[extension];
  if (!type?.startsWith("image/")) {
    throw new Error(`Blog author "${login}": avatar "${author.avatar}" is not a supported image`);
  }
  const file = path.join(cacheDir, author.avatar);
  let bytes: Buffer;
  try {
    bytes = readFileSync(file);
  } catch (error) {
    throw new Error(`Blog author "${login}": ${(error as Error).message}`);
  }
  const sum = createHash("sha256").update(bytes).digest("hex");
  if (sum !== author.avatarSHA256) {
    throw new Error(`Blog author "${login}": avatar ${author.avatar} does not match authors.json`);
  }
  const avatar = new Uint8Array(bytes);
  return {
    author: {
      login: author.login,
      name: author.name,
      profileUrl: author.profileUrl,
      avatar: assetUrl(avatar, `${author.login.toLowerCase()}${extension}`),
    },
    avatar,
  };
}

/** One read per login per build process; logins are case-insensitive. */
const resolved = new Map<string, Promise<ResolvedAuthor>>();

function resolve(login: string): Promise<ResolvedAuthor> {
  const key = login.toLowerCase();
  let author = resolved.get(key);
  if (!author) {
    author = Promise.resolve().then(() => resolveAuthor(login));
    resolved.set(key, author);
  }
  return author;
}

/**
 * Profiles of a post's authors from `content/authors`. A login that is not
 * in that cache, or an avatar that does not match its recorded SHA-256,
 * fails the build.
 */
export const getAuthors = (logins: readonly string[]): Promise<PostAuthor[]> =>
  Promise.all(logins.map(async (login) => (await resolve(login)).author));

/** The avatar bytes behind each author's published URL, for the static export. */
export const authorAvatars = async (logins: readonly string[]): Promise<[string, Uint8Array][]> =>
  (await Promise.all(logins.map(resolve))).map(({ author, avatar }) => [author.avatar, avatar]);
