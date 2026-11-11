// Markdown to HTML, on the server at build time: GitHub-flavored markdown,
// heading anchors that match GitHub's, links rewritten for the site, and code
// highlighted by Shiki for both color schemes.

import { codeToKeyedTokens } from "@shikijs/magic-move/core"
import type { KeyedTokensInfo } from "@shikijs/magic-move/types"
import GithubSlugger from "github-slugger"
import type { Element, ElementContent, Root, RootContent } from "hast"
import type { Root as MdRoot } from "mdast"
import { toString as mdToString } from "mdast-util-to-string"
import { toString } from "hast-util-to-string"
import rehypeStringify from "rehype-stringify"
import remarkGfm from "remark-gfm"
import remarkParse from "remark-parse"
import remarkRehype from "remark-rehype"
import { createHighlighter, type BundledLanguage, type Highlighter } from "shiki"
import { unified } from "unified"
import { visit } from "unist-util-visit"

import type { TocItem } from "@/lib/docs"
import { omitRepositoryOnly, rewriteHref, type LinkOptions } from "@/lib/markdown-source"

export { slugOf } from "@/lib/markdown-source"

/** The text of a page under one heading, for search. */
export interface Section {
  id: string
  heading: string
  text: string
}

export interface RenderedMarkdown {
  title: string
  /** The first paragraph, as text. */
  lead: string
  html: string
  toc: TocItem[]
  sections: Section[]
}

const themes = { light: "github-light-default", dark: "github-dark-default" } as const

let highlighter: Promise<Highlighter> | undefined
function getHighlighter() {
  highlighter ??= createHighlighter({
    themes: Object.values(themes),
    langs: ["go", "ts", "tsx", "js", "json", "sh", "css", "html", "xml", "yaml"],
  })
  return highlighter
}

export async function renderMarkdown(source: string, file: string, links: LinkOptions): Promise<RenderedMarkdown> {
  const shiki = await getHighlighter()
  const out: RenderedMarkdown = { title: "", lead: "", html: "", toc: [], sections: [] }
  const processor = unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(() => omitRepositoryOnly)
    .use(() => previews)
    .use(remarkRehype)
    .use(() => (tree: Root) => transform(tree, out, shiki, file, links))
    .use(rehypeStringify)
  out.html = String(await processor.process(source))
  return out
}

function transform(tree: Root, out: RenderedMarkdown, shiki: Highlighter, file: string, links: LinkOptions) {
  const slugger = new GithubSlugger()
  let section: Section = { id: "", heading: "", text: "" }
  out.sections.push(section)

  // Headings are top-level, so one pass finds the title, the table of
  // contents and the sections in document order.
  tree.children = tree.children.flatMap((node): RootContent[] => {
    if (node.type !== "element") return [node]
    const depth = headingDepth(node)
    if (depth === 1 && !out.title) {
      out.title = section.heading = toString(node)
      slugger.slug(out.title) // GitHub counts the title in its anchors too
      return []
    }
    if (depth >= 2) {
      const text = toString(node)
      const id = slugger.slug(text)
      node.properties.id = id
      if (!containsLink(node)) {
        node.children = [{ type: "element", tagName: "a", properties: { href: `#${id}` }, children: node.children }]
      }
      if (depth <= 3) {
        out.toc.push({ id, text, depth })
        section = { id, heading: text, text: "" }
        out.sections.push(section)
      }
      return [node]
    }
    const text = toString(node).replace(/\s+/g, " ").trim()
    if (node.tagName === "p" && !out.lead) out.lead = text
    if (node.tagName !== "pre") section.text += " " + text
    return [node]
  })
  for (const s of out.sections) s.text = s.text.trim()

  visit(tree, "element", (node, index, parent) => {
    if (node.tagName === "a" && typeof node.properties.href === "string") {
      rewriteLink(node, node.properties.href, file, links)
    }
    if (!parent || index === undefined) return
    if (node.tagName === "pre") {
      parent.children[index] = codeBlock(node, shiki)
      return "skip"
    }
    if (node.tagName === "table") {
      parent.children[index] = { type: "element", tagName: "div", properties: { className: ["table-wrap"] }, children: [node] }
      return "skip"
    }
  })

  cardLists(tree)
}

