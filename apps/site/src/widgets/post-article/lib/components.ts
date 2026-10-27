import type { Components } from "hast-util-to-jsx-runtime";
import { CodeBlock } from "../ui/embeds/code-block";
import { EmbedFrame } from "../ui/embeds/embed-frame";
import { LazyImage } from "../ui/embeds/lazy-image";
import { PillFigure } from "../ui/embeds/pill-figure";
import { VideoClip } from "../ui/embeds/video-clip";
import { H2, H3, Note, ProseLink, ProseTable } from "../ui/prose-elements";

/**
 * How post elements render. Standard Markdown and HTML elements get the
 * prose behaviors; the hyphenated names are the embeds a post can use in raw
 * HTML (docs/frontend/blog.md lists their attributes). `rehype-post.ts`
 * validates their attributes at build time.
 */
export const postComponents = {
  a: ProseLink,
  aside: Note,
  h2: H2,
  h3: H3,
  img: LazyImage,
  pre: CodeBlock,
  table: ProseTable,
  "dictation-pill": PillFigure,
  "embed-frame": EmbedFrame,
  "video-clip": VideoClip,
} as unknown as Components;
