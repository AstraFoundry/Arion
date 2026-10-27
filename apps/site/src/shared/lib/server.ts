import "server-only";
import { readFileSync } from "node:fs";
import path from "node:path";
import { imageSize } from "image-size";

export interface ImageSize {
  width: number;
  height: number;
}

/**
 * Intrinsic size of an image under `public/`, addressed by its site path
 * (`/blog/<slug>/cover.webp`). Pages set it on `<img>` so lazy images reserve
 * their space. A missing or unreadable file fails the build.
 */
export function publicImageSize(src: string): ImageSize {
  if (!src.startsWith("/") || src.startsWith("//")) {
    throw new Error(`"${src}" is not a site path; local images start with "/"`);
  }
  const file = path.join(process.cwd(), "public", decodeURIComponent(src.split(/[?#]/)[0]));
  const { width, height } = imageSize(readFileSync(file));
  if (!width || !height) throw new Error(`Cannot read the size of ${file}`);
  return { width, height };
}