const preview = /^<!--\s*(preview|app)\s+([\w-]+)\s+(\d+)\s*-->$/

/**
 * A comment `<!-- preview button 140 -->`, which GitHub does not show,
 * becomes a frame of the demo of that name (website/docs-demo), as high
 * as it says, in a window of macOS titled as the page. `<!-- app mayoi
 * 480 -->` runs the example app of that name (examples/mayoi, one of
 * docs-demo.ts's apps), in a window titled as the app.
 */
function previews(tree: MdRoot) {
  const heading = tree.children.find((node) => node.type === "heading" && node.depth === 1)
  const pageTitle = heading ? mdToString(heading) : ""
  visit(tree, "html", (node, index, parent) => {
    const m = preview.exec(node.value.trim())
    if (!m || !parent || index === undefined) return
    const [, kind, name, height] = m
    const isApp = kind === "app"
    const title = isApp ? name[0].toUpperCase() + name.slice(1) : pageTitle
    const el = (tagName: string, className: string, children: ElementContent[] = [], properties: Element["properties"] = {}): Element => ({
      type: "element",
      tagName,
      properties: { className: [className], ...properties },
      children,
    })
    const src = isApp ? `/docs-demo?app=${name}` : `/docs-demo?demo=${name}`
    // Lucide's square-arrow-out-up-right.
    const icon: Element = {
      type: "element",
      tagName: "svg",
      properties: {
        viewBox: "0 0 24 24", width: 14, height: 14, fill: "none", stroke: "currentColor",
        strokeWidth: 2, strokeLinecap: "round", strokeLinejoin: "round", ariaHidden: "true",
      },
      children: ["M21 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h6", "m21 3-9 9", "M15 3h6v6"].map(
        (d): Element => ({ type: "element", tagName: "path", properties: { d }, children: [] }),
      ),
    }
    const titlebar = el("div", "ui-preview-titlebar", [
      el("span", "ui-preview-lights", [el("i", "close"), el("i", "minimize"), el("i", "zoom")], { ariaHidden: "true" }),
      el("span", "ui-preview-title", [{ type: "text", value: title || name }]),
      el("a", "ui-preview-open", [icon], {
        href: src,
        target: "_blank",
        rel: "noopener",
        title: "Open in a new tab",
        ariaLabel: `Open ${title || name} in a new tab`,
      }),
    ])
    const iframe = el("iframe", "ui-preview-frame", [], {
      src,
      title: isApp ? `${title}, running` : `A preview of ${title || name}`,
      loading: "lazy",
      style: `height:${height}px`,
    })
    // A node of its own type: remark-rehype makes it the element its data
    // names, where it drops HTML.
    parent.children[index] = {
      type: "uiPreview",
      data: { hName: "figure", hProperties: { className: ["ui-preview"] }, hChildren: [titlebar, iframe] },
    } as unknown as (typeof parent.children)[number]
  })
}

function headingDepth(node: Element) {
  const m = /^h([1-6])$/.exec(node.tagName)
  return m ? Number(m[1]) : 0
}

function containsLink(node: Element): boolean {
  return node.children.some((c) => c.type === "element" && (c.tagName === "a" || containsLink(c)))
}

function rewriteLink(node: Element, href: string, file: string, links: LinkOptions) {
  const rewritten = rewriteHref(href, file, links)
  node.properties.href = rewritten
  if (/^https?:/.test(rewritten)) node.properties.rel = ["noreferrer"]
}

/** A highlighted code block with a copy button, from remark-rehype's `<pre><code>`. */
function codeBlock(pre: Element, shiki: Highlighter): Element {
  const code = pre.children.find((c): c is Element => c.type === "element" && c.tagName === "code")
  const classes = (code?.properties.className as string[] | undefined) ?? []
  const lang = classes.find((c) => c.startsWith("language-"))?.slice("language-".length) ?? "text"
  return {
    type: "element",
    tagName: "div",
    properties: { className: ["code-block"] },
    children: [
      highlight(shiki, toString(code ?? pre).replace(/\n$/, ""), lang),
      {
        type: "element",
        tagName: "button",
        properties: { type: "button", className: ["copy-button"], ariaLabel: "Copy code" },
        children: [
          icon("icon-copy", ["M8 8h12a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H10a2 2 0 0 1-2-2z", "M4 16a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2"]),
          icon("icon-check", ["M20 6 9 17l-5-5"]),
        ],
      },
    ],
  }
}

