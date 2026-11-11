// The page's theme color (Page.ThemeColor in Go): the content of the first
// <meta name="theme-color"> in the document's head whose media is absent,
// empty or matches, and whose content is a color, as browsers pick it. The
// engine parses the color, so every CSS color works, and Go hears the color
// as #rrggbbaa, or "" for none, whenever it changes.

/** A meta element, as far as picking a theme color goes. */
export interface Meta {
  getAttribute(name: string): string | null;
}

/** Picks the theme color of the metas named theme-color, in tree order: the
 * first whose media matches and whose content is a color, as parse returns it,
 * else null. */
export function pick(
  metas: Iterable<Meta>,
  matches: (media: string) => boolean,
  parse: (value: string) => string | null,
): string | null {
  for (const m of metas) {
    const media = m.getAttribute("media")?.trim();
    if (media && !matches(media)) continue;
    const c = parse(m.getAttribute("content") ?? "");
    if (c !== null) return c;
  }
  return null;
}

const hex2 = (n: number) => Math.round(Math.min(Math.max(n, 0), 255)).toString(16).padStart(2, "0");

/** Returns a canvas's fillStyle as #rrggbbaa: engines serialize sRGB colors as
 * #rrggbb, or rgba(r, g, b, a) when they are translucent. Other color spaces
 * (oklch(), color(display-p3 …)) give null. */
export function fillStyleHex(s: string): string | null {
  if (/^#[0-9a-f]{6}$/i.test(s)) return s.toLowerCase() + "ff";
  const m = /^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)(?:[\s,/]+([\d.]+))?\s*\)$/i.exec(s);
  if (!m) return null;
  return "#" + hex2(+m[1]!) + hex2(+m[2]!) + hex2(+m[3]!) + hex2(m[4] === undefined ? 255 : +m[4] * 255);
}

/** Tells Go of the theme color when it changes. Before the DOM is ready it
 * tells only of a color, so that going from a page to another of the same
 * color tells nothing; once it is ready, of none too. */
export function reporter(post: (color: string) => void) {
  let last: string | null | undefined; // undefined: told nothing yet
  let ready = false;
  const report = (c: string | null) => {
    if (c === last || (c === null && last === undefined && !ready)) return;
    last = c;
    post(c ?? "");
  };
  return {
    report,
    /** The DOM is ready: the color is known, none included. */
    ready(c: string | null) {
      ready = true;
      report(c);
    },
  };
}

/** Watches the theme color of the document: its head's metas with a
 * MutationObserver, and their media with matchMedia, so that a change of the
 * system's appearance picks the meta for it. Returns the function to call
 * when the DOM is ready, which tells Go the color it found. */
export function watchThemeColor(post: (color: string) => void): () => void {
  const r = reporter(post);
  const queries = new Map<string, MediaQueryList>();
  const matches = (media: string) => {
    let q = queries.get(media);
    if (!q) {
      q = matchMedia(media);
      if (q.addEventListener) q.addEventListener("change", update);
      else q.addListener(update); // WebKit before 14
      queries.set(media, q);
    }
    return q.matches;
  };
  const current = () =>
    document.head ? pick(document.head.querySelectorAll('meta[name="theme-color" i]'), matches, parseColor) : null;
  function update() {
    r.report(current());
  }

  // The head only, rather than the whole document, so that pages changing
  // their body pay nothing; the document and its root element for a head
  // that comes, or is replaced, later.
  const observer = new MutationObserver((records) => {
    let changed = false;
    for (const rec of records) {
      if (rec.target === document || rec.target === document.documentElement) {
        observe();
        changed = true;
      } else if (rec.type === "attributes") {
        changed ||= isMeta(rec.target);
      } else {
        changed ||= hasMeta(rec.addedNodes) || hasMeta(rec.removedNodes);
      }
    }
    if (changed) update();
  });
  let head: HTMLHeadElement | null = null;
  let root: Element | null = null;
  function observe() {
    if (document.documentElement !== root && (root = document.documentElement)) {
      observer.observe(root, { childList: true });
    }
    if (document.head !== head && (head = document.head)) {
      observer.observe(head, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ["name", "content", "media"],
      });
    }
  }
  observer.observe(document, { childList: true });
  observe();
  update();
  return () => r.ready(current());
}

const isMeta = (n: Node) => (n as Element).localName === "meta";

function hasMeta(nodes: NodeList): boolean {
  for (let i = 0; i < nodes.length; i++) {
    const n = nodes[i]!;
    if (n.nodeType === 1 && (isMeta(n) || (n as Element).querySelector("meta"))) return true;
  }
  return false;
}

let ctx: CanvasRenderingContext2D | null | undefined;

/** Parses a CSS color as the engine does, with a canvas: its fillStyle keeps
 * its value when given something else than a color. Colors the engine does not
 * serialize as sRGB are drawn and read back. */
function parseColor(value: string): string | null {
  value = value.trim();
  if (!value) return null;
  if (ctx === undefined) {
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = 1;
    ctx = canvas.getContext("2d", { willReadFrequently: true });
  }
  if (!ctx) return null;
  ctx.fillStyle = "#000";
  ctx.fillStyle = value;
  if (ctx.fillStyle === "#000000") {
    ctx.fillStyle = "#fff";
    ctx.fillStyle = value;
    if (ctx.fillStyle === "#ffffff") return null; // not a color
  }
  const hex = fillStyleHex(String(ctx.fillStyle));
  if (hex) return hex;
  ctx.clearRect(0, 0, 1, 1);
  ctx.fillRect(0, 0, 1, 1);
  const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data;
  return "#" + hex2(r!) + hex2(g!) + hex2(b!) + hex2(a!);
}
