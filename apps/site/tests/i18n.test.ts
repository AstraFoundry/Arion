import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { format, localeAlternates, localizePath, parseLocalePath, LOCALES } from "@/shared/i18n";

describe("locale paths", () => {
  it("prefixes every locale but the default", () => {
    expect(localizePath("/", "en")).toBe("/");
    expect(localizePath("/privacy/", "en")).toBe("/privacy/");
    expect(localizePath("/", "zh")).toBe("/zh/");
    expect(localizePath("/self-hosting/", "ja")).toBe("/ja/self-hosting/");
    expect(localizePath("/#open-source", "zh")).toBe("/zh/#open-source");
  });

  it("leaves external and protocol-relative URLs alone", () => {
    expect(localizePath("https://astrafoundry.si/account", "zh")).toBe(
      "https://astrafoundry.si/account",
    );
    expect(localizePath("//cdn.example.com/a.js", "ja")).toBe("//cdn.example.com/a.js");
  });

  it("parses the locale back out of a pathname", () => {
    expect(parseLocalePath("/")).toEqual({ locale: "en", path: "/" });
    expect(parseLocalePath("/credits/")).toEqual({ locale: "en", path: "/credits/" });
    expect(parseLocalePath("/zh/")).toEqual({ locale: "zh", path: "/" });
    expect(parseLocalePath("/ja/privacy/")).toEqual({ locale: "ja", path: "/privacy/" });
    expect(parseLocalePath("/en/privacy/")).toEqual({ locale: "en", path: "/en/privacy/" });
  });

  it("lists every translation as an alternate", () => {
    expect(localeAlternates("/privacy/", "ja")).toEqual({
      canonical: "/ja/privacy/",
      languages: {
        en: "/privacy/",
        "zh-CN": "/zh/privacy/",
        ja: "/ja/privacy/",
        "x-default": "/privacy/",
      },
    });
  });
});

/** The sentence the release-post dictation stage types for this locale. */
function stageDictation(locale: (typeof LOCALES)[number]): string {
  const html = readFileSync(path.resolve(`public/blog/arion-0-1-1/stage.${locale}.html`), "utf8");
  const block = html.match(/"dictation": \{[\s\S]*?\n        \}/);
  const final = block?.[0].match(/"final": "([^"]+)"/);
  if (!final?.[1]) throw new Error(`stage.${locale}.html has no dictation final`);
  return final[1];
}

describe("hero demo language", () => {
  it("types the page language's sentence, the same line as the release-post stage", () => {
    const sentences = LOCALES.map((locale) => {
      const messages = JSON.parse(
        readFileSync(path.resolve(`src/shared/i18n/messages/${locale}.json`), "utf8"),
      ) as { demo: { sentence: string } };
      expect(messages.demo.sentence, locale).toBe(stageDictation(locale));
      return messages.demo.sentence;
    });
    expect(new Set(sentences).size).toBe(LOCALES.length);
  });

  it("embeds each translation's stage recording from that locale's file", () => {
    for (const locale of LOCALES) {
      const markdown = readFileSync(path.resolve(`content/blog/arion-0-1-1/${locale}.md`), "utf8");
      const sources = [...markdown.matchAll(/<embed-frame src="([^"]+)"/g)].map((match) => match[1]);
      expect(sources.length).toBeGreaterThan(0);
      for (const source of sources) {
        expect(source, locale).toContain(`/stage.${locale}.html`);
      }
    }
  });
});

describe("format", () => {
  it("fills named placeholders and leaves unknown ones intact", () => {
    expect(format("macOS {macos} or later", { macos: 13 })).toBe("macOS 13 or later");
    expect(format("{a} and {b}", { a: "x" })).toBe("x and {b}");
  });
});