function icon(className: string, paths: string[]): Element {
  return {
    type: "element",
    tagName: "svg",
    properties: {
      className: [className],
      viewBox: "0 0 24 24",
      fill: "none",
      stroke: "currentColor",
      strokeWidth: "2",
      strokeLinecap: "round",
      strokeLinejoin: "round",
      ariaHidden: "true",
    },
    children: paths.map((d) => ({ type: "element", tagName: "path", properties: { d }, children: [] })),
  }
}

function highlight(shiki: Highlighter, code: string, lang: string): Element {
  const known = shiki.getLoadedLanguages().includes(lang)
  const root = shiki.codeToHast(code, { lang: known ? lang : "text", themes, defaultColor: false })
  const pre = root.children[0] as Element
  // The page styles the background, which Shiki's inline style would override.
  delete pre.properties.style
  delete pre.properties.tabindex
  return pre
}

/** Code highlighted for both color schemes, as HTML. */
export async function highlightCode(code: string, lang: string) {
  const pre = highlight(await getHighlighter(), code, lang)
  return unified().use(rehypeStringify).stringify({ type: "root", children: [pre] })
}

/** Code highlighted for both color schemes, as the tokens of a magic move. */
export async function highlightTokens(code: string, lang: BundledLanguage): Promise<KeyedTokensInfo> {
  return codeToKeyedTokens(await getHighlighter(), code, { lang, themes, defaultColor: false })
}

/** Lists of pages become cards: `- [Title](page.md): what it covers`. */
function cardLists(tree: Root) {
  tree.children = tree.children.map((node) => {
    if (node.type !== "element" || node.tagName !== "ul") return node
    const items = node.children.filter((c): c is Element => c.type === "element" && c.tagName === "li")
    const cards = items.map(card)
    if (!cards.every((c) => c !== undefined)) return node
    return { type: "element", tagName: "div", properties: { className: ["doc-cards"] }, children: cards }
  })
}

function card(li: Element): Element | undefined {
  const [first] = li.children.filter((c) => c.type !== "text" || c.value.trim())
  const inline = first?.type === "element" && first.tagName === "p" ? first.children : li.children
  const [head, ...rest] = inline
  let href: unknown
  let title: ElementContent[]
  let description: ElementContent[]
  if (head?.type === "element" && head.tagName === "a") {
    // A page of the docs, then what it covers.
    href = head.properties.href
    if (typeof href !== "string" || !href.startsWith("/docs")) return
    if (rest[0]?.type !== "text" || !rest[0].value.startsWith(":")) return
    title = head.children
    description = rest
  } else if (head?.type === "text" && head.value.includes(": ")) {
    // "The Go API: `go doc -all <package>`, …" links to the package on pkg.go.dev.
    const [name = "", after = ""] = head.value.split(": ", 2)
    const pkg = /^go doc (?:-all )?(\S+)$/.exec(rest[0]?.type === "element" ? toString(rest[0]) : "")?.[1]
    if (!pkg) return
    href = `https://pkg.go.dev/${pkg}`
    title = [{ type: "text", value: name }]
    description = [{ type: "text", value: after }, ...rest]
  } else {
    return
  }
  return {
    type: "element",
    tagName: "a",
    properties: { href: String(href), className: ["doc-card"] },
    children: [
      { type: "element", tagName: "span", properties: { className: ["doc-card-title"] }, children: title },
      { type: "element", tagName: "span", properties: { className: ["doc-card-description"] }, children: capitalize(description) },
    ],
  }
}

/** Drops the ": " between a card's link and its description, and capitalizes the description. */
function capitalize(nodes: ElementContent[]): ElementContent[] {
  const [first, ...rest] = nodes
  if (first?.type !== "text") return nodes
  const value = first.value.replace(/^[\s:]+/, "")
  return [{ ...first, value: value.charAt(0).toUpperCase() + value.slice(1) }, ...rest]
}
