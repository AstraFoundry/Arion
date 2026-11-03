import type { Element, ElementContent, Root } from "hast";
import { toString } from "hast-util-to-string";
import { SKIP, visit } from "unist-util-visit";
import type { ImageSize, MediaRef } from "@/shared/lib/server";

/**
 * Raw HTML in a post is trusted (posts are reviewed like code) but must stay
 * static: scripts, styles, event handlers and bare frames are rejected so
 * interactive content goes through `<embed-frame>`, which isolates it and
 * loads it lazily.
 */
const FORBIDDEN_ELEMENTS: Record<string, string> = {
  script: "put interactive HTML in a file and use <embed-frame>",
  style: "style posts through the prose styles, not inline <style>",
  iframe: "use <embed-frame>, which loads lazily and asks before contacting another site",
  object: "use <embed-frame>",
  embed: "use <embed-frame>",
  video: "use <video-clip>, which loads lazily",
};

export class PostContentError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PostContentError";
  }
}

export function rehypeStaticHtml() {
  return (tree: Root) => {
    visit(tree, "element", (node) => {
      const advice = FORBIDDEN_ELEMENTS[node.tagName];
      if (advice) throw new PostContentError(`<${node.tagName}> is not allowed in posts: ${advice}`);
      for (const name of Object.keys(node.properties)) {
        if (/^on[A-Z]/.test(name)) {
          throw new PostContentError(`<${node.tagName}> has an event handler (${name}); posts are static`);
        }
      }
    });
  };
}

const isBlank = (node: ElementContent) => node.type === "text" && node.value.trim() === "";

/** Embeds that render a figure; Markdown wraps one written on a single line in a paragraph. */
const BLOCK_EMBEDS = new Set(["dictation-pill", "embed-frame", "github-people", "video-clip", "voice-quote"]);

/**
 * A paragraph that holds only an image becomes a figure; the Markdown title
 * (`![alt](src "Caption")`) becomes its caption. A paragraph that holds only
 * a block embed is replaced by the embed, since a figure cannot sit in a `<p>`.
 */
export function rehypeFigures() {
  return (tree: Root) => {
    visit(tree, "element", (node, index, parent) => {
      if (node.tagName !== "p") return;
      const content = node.children.filter((child) => !isBlank(child));
      const [image] = content;
      if (content.length > 1) {
        const embed = content.find(
          (child) => child.type === "element" && BLOCK_EMBEDS.has(child.tagName),
        );
        if (embed?.type === "element") {
          throw new PostContentError(`<${embed.tagName}> must be on its own line, not inside a paragraph`);
        }
        return;
      }
      if (content.length !== 1 || image.type !== "element") return;
      if (BLOCK_EMBEDS.has(image.tagName) && parent && index !== undefined) {
        parent.children[index] = image;
        return SKIP;
      }
      if (image.tagName !== "img") return;

      const caption = image.properties.title;
      delete image.properties.title;
      node.tagName = "figure";
      node.properties = { className: ["post-wide"] };
      node.children = [image];
      if (typeof caption === "string" && caption.trim() !== "") {
        node.children.push({
          type: "element",
          tagName: "figcaption",
          properties: {},
          children: [{ type: "text", value: caption }],
        });
      }
      return SKIP;
    });
  };
}

/** `<embed-frame ratio>`: width/height, such as `16/9` or `4/3`. */
const RATIO = /^\d+(\.\d+)?\/\d+(\.\d+)?$/;
const PILL_STATES = ["listening", "inserted", "learned"];

function requireSize(node: Element, what: string) {
  const { width, height } = node.properties;
  if (!Number(width) || !Number(height)) {
    throw new PostContentError(`${what} needs width and height attributes so the page does not jump`);
  }
}

/** Element properties that name media files. */
const MEDIA_PROPERTIES: Record<string, readonly string[]> = {
  img: ["src"],
  "video-clip": ["src", "poster"],
  "embed-frame": ["poster"],
};

export interface MediaOptions {
  /** Resolves a `src` as written in the post (see `resolveMedia`). */
  resolve: (src: string) => MediaRef;
  /** Every reference resolved, by URL; filled while the tree is processed. */
  refs?: Map<string, MediaRef>;
}

/**
 * Rewrites media sources to the URLs the page uses: remote URLs stay,
 * `public/` paths stay, and project paths (relative to the post) become
 * `/blog-assets/…`. Remote images are requested without a referrer.
 */
export function rehypeResolveMedia(options: MediaOptions) {
  return (tree: Root) => {
    visit(tree, "element", (node) => {
      for (const name of MEDIA_PROPERTIES[node.tagName] ?? []) {
        const src = node.properties[name];
        if (src === undefined) continue;
        if (typeof src !== "string" || src === "") {
          throw new PostContentError(`<${node.tagName}> has an empty ${name}`);
        }
        let ref: MediaRef;
        try {
          ref = options.resolve(src);
        } catch (error) {
          throw new PostContentError(`<${node.tagName} ${name}>: ${(error as Error).message}`);
        }
        node.properties[name] = ref.url;
        options.refs?.set(ref.url, ref);
        if (node.tagName === "img" && !ref.file) node.properties.referrerPolicy = "no-referrer";
      }
    });
  };
}

/**
 * Sets the intrinsic size on every image and video so lazy media reserve
 * their space: measured from the file, or from a remote image fetched at
 * build time, unless the post states width and height itself.
 */
