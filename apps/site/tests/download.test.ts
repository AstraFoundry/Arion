import { describe, expect, it } from "vitest";
import { chooseDownload, type ChannelSummary } from "@/features/download-release";
import { centerDownloadUrl, centerReleaseUrl } from "@/shared/config";

const release = (version: string, prerelease: boolean) => ({
  version,
  prerelease,
  publishedAt: "2026-10-20T08:00:00Z",
  notes: "",
  installer: { name: `Arion.${version}.arm64.dmg`, size: 1, sha256: "0".repeat(64), url: "https://api.arion.astrafoundry.si/v1/updates/x" },
});
const empty = (channel: "stable" | "beta"): ChannelSummary => ({ channel, targets: {} });

describe("chooseDownload", () => {
  it("prefers the stable release of a target", () => {
    const stable: ChannelSummary = { channel: "stable", targets: { "darwin-arm64": release("0.2.0", false) } };
    const beta: ChannelSummary = { channel: "beta", targets: { "darwin-arm64": release("0.3.0-beta.1", true) } };
    expect(chooseDownload("darwin-arm64", stable, beta)).toMatchObject({ kind: "stable", release: { version: "0.2.0" } });
  });

  it("offers the preview when a target has no stable release", () => {
    const beta: ChannelSummary = { channel: "beta", targets: { "darwin-arm64": release("0.1.0-alpha.6", true) } };
    expect(chooseDownload("darwin-arm64", empty("stable"), beta)).toMatchObject({ kind: "preview" });
    expect(chooseDownload("windows-amd64", empty("stable"), beta)).toEqual({ kind: "unavailable" });
  });
});

describe("center links", () => {
  it("are built from the configured center origin", () => {
    expect(centerReleaseUrl("beta")).toBe("https://api.arion.astrafoundry.si/v1/releases/beta");
    expect(centerDownloadUrl("stable", "windows-amd64")).toBe(
      "https://api.arion.astrafoundry.si/v1/downloads/stable/windows-amd64",
    );
  });
});
