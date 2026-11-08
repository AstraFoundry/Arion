import { LOCALE_TAGS, LOCALES, type Locale } from "@/shared/i18n";

/** An X username: 1–15 letters, digits or underscores. */
export const X_HANDLE = /^[A-Za-z0-9_]{1,15}$/;
/** A link to one post on X, when a testimonial names it. */
const X_POST_URL = /^https:\/\/x\.com\/[A-Za-z0-9_]{1,15}\/status\/\d+$/;

/** One entry of `content/testimonials.json`: a post quoted as written. */
export interface TestimonialSource {
  handle: string;
  /** BCP 47 tag of the quoted text; the page marks the quote with it. */
  lang: string;
  /** The post text; a blank line separates paragraphs. */
  text: string;
  /** The post itself; without it the card links to the author's profile. */
  postUrl?: string;
  /**
   * The text in each site locale whose language differs from the post's.
   * A locale without an entry shows the post as written.
   */
  translations: Partial<Record<Locale, string>>;
}

/** A testimonial with the author's X profile, resolved at build time. */
export interface Testimonial {
  handle: string;
  name: string;
  /** Published URL of the author's avatar. */
  avatar: string;
  lang: string;
  paragraphs: string[];
  /** The post when known, otherwise the profile. */
  href: string;
}

export class TestimonialError extends Error {
  constructor(where: string, message: string) {
    super(`${where}: ${message}`);
    this.name = "TestimonialError";
  }
}

/** Validates the parsed JSON; anything malformed fails the build. */
export function parseTestimonials(value: unknown, where: string): TestimonialSource[] {
  if (!Array.isArray(value) || value.length === 0) {
    throw new TestimonialError(where, "expected a non-empty array");
  }
  const seen = new Set<string>();
  return value.map((entry: unknown, index) => {
    const at = `${where}[${index}]`;
    if (typeof entry !== "object" || entry === null) throw new TestimonialError(at, "expected an object");
    const { handle, lang, text, postUrl, translations, ...rest } = entry as Record<string, unknown>;
    const extra = Object.keys(rest);
    if (extra.length) throw new TestimonialError(at, `unknown keys ${extra.join(", ")}`);
    if (typeof handle !== "string" || !X_HANDLE.test(handle)) {
      throw new TestimonialError(at, "handle must be an X username without @");
    }
    if (seen.has(handle.toLowerCase())) throw new TestimonialError(at, `@${handle} appears twice`);
    seen.add(handle.toLowerCase());
    if (typeof lang !== "string" || !lang.trim()) throw new TestimonialError(at, "lang is required");
    if (typeof text !== "string" || !text.trim()) throw new TestimonialError(at, "text is required");
    if (postUrl !== undefined && (typeof postUrl !== "string" || !X_POST_URL.test(postUrl))) {
      throw new TestimonialError(at, "postUrl must be https://x.com/<handle>/status/<id>");
    }
    if (typeof translations !== "object" || translations === null || Array.isArray(translations)) {
      throw new TestimonialError(at, "translations must map site locales to text");
    }
    const unknown = Object.keys(translations).filter((key) => !(LOCALES as readonly string[]).includes(key));
    if (unknown.length) throw new TestimonialError(at, `unknown translation locales ${unknown.join(", ")}`);
    const texts = translations as Record<string, unknown>;
    for (const locale of LOCALES) {
      const translated = texts[locale];
      if (translated === undefined) {
        if (LOCALE_TAGS[locale] !== lang) throw new TestimonialError(at, `needs a ${locale} translation`);
      } else if (typeof translated !== "string" || !translated.trim()) {
        throw new TestimonialError(at, `${locale} translation is empty`);
      }
    }
    return {
      handle,
      lang,
      text,
      ...(postUrl ? { postUrl } : {}),
      translations: translations as Partial<Record<Locale, string>>,
    };
  });
}

/** The post's text and language as the `locale` page shows it. */
export function textFor(source: TestimonialSource, locale: Locale): { lang: string; text: string } {
  const translated = source.translations[locale];
  return translated ? { lang: LOCALE_TAGS[locale], text: translated } : { lang: source.lang, text: source.text };
}

/** Paragraphs of a post, split on blank lines. */
export const paragraphsOf = (text: string): string[] =>
  text
    .split(/\n\s*\n/)
    .map((paragraph) => paragraph.trim())
    .filter(Boolean);

/** The profile page of an X handle. */
export const xProfileUrl = (handle: string) => `https://x.com/${handle}`;
