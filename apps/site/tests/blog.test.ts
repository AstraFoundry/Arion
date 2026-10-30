import type { Element, Root } from "hast";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { unified } from "unified";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it, vi } from "vitest";
import {
  parseFrontmatter,
  type PostMeta,
  sortNewestFirst,
  translationMismatch,
} from "@/entities/post/model/post";
import { authorAvatars, getAuthors } from "@/entities/post/api/authors";
import { mediaSize, PROJECT_ASSET_BASE, resolveMedia } from "@/shared/lib/server";
import {
  rehypeFigures,
  rehypeMediaSize,
  rehypeResolveMedia,
  rehypeStaticHtml,
} from "@/widgets/post-article/lib/rehype-post";

const valid = {
  title: "A post",
  description: "What it says.",
  date: "2026-10-27",
  category: "product",
};

const defaults = { authors: [], tags: [], draft: false };

describe("post front matter", () => {
  it("accepts a complete post and a YAML date", () => {
    expect(parseFrontmatter(valid, "en.md")).toEqual({ ...valid, ...defaults });
    expect(parseFrontmatter({ ...valid, date: new Date("2026-10-27T00:00:00Z") }, "en.md").date).toBe(
      "2026-10-27",
    );
  });

  it("reads the optional standard fields", () => {
    const full = {
      ...valid,
      updated: "2026-11-02",
      authors: "ada, lena-k",
      tags: ["privacy", "offline"],
      image: "./cover.webp",
      imageAlt: "A cover",
      draft: true,
    };
    expect(parseFrontmatter(full, "en.md")).toEqual({
      ...valid,
      updated: "2026-11-02",
      authors: ["ada", "lena-k"],
      tags: ["privacy", "offline"],
      image: { src: "./cover.webp", alt: "A cover" },
      draft: true,
    });
  });

  it("rejects malformed standard fields", () => {
    expect(() => parseFrontmatter({ ...valid, image: "./cover.webp" }, "en.md")).toThrow(/imageAlt/);
    expect(() => parseFrontmatter({ ...valid, updated: "2026-01-01" }, "en.md")).toThrow(/before "date"/);
    expect(parseFrontmatter({ ...valid, authors: ["ada", "Lena"] }, "en.md").authors).toEqual(["ada", "Lena"]);
    expect(() => parseFrontmatter({ ...valid, authors: "Ada Lovelace" }, "en.md")).toThrow(/not a GitHub login/);
    expect(() => parseFrontmatter({ ...valid, authors: "ada,,lena" }, "en.md")).toThrow(/not a GitHub login/);
    expect(() => parseFrontmatter({ ...valid, authors: "-ada" }, "en.md")).toThrow(/not a GitHub login/);
    expect(() => parseFrontmatter({ ...valid, authors: "ada, Ada" }, "en.md")).toThrow(/twice/);
    expect(() => parseFrontmatter({ ...valid, authors: [1] }, "en.md")).toThrow(/separated by commas/);
    expect(() => parseFrontmatter({ ...valid, tags: ["a", "a"] }, "en.md")).toThrow(/twice/);
    expect(() => parseFrontmatter({ ...valid, draft: "yes" }, "en.md")).toThrow(/draft/);
    expect(() => parseFrontmatter({ ...valid, summary: "x" }, "en.md")).toThrow(/unknown front matter "summary"/);
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
  description: slug,
  date,
  authors: [],
  tags: [],
  category: "product",
  draft: false,
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
    expect(translationMismatch([meta("a", "2026-01-01"), meta("a", "2026-01-01", { draft: true })])).toBe(
      "draft",
    );
    expect(
      translationMismatch([meta("a", "2026-01-01", { authors: ["ada"] }), meta("a", "2026-01-01")]),
    ).toBe("authors");
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
    .use(rehypeResolveMedia, {
      resolve: (src) =>
        src.startsWith("https://")
          ? { url: src }
          : { url: src.startsWith("/") ? src : `/resolved/${src}`, file: `/repo/${src}` },
    })
    .use(rehypeMediaSize, { measure: async () => ({ width: 800, height: 600 }) });
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

  it("resolves project paths and measures remote images without a referrer", async () => {
    const [local] = elements(await hast("![A chart](../shared/chart.png)"));
    expect((local.children[0] as Element).properties).toMatchObject({ src: "/resolved/../shared/chart.png" });

    const [remote] = elements(await hast("![A chart](https://img.example.com/a.png)"));
    expect((remote.children[0] as Element).properties).toMatchObject({
      src: "https://img.example.com/a.png",
      referrerPolicy: "no-referrer",
      width: 800,
      height: 600,
    });
  });

  it("keeps a stated size and rejects half of one", async () => {
    const [stated] = elements(
      await hast('<img src="https://img.example.com/a.png" alt="A" width="20" height="10">'),
    );
    expect(stated.properties).toMatchObject({ width: 20, height: 10 });
    await expect(hast('<img src="https://img.example.com/a.png" alt="A" width="20">')).rejects.toThrow(
      /only one of width and height/,
    );
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

describe("media resolution", () => {
  const post = path.resolve("content/blog/where-your-dictation-goes/en.md");

  it("keeps remote and public URLs and publishes project files by content hash", () => {
    expect(resolveMedia("https://img.example.com/a.png", post)).toEqual({ url: "https://img.example.com/a.png" });
    expect(resolveMedia("/og.jpg", post)).toEqual({ url: "/og.jpg", file: path.resolve("public/og.jpg") });

    const project = resolveMedia("../../../../desktop/resources/icon.png", post);
    expect(project.file).toBe(path.resolve("../desktop/resources/icon.png"));
    expect(project.url).toMatch(new RegExp(`^${PROJECT_ASSET_BASE}[0-9a-f]{12}/icon\\.png$`));
  });

  it("rejects missing files, paths outside the repository and other file types", () => {
    expect(() => resolveMedia("./missing.png", post)).toThrow(/names no file/);
    expect(() => resolveMedia("/missing.png", post)).toThrow(/names no file/);
    expect(() => resolveMedia("../../../../../../outside.png", post)).toThrow(/outside the repository/);
    expect(() => resolveMedia("./en.md", post)).toThrow(/project paths may name/);
    expect(() => resolveMedia("data:image/png;base64,AA", post)).toThrow(/media URLs/);
  });

  it("measures local files and downloads remote images once", async () => {
    expect(await mediaSize({ url: "/og.jpg", file: path.resolve("public/og.jpg") })).toEqual({
      width: 1200,
      height: 630,
    });

    const bytes = readFileSync(path.resolve("public/og.jpg"));
    const fetchMock = vi.fn(async () => new Response(bytes));
    vi.stubGlobal("fetch", fetchMock);
    try {
      const url = "https://img.example.com/og.jpg";
      expect(await mediaSize({ url })).toEqual({ width: 1200, height: 630 });
      expect(await mediaSize({ url })).toEqual({ width: 1200, height: 630 });
      expect(fetchMock).toHaveBeenCalledTimes(1);

      fetchMock.mockResolvedValueOnce(new Response("gone", { status: 404 }));
      await expect(mediaSize({ url: "https://img.example.com/gone.png" })).rejects.toThrow(/HTTP 404/);
    } finally {
      vi.unstubAllGlobals();
    }
  });
});

describe("post authors", () => {
  const avatar = readFileSync(path.resolve("public/og.jpg"));
  const user = (login: string, name: string | null) => ({
    login,
    name,
    html_url: `https://github.com/${login}`,
    avatar_url: `https://avatars.githubusercontent.com/u/1?v=4`,
  });

  function github(users: Record<string, ReturnType<typeof user>>) {
    return vi.fn(async (input: string | URL, init?: RequestInit) => {
      const url = new URL(input);
      if (url.host === "avatars.githubusercontent.com") {
        return new Response(avatar, { headers: { "Content-Type": "image/jpeg" } });
      }
      const found = users[url.pathname.replace("/users/", "").toLowerCase()];
      const headers = new Headers(init?.headers);
      if (!headers.get("Accept")?.includes("github")) return new Response("", { status: 400 });
      return found ? Response.json(found) : new Response("", { status: 404 });
    });
  }

  it("reads each profile once and publishes its avatar by content hash", async () => {
    const fetchMock = github({ ada: user("Ada", "Ada Lovelace"), lena: user("lena", null) });
    vi.stubGlobal("fetch", fetchMock);
    try {
      const authors = await getAuthors(["ada", "lena"]);
      expect(authors).toEqual([
        {
          login: "Ada",
          name: "Ada Lovelace",
          profileUrl: "https://github.com/Ada",
          avatar: expect.stringMatching(new RegExp(`^${PROJECT_ASSET_BASE}[0-9a-f]{12}/ada\\.jpg$`)),
        },
        expect.objectContaining({ login: "lena", name: "lena" }),
      ]);
      expect(fetchMock.mock.calls.map(([url]) => String(url))).toContain(
        "https://avatars.githubusercontent.com/u/1?v=4&s=64",
      );

      await getAuthors(["ADA"]);
      expect(fetchMock).toHaveBeenCalledTimes(4);
      const [[url, bytes]] = await authorAvatars(["ada"]);
      expect(url).toBe(authors[0].avatar);
      expect(bytes).toEqual(new Uint8Array(avatar));
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("fails the build for an unknown login or a rate limit", async () => {
    vi.stubGlobal("fetch", github({}));
    try {
      await expect(getAuthors(["nobody"])).rejects.toThrow(/Blog author "nobody": .*HTTP 404/);
      vi.stubGlobal("fetch", vi.fn(async () => new Response("", { status: 403 })));
      await expect(getAuthors(["limited"])).rejects.toThrow(/GITHUB_TOKEN/);
    } finally {
      vi.unstubAllGlobals();
    }
  });
});