export function rehypeMediaSize(options: { measure: (url: string) => Promise<ImageSize> }) {
  return async (tree: Root) => {
    const pending: Promise<void>[] = [];
    const sizeFrom = (node: Element, url: unknown) => {
      const { width, height } = node.properties;
      if (width && height) return;
      if (width || height) {
        throw new PostContentError(`<${node.tagName} src="${String(node.properties.src)}"> states only one of width and height`);
      }
      if (typeof url !== "string") return;
      pending.push(
        options.measure(url).then(
          (size) => {
            Object.assign(node.properties, size);
          },
          (error: Error) => {
            throw new PostContentError(`Cannot size ${url}: ${error.message}`);
          },
        ),
      );
    };

    visit(tree, "element", (node) => {
      const { properties } = node;
      if (node.tagName === "img") {
        if (typeof properties.alt !== "string") {
          throw new PostContentError(`The image ${String(properties.src)} needs alt text`);
        }
        sizeFrom(node, properties.src);
      } else if (node.tagName === "video-clip") {
        sizeFrom(node, properties.poster);
        if (!properties.poster) requireSize(node, `<video-clip src="${String(properties.src)}">`);
      } else if (node.tagName === "embed-frame") {
        if (typeof properties.title !== "string" || properties.title.trim() === "") {
          throw new PostContentError(`<embed-frame src="${String(properties.src)}"> needs a title`);
        }
        const { ratio, height } = properties;
        const validRatio = typeof ratio === "string" && RATIO.test(ratio);
        if (validRatio === Boolean(Number(height))) {
          throw new PostContentError(
            `<embed-frame src="${String(properties.src)}"> needs either a ratio (such as 16/9) or a height`,
          );
        }
      } else if (node.tagName === "dictation-pill") {
        const { state, label } = properties;
        if (!PILL_STATES.includes(state as string)) {
          throw new PostContentError(`<dictation-pill> state must be one of ${PILL_STATES.join(", ")}`);
        }
        if (typeof label !== "string" || label.trim() === "") {
          throw new PostContentError("<dictation-pill> needs a label describing the illustration");
        }
        const field = state === "listening" ? "draft" : state === "learned" ? "term" : undefined;
        if (field && typeof properties[field] !== "string") {
          throw new PostContentError(`<dictation-pill state="${String(state)}"> needs ${field}`);
        }
      }
    });
    await Promise.all(pending);
  };
}

/** GitHub profile fields an embed shows; the post entity's `PostAuthor` matches it. */
export interface GitHubPerson {
  login: string;
  name: string;
  profileUrl: string;
  avatar: string;
}

/** Same rule as front matter `authors`: a GitHub login. */
const LOGIN = /^[a-z\d](?:[a-z\d]|-(?=[a-z\d])){0,38}$/i;

/** The logins an embed names: `<github-people logins="a, b">` or `<voice-quote login="a">`. */
function embedLogins(node: Element): string[] | undefined {
  if (node.tagName === "github-people") {
    const { logins, label } = node.properties;
    if (typeof label !== "string" || label.trim() === "") {
      throw new PostContentError("<github-people> needs a label naming the group");
    }
    const list = typeof logins === "string" ? logins.split(",").map((login) => login.trim()) : [];
    if (list.length === 0 || list.some((login) => !LOGIN.test(login))) {
      throw new PostContentError(`<github-people label="${label}"> needs logins: GitHub logins separated by commas`);
    }
    if (new Set(list.map((login) => login.toLowerCase())).size !== list.length) {
      throw new PostContentError(`<github-people label="${label}"> lists a login twice`);
    }
    return list;
  }
  if (node.tagName === "voice-quote") {
    const { login } = node.properties;
    if (typeof login !== "string" || !LOGIN.test(login)) {
      throw new PostContentError("<voice-quote> needs login, the speaker's GitHub login");
    }
    if (toString(node).trim() === "") throw new PostContentError(`<voice-quote login="${login}"> needs the quote as its content`);
    return [login];
  }
  return undefined;
}

/** Every GitHub login the post's people embeds name, validated. */
export function collectPeopleLogins(tree: Root): string[] {
  const logins: string[] = [];
  visit(tree, "element", (node) => {
    logins.push(...(embedLogins(node) ?? []));
  });
  return logins;
}

/**
 * Replaces the logins on `<github-people>` and `<voice-quote>` with the
 * profiles GitHub returns at build time (as JSON in `people`), so the page
 * shows names and avatars published with the site.
 */
export function rehypeGitHubPeople(options: { resolve: (logins: string[]) => Promise<GitHubPerson[]> }) {
  return async (tree: Root) => {
    const pending: Promise<void>[] = [];
    visit(tree, "element", (node) => {
      const logins = embedLogins(node);
      if (!logins) return;
      pending.push(
        options.resolve(logins).then((people) => {
          delete node.properties.logins;
          delete node.properties.login;
          node.properties.people = JSON.stringify(people);
        }),
      );
    });
    await Promise.all(pending);
  };
}

export interface PostHeading {
  id: string;
  text: string;
}

/** Level-two headings with their slug ids, for the table of contents. */
export function collectHeadings(tree: Root, skipId: string): PostHeading[] {
  const headings: PostHeading[] = [];
  visit(tree, "element", (node) => {
    const id = node.properties.id;
    if (node.tagName === "h2" && typeof id === "string" && id !== skipId) {
      headings.push({ id, text: toString(node) });
    }
  });
  return headings;
}
