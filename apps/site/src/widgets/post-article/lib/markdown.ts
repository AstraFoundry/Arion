import "server-only";
import rehypeShiki from "@shikijs/rehype";
import type { Root } from "hast";
import rehypeRaw from "rehype-raw";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { unified } from "unified";
import { publicImageSize } from "@/shared/lib/server";
import {
  collectHeadings,
  type PostHeading,
  rehypeFigures,
  rehypeMediaSize,
  rehypeStaticHtml,
} from "./rehype-post";

/** The id remark-rehype gives the footnotes heading. */
export const FOOTNOTES_LABEL_ID = "footnote-label";

export interface MarkdownLabels {
  footnotes: string;
  footnoteBack: (index: number) => string;
}

export interface RenderedMarkdown {
  tree: Root;
  headings: PostHeading[];
}

/**
 * Markdown (GFM: tables, footnotes, task lists) with trusted raw HTML, to a
 * hast tree ready for React. Runs at build time only: code is highlighted
 * and media measured here, so the page ships no Markdown or highlighter code.
 */
export async function renderMarkdown(
  markdown: string,
  labels: MarkdownLabels,
): Promise<RenderedMarkdown> {
  const processor = unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkRehype, {
      allowDangerousHtml: true,
      footnoteLabel: labels.footnotes,
      footnoteLabelTagName: "h2",
      footnoteLabelProperties: { className: ["post-footnotes-label"] },
      footnoteBackLabel: (referenceIndex) => labels.footnoteBack(referenceIndex + 1),
    })
    .use(rehypeRaw)
    .use(rehypeStaticHtml)
    .use(rehypeFigures)
    .use(rehypeMediaSize, { measure: publicImageSize })
    .use(rehypeSlug)
    .use(rehypeShiki, {
      themes: { light: "github-light", dark: "github-dark" },
      defaultColor: false,
    });

  const tree = (await processor.run(processor.parse(markdown))) as Root;
  return { tree, headings: collectHeadings(tree, FOOTNOTES_LABEL_ID) };
}
