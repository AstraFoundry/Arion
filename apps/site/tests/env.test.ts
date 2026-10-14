import { describe, expect, it } from "vitest";
import { parsePublicEnv, PublicEnvError, siteConfig } from "@/shared/config";

const valid = {
  siteUrl: "https://arion.astrafoundry.si",
  platformUrl: "https://astrafoundry.si",
};

describe("parsePublicEnv", () => {
  it("normalizes both origins", () => {
    expect(
      parsePublicEnv({ siteUrl: " https://arion.example.com/ ", platformUrl: "http://localhost:8787" }),
    ).toEqual({ siteUrl: "https://arion.example.com", platformUrl: "http://localhost:8787" });
  });

  it("requires both values", () => {
    expect(() => parsePublicEnv({ ...valid, siteUrl: undefined })).toThrow(/NEXT_PUBLIC_SITE_URL/);
    expect(() => parsePublicEnv({ ...valid, platformUrl: " " })).toThrow(
      /NEXT_PUBLIC_PLATFORM_URL/,
    );
  });

  it("rejects anything but an http(s) origin", () => {
    for (const platformUrl of ["astrafoundry.si", "ftp://astrafoundry.si", "https://astrafoundry.si/account", "https://astrafoundry.si/?a=1"]) {
      expect(() => parsePublicEnv({ ...valid, platformUrl })).toThrow(PublicEnvError);
    }
  });
});

describe("siteConfig", () => {
  it("builds the platform links from the configured origin", () => {
    expect(siteConfig.accountUrl).toBe("https://astrafoundry.si/account");
    expect(siteConfig.hostedSyncWaitlistUrl).toBe(
      "https://astrafoundry.si/products/arion/waitlist/hosted-sync",
    );
  });
});
