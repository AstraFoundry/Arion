import "server-only";
import { createHash } from "node:crypto";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { imageSize } from "image-size";
import { isRemoteUrl } from "./remote-url";

export interface ImageSize {
  width: number;
  height: number;
}

/** A media reference after resolution: the URL pages use, and its file when local. */
export interface MediaRef {
  url: string;
  /** Absolute path of a file in `public/` or in the repository; absent for remote URLs. */
  file?: string;
}

/** The site package; builds and tests run from here. */
const SITE_ROOT = process.cwd();
/** Project paths may point anywhere in the repository, never outside it. */
export const REPOSITORY_ROOT = path.resolve(SITE_ROOT, "..", "..");

/** Where repository files referenced by a project path are published. */
export const PROJECT_ASSET_BASE = "/blog-assets/";

/** Media types a project path may name, by extension. */
export const PROJECT_ASSET_TYPES: Record<string, string> = {
  ".avif": "image/avif",
  ".gif": "image/gif",
  ".jpeg": "image/jpeg",
  ".jpg": "image/jpeg",
  ".mp4": "video/mp4",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".webm": "video/webm",
  ".webp": "image/webp",
};

function requireFile(file: string, src: string): void {
  if (!existsSync(file) || !statSync(file).isFile()) throw new Error(`"${src}" names no file (looked for ${file})`);
}

/**
 * Bytes published at `/blog-assets/<content hash>/<name>`; the hash changes
 * with the bytes, so the URL can be cached forever.
 */
export function assetUrl(bytes: Uint8Array, name: string): string {
  const hash = createHash("sha256").update(bytes).digest("hex").slice(0, 12);
  return `${PROJECT_ASSET_BASE}${hash}/${name.replace(/[^A-Za-z0-9._-]+/g, "-")}`;
}

/** A repository file published under `/blog-assets/`. */
export const projectAssetUrl = (file: string) => assetUrl(readFileSync(file), path.basename(file));

/**
 * Resolves a media `src` written in a post at `fromFile`:
 * - `https://…`: a remote file, used as is;
 * - `/…`: a file in the site's `public/`;
 * - anything else: a path relative to the post, to any file in the
 *   repository, published under `/blog-assets/`.
 * A missing file, a path outside the repository or an unsupported type fails the build.
 */
export function resolveMedia(src: string, fromFile: string): MediaRef {
  if (isRemoteUrl(src)) return { url: src };
  if (src.startsWith("//") || /^[a-z][a-z\d+.-]*:/i.test(src)) {
    throw new Error(`"${src}": media URLs are https://…, /public-path or a relative project path`);
  }
  const bare = decodeURIComponent(src.split(/[?#]/)[0]);
  if (src.startsWith("/")) {
    const file = path.join(SITE_ROOT, "public", bare);
    requireFile(file, src);
    return { url: src, file };
  }

  const file = path.resolve(path.dirname(fromFile), bare);
  const inside = path.relative(REPOSITORY_ROOT, file);
  if (inside.startsWith("..") || path.isAbsolute(inside)) {
    throw new Error(`"${src}" points outside the repository`);
  }
  if (!PROJECT_ASSET_TYPES[path.extname(file).toLowerCase()]) {
    throw new Error(
      `"${src}": project paths may name ${Object.keys(PROJECT_ASSET_TYPES).join(", ")} files`,
    );
  }
  requireFile(file, src);
  return { url: projectAssetUrl(file), file };
}

function sizeOf(bytes: Uint8Array, where: string): ImageSize {
  const { width, height } = imageSize(bytes);
  if (!width || !height) throw new Error(`Cannot read the size of ${where}`);
  return { width, height };
}

/** How long a build waits for one remote image. */
const REMOTE_TIMEOUT_MS = 20_000;
const remoteSizes = new Map<string, Promise<ImageSize>>();

async function fetchSize(url: string): Promise<ImageSize> {
  const response = await fetch(url, { signal: AbortSignal.timeout(REMOTE_TIMEOUT_MS) });
  if (!response.ok) throw new Error(`Fetching ${url} for its size failed: HTTP ${response.status}`);
  return sizeOf(new Uint8Array(await response.arrayBuffer()), url);
}

/**
 * Intrinsic size of a resolved image, so lazy images reserve their space.
 * Local files are read; a remote image is downloaded once per build, and an
 * unreachable one fails the build rather than shipping a page that jumps.
 */
export function mediaSize(ref: MediaRef): Promise<ImageSize> {
  if (ref.file) return Promise.resolve(sizeOf(readFileSync(ref.file), ref.file));
  let size = remoteSizes.get(ref.url);
  if (!size) {
    size = fetchSize(ref.url);
    remoteSizes.set(ref.url, size);
  }
  return size;
}
