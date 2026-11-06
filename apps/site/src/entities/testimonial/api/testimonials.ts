import "server-only";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";
import { assetUrl, PROJECT_ASSET_TYPES } from "@/shared/lib/server";
import {
  paragraphsOf,
  parseTestimonials,
  type Testimonial,
  TestimonialError,
  type TestimonialSource,
  xProfileUrl,
} from "../model/testimonial";

/**
 * Quotes, and the X profiles `pnpm testimonials:update` committed beside
 * them. The site build reads this directory and does not call X.
 */
const CONTENT_DIR = path.join(process.cwd(), "content", "testimonials");
const UPDATE = "Run pnpm testimonials:update and commit content/testimonials.";

interface CachedProfile {
  handle: string;
  name: string;
  avatar: string;
  avatarSHA256: string;
}

interface Resolved {
  testimonial: Testimonial;
  avatar: Uint8Array;
}

function readJSON(name: string): unknown {
  const file = path.join(CONTENT_DIR, name);
  try {
    return JSON.parse(readFileSync(file, "utf8"));
  } catch (error) {
    throw new TestimonialError(`content/testimonials/${name}`, (error as Error).message);
  }
}

function profiles(): Map<string, CachedProfile> {
  const parsed = readJSON("profiles.json") as { profiles?: CachedProfile[] };
  if (!Array.isArray(parsed.profiles)) {
    throw new TestimonialError("content/testimonials/profiles.json", `has no profiles list. ${UPDATE}`);
  }
  return new Map(parsed.profiles.map((profile) => [profile.handle?.toLowerCase(), profile]));
}

function resolve(source: TestimonialSource, cache: Map<string, CachedProfile>): Resolved {
  const where = `testimonial @${source.handle}`;
  const profile = cache.get(source.handle.toLowerCase());
  if (!profile) throw new TestimonialError(where, `is not in the profile cache. ${UPDATE}`);
  if (!profile.name || !profile.avatar || !profile.avatarSHA256) {
    throw new TestimonialError(where, `has an incomplete cache entry. ${UPDATE}`);
  }
  const extension = path.extname(profile.avatar).toLowerCase();
  if (path.basename(profile.avatar) !== profile.avatar || !PROJECT_ASSET_TYPES[extension]?.startsWith("image/")) {
    throw new TestimonialError(where, `avatar "${profile.avatar}" is not an image file in the cache`);
  }
  const bytes = readFileSync(path.join(CONTENT_DIR, profile.avatar));
  if (createHash("sha256").update(bytes).digest("hex") !== profile.avatarSHA256) {
    throw new TestimonialError(where, `avatar ${profile.avatar} does not match profiles.json`);
  }
  const avatar = new Uint8Array(bytes);
  return {
    testimonial: {
      handle: profile.handle,
      name: profile.name,
      avatar: assetUrl(avatar, profile.avatar),
      lang: source.lang,
      paragraphs: paragraphsOf(source.text),
      href: source.postUrl ?? xProfileUrl(profile.handle),
    },
    avatar,
  };
}

/** One read per build process, shared by every locale and the asset route. */
let all: Resolved[] | undefined;

function resolveAll(): Resolved[] {
  if (!all) {
    const sources = parseTestimonials(readJSON("testimonials.json"), "content/testimonials/testimonials.json");
    const cache = profiles();
    all = sources.map((source) => resolve(source, cache));
  }
  return all;
}

/**
 * The quotes in `content/testimonials/testimonials.json` with each author's
 * cached X name and avatar. A handle missing from the cache, or an avatar
 * that does not match its recorded SHA-256, fails the build.
 */
export const getTestimonials = async (): Promise<Testimonial[]> =>
  resolveAll().map(({ testimonial }) => testimonial);

/** The avatar bytes behind each published avatar URL, for the static export. */
export const testimonialAvatars = async (): Promise<[string, Uint8Array][]> =>
  resolveAll().map(({ testimonial, avatar }) => [testimonial.avatar, avatar]);
