import "server-only";
import rehypeShiki from "@shikijs/rehype";
import type { Root } from "hast";
import rehypeRaw from "rehype-raw";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import remarkRehype from "remark-rehype";
import { unified } from "unified";
import { mediaSize, type MediaRef, resolveMedia } from "@/shared/lib/server";
import {
  collectHeadings,
  type PostHeading,
  rehypeFigures,
  rehypeMediaSize,
  rehypeResolveMedia,
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

/** Parse, trusted raw HTML, validation, figures and media URLs: the shared front half. */
function basePipeline(file: string, refs: Map<string, MediaRef>, labels?: MarkdownLabels) {
  const footnotes = labels && {
    footnoteLabel: labels.footnotes,
    footnoteLabelTagName: "h2",
    footnoteLabelProperties: { className: ["post-footnotes-label"] },
    footnoteBackLabel: (referenceIndex: number) => labels.footnoteBack(referenceIndex + 1),
  };
  return unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkRehype, { allowDangerousHtml: true, ...footnotes })
    .use(rehypeRaw)
    .use(rehypeStaticHtml)
    .use(rehypeFigures)
    .use(rehypeResolveMedia, { resolve: (src) => resolveMedia(src, file), refs });
}

/**
 * Markdown (GFM: tables, footnotes, task lists) with trusted raw HTML, to a
 * hast tree ready for React. Runs at build time only: code is highlighted
 * and media resolved and measured here, so the page ships no Markdown or
 * highlighter code. `file` is the post's path; relative media start there.
 */
export async function renderMarkdown(
  markdown: string,
  file: string,
  labels: MarkdownLabels,
): Promise<RenderedMarkdown> {
  const refs = new Map<string, MediaRef>();
  const processor = basePipeline(file, refs, labels)
    .use(rehypeMediaSize, {
      measure: (url) => {
        const ref = refs.get(url);
        if (!ref) throw new Error(`${url} was not resolved before measuring`);
        return mediaSize(ref);
      },
    })
    .use(rehypeSlug)
    .use(rehypeShiki, {
      themes: { light: "github-light", dark: "github-dark" },
      defaultColor: false,
    });

  const tree = (await processor.run(processor.parse(markdown))) as Root;
  return { tree, headings: collectHeadings(tree, FOOTNOTES_LABEL_ID) };
}

/** Every media file a post body names, resolved; nothing is fetched or highlighted. */
export async function postMediaRefs(markdown: string, file: string): Promise<MediaRef[]> {
  const refs = new Map<string, MediaRef>();
  const processor = basePipeline(file, refs);
  await processor.run(processor.parse(markdown));
  return [...refs.values()];
}
