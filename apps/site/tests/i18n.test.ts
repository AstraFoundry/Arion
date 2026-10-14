import { describe, expect, it } from "vitest";
import { format, localeAlternates, localizePath, parseLocalePath } from "@/shared/i18n";

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

describe("format", () => {
  it("fills named placeholders and leaves unknown ones intact", () => {
    expect(format("macOS {macos} or later", { macos: 13 })).toBe("macOS 13 or later");
    expect(format("{a} and {b}", { a: "x" })).toBe("x and {b}");
  });
});
