import type { Element, Root } from "hast";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { unified } from "unified";
import { describe, expect, it } from "vitest";
import {
  parseFrontmatter,
  type PostMeta,
  sortNewestFirst,
  translationMismatch,
} from "@/entities/post/model/post";
import {
  rehypeFigures,
  rehypeMediaSize,
  rehypeStaticHtml,
} from "@/widgets/post-article/lib/rehype-post";

const valid = {
  title: "A post",
  summary: "What it says.",
  date: "2026-10-27",
  category: "product",
};

describe("post front matter", () => {
  it("accepts a complete post and a YAML date", () => {
    expect(parseFrontmatter(valid, "en.md")).toEqual(valid);
    expect(parseFrontmatter({ ...valid, date: new Date("2026-10-27T00:00:00Z") }, "en.md").date).toBe(
      "2026-10-27",
    );
  });

  it("pairs the cover with its alt text", () => {
    const cover = { ...valid, cover: "/blog/a/cover.webp", coverAlt: "A cover" };
    expect(parseFrontmatter(cover, "en.md").cover).toEqual({ src: "/blog/a/cover.webp", alt: "A cover" });
    expect(() => parseFrontmatter({ ...valid, cover: "/blog/a/cover.webp" }, "en.md")).toThrow(/coverAlt/);
  });

  it("rejects missing, unknown and malformed fields", () => {
    expect(() => parseFrontmatter({ ...valid, title: "" }, "en.md")).toThrow(/title/);
    expect(() => parseFrontmatter({ ...valid, author: "x" }, "en.md")).toThrow(/unknown front matter "author"/);
    expect(() => parseFrontmatter({ ...valid, category: "news" }, "en.md")).toThrow(/category/);
    expect(() => parseFrontmatter({ ...valid, date: "27/10/2026" }, "en.md")).toThrow(/date/);
    expect(() => parseFrontmatter(undefined, "en.md")).toThrow(/missing front matter/);
  });
});

const meta = (slug: string, date: string, extra: Partial<PostMeta> = {}): PostMeta => ({
  slug,
  locale: "en",
  title: slug,
  summary: slug,
  date,
  category: "product",
  ...extra,
});

describe("post lists", () => {
  it("sorts newest first, then by slug", () => {
    const sorted = sortNewestFirst([meta("b", "2026-01-01"), meta("c", "2026-03-01"), meta("a", "2026-01-01")]);
    expect(sorted.map((post) => post.slug)).toEqual(["c", "a", "b"]);
  });

  it("finds translations that disagree", () => {
    expect(translationMismatch([meta("a", "2026-01-01"), meta("a", "2026-01-01")])).toBeUndefined();
    expect(translationMismatch([meta("a", "2026-01-01"), meta("a", "2026-01-02")])).toBe("date");
    expect(
      translationMismatch([meta("a", "2026-01-01"), meta("a", "2026-01-01", { category: "engineering" })]),
    ).toBe("category");
  });
});

async function hast(markdown: string): Promise<Root> {
  const processor = unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkRehype, { allowDangerousHtml: true })
    .use(rehypeRaw)
    .use(rehypeStaticHtml)
    .use(rehypeFigures)
    .use(rehypeMediaSize, { measure: () => ({ width: 800, height: 600 }) });
  return (await processor.run(processor.parse(markdown))) as Root;
}

const elements = (tree: Root) => tree.children.filter((node): node is Element => node.type === "element");

describe("post markdown", () => {
  it("turns a lone image into a sized figure with its caption", async () => {
    const [figure] = elements(await hast('![A chart](/blog/a/chart.webp "The caption")'));
    expect(figure.tagName).toBe("figure");
    const [image, caption] = figure.children as Element[];
    expect(image.properties).toMatchObject({ src: "/blog/a/chart.webp", alt: "A chart", width: 800, height: 600 });
    expect(image.properties.title).toBeUndefined();
    expect(caption.tagName).toBe("figcaption");
  });

  it("requires a size for remote images", async () => {
    await expect(hast("![A chart](https://example.com/a.png)")).rejects.toThrow(/width and height/);
  });

  it("lifts a one-line embed out of its paragraph", async () => {
    const tree = await hast('<embed-frame src="/blog/a/demo.html" title="Demo" ratio="16/9"></embed-frame>');
    expect(elements(tree).map((node) => node.tagName)).toEqual(["embed-frame"]);
  });

  it("rejects embeds inside a sentence and invalid embed attributes", async () => {
    await expect(
      hast('Try <embed-frame src="/a.html" title="Demo" ratio="16/9"></embed-frame> now.'),
    ).rejects.toThrow(/own line/);
    await expect(hast('<embed-frame src="/a.html" title="Demo"></embed-frame>')).rejects.toThrow(/ratio/);
    await expect(hast('<dictation-pill state="learned" label="Pill"></dictation-pill>')).rejects.toThrow(/term/);
  });

  it("keeps posts static", async () => {
    await expect(hast("<script>alert(1)</script>")).rejects.toThrow(/<script>/);
    await expect(hast('<iframe src="https://example.com"></iframe>')).rejects.toThrow(/embed-frame/);
    await expect(hast('<button onclick="x()">Go</button>')).rejects.toThrow(/event handler/);
  });
});
