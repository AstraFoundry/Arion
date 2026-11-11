// What a context menu opens on, for Page.OnContextMenu in Go. The page's
// contextmenu event tells it just before the web view asks Go about its
// menu, which WebKit on macOS tells nothing of what was clicked, nor
// WebKitGTK of the selection.

/** The record Go reads: the link and its text, the image, the video or
 * audio, the selection, and whether the element is editable. Empty parts
 * are left out. */
export interface ContextTarget {
  l?: string;
  lt?: string;
  i?: string;
  m?: string;
  s?: string;
  e?: true;
}

/** The parts of an element a context menu's target is read from: an
 * Element of the DOM, or what a test makes. */
export interface TargetElement {
  closest(selector: string): TargetElement | null;
  /** A link's: a string, or an SVG link's animated string. */
  href?: string | { baseVal: string };
  /** An image's or a media element's. */
  currentSrc?: string;
  src?: string;
  innerText?: string;
  textContent?: string | null;
  isContentEditable?: boolean;
  /** A text field's. */
  type?: string;
  readOnly?: boolean;
  disabled?: boolean;
  value?: string;
  selectionStart?: number | null;
  selectionEnd?: number | null;
}

// Inputs that take no text, which a context menu offers no editing for.
const NOT_TEXT = new Set(["button", "checkbox", "color", "file", "hidden", "image", "radio", "range", "reset", "submit"]);

/** Describes el, the element a context menu opens on, with the text
 * selected in the document. */
export function contextTarget(el: TargetElement | null, selection: string): ContextTarget {
  const t: ContextTarget = {};
  const field = el?.closest("input,textarea") ?? null;
  const text = field && !(field.type && NOT_TEXT.has(field.type.toLowerCase()));
  if (text && !field.readOnly && !field.disabled) t.e = true;
  else if (el?.isContentEditable) t.e = true;
  // A text field's selection is not the document's. A password's is
  // nobody's business.
  if (text && field.type?.toLowerCase() !== "password") {
    const start = field.selectionStart ?? 0;
    const end = field.selectionEnd ?? 0;
    if (end > start) selection = (field.value ?? "").slice(start, end);
  } else if (text) {
    selection = "";
  }
  if (selection) t.s = selection;
  if (!el) return t;
  const link = el.closest("a[href],area[href]");
  if (link) {
    const href = typeof link.href === "string" ? link.href : (link.href?.baseVal ?? "");
    if (href) t.l = href;
    const label = (link.innerText ?? link.textContent ?? "").trim();
    if (label) t.lt = label;
  }
  const img = el.closest("img");
  if (img) {
    const src = img.currentSrc || img.src;
    if (src) t.i = src;
  }
  const media = el.closest("video,audio");
  if (media) {
    const src = media.currentSrc || media.src;
    if (src) t.m = src;
  }
  return t;
}
