import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { paragraphsOf, parseTestimonials, textFor } from "@/entities/testimonial/model/testimonial";

describe("testimonials", () => {
  it("accepts the published content file", () => {
    const file = path.join(process.cwd(), "content", "testimonials", "testimonials.json");
    const entries = parseTestimonials(JSON.parse(readFileSync(file, "utf8")), "content/testimonials/testimonials.json");
    expect(entries.length).toBeGreaterThan(0);
  });

  it("rejects malformed entries", () => {
    const entry = { handle: "someone", lang: "zh-CN", text: "好用", translations: { en: "Good", ja: "良い" } };
    expect(() => parseTestimonials([], "t")).toThrow(/non-empty/);
    expect(() => parseTestimonials([{ ...entry, handle: "@someone" }], "t")).toThrow(/without @/);
    expect(() => parseTestimonials([entry, { ...entry, handle: "SomeOne" }], "t")).toThrow(/twice/);
    expect(() => parseTestimonials([{ ...entry, text: " " }], "t")).toThrow(/text/);
    expect(() => parseTestimonials([{ ...entry, extra: 1 }], "t")).toThrow(/unknown keys extra/);
    expect(() => parseTestimonials([{ ...entry, postUrl: "https://twitter.com/a/status/1" }], "t")).toThrow(
      /postUrl/,
    );
  });

  it("requires a translation for every other-language locale", () => {
    const entry = { handle: "someone", lang: "zh-CN", text: "好用", translations: { en: "Good" } };
    expect(() => parseTestimonials([entry], "t")).toThrow(/needs a ja translation/);
    expect(() => parseTestimonials([{ ...entry, translations: { en: "Good", ja: "良い", fr: "Bon" } }], "t")).toThrow(
      /unknown translation locales fr/,
    );
  });

  it("shows the translation for the page locale", () => {
    const [source] = parseTestimonials(
      [{ handle: "someone", lang: "zh-CN", text: "好用", translations: { en: "Good", ja: "良い" } }],
      "t",
    );
    expect(textFor(source, "en")).toEqual({ lang: "en", text: "Good" });
    expect(textFor(source, "zh")).toEqual({ lang: "zh-CN", text: "好用" });
  });

  it("splits paragraphs on blank lines", () => {
    expect(paragraphsOf("一\n\n二\n  \n三")).toEqual(["一", "二", "三"]);
  });
});
