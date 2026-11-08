import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { siteConfig } from "@/shared/config";

const root = path.resolve(import.meta.dirname, "..");
const skill = readFileSync(path.join(root, "public/feedback.md"), "utf8");
const headers = readFileSync(path.join(root, "public/_headers"), "utf8");

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const file = path.join(dir, name);
    return statSync(file).isDirectory() ? sources(file) : [file];
  });
}

describe("feedback skill", () => {
  it("points agents at the repository and the community chats the site links", () => {
    expect(skill).toContain(siteConfig.repositoryUrl);
    expect(skill).toContain(siteConfig.community.discord);
    expect(skill).toContain(siteConfig.community.feishu);
  });

  it("is served as unindexed Markdown", () => {
    const block = headers.split(/\n(?=\S)/).find((rule) => rule.startsWith("/feedback.md\n"));
    expect(block).toBeDefined();
    expect(block).toMatch(/Content-Type: text\/markdown; charset=utf-8/);
    expect(block).toMatch(/X-Robots-Tag: noindex/);
  });

  it("is not linked from any page", () => {
    for (const dir of ["app", "src", "content"]) {
      for (const file of sources(path.join(root, dir))) {
        if (/\.(tsx?|json|md)$/.test(file)) {
          expect(readFileSync(file, "utf8"), file).not.toContain("feedback.md");
        }
      }
    }
  });
});
