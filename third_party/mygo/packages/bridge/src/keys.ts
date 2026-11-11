// Keys a page leaves unhandled, which the native UI around a web view gets
// as if the web view's element had the focus (platform.WebViewKey). WebKit
// tells of them itself, on macOS and Linux; WebView2 does not, so a script
// in an isolated world of the page (keys-world.ts) decides with these
// rules, which follow what the engine does with a key the page did not
// cancel: typing and editing in a text field, activating a control,
// scrolling, closing a dialog.

/** What forward reads of a keydown event. */
export interface KeyPress {
  key: string;
  keyCode: number;
  ctrl: boolean;
  alt: boolean;
  shift: boolean;
  meta: boolean;
  /** AltGr is held (getModifierState("AltGraph")), which types the
   * layout's third characters, as @ on a German keyboard: Chromium may
   * tell neither Ctrl nor Alt with it. */
  altGraph: boolean;
  /** An input method composes with it (isComposing, or keyCode 229). */
  composing: boolean;
}

/** What the key went to: a text field, textarea, select or editable
 * content, which types and edits with keys; a control that Enter and Space
 * act on (a button, a link, a checkbox); one that the arrows, Home, End and
 * Page Up and Down change too (a slider, radio buttons, a video with
 * controls); or anything else. */
export type Target = "text" | "control" | "adjustable" | "other";

/** The way a key scrolls a page. */
export type Direction = "up" | "down" | "left" | "right";

// Keys pressed on their own, which no native UI takes as a key.
const MODIFIERS = new Set([
  "Shift",
  "Control",
  "Alt",
  "AltGraph",
  "Meta",
  "OS",
  "Super",
  "Hyper",
  "Fn",
  "FnLock",
  "CapsLock",
  "NumLock",
  "ScrollLock",
  "Symbol",
  "SymbolLock",
]);

// Keys that move the caret or the selection in a text field, or scroll a
// page.
const MOVES = new Set(["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Home", "End", "PageUp", "PageDown"]);

// The edit commands' keys with Ctrl: select all, copy, cut, paste, undo and
// redo, by key code as the engine takes them, whatever the layout types.
const EDIT_CODES = new Set([65, 67, 86, 88, 89, 90]);

/** The way the key scrolls the page, if it does: the arrows, Page Up and
 * Down, Space (Shift+Space up), and Home and End, with Ctrl too. */
export function scrollDirection(k: KeyPress): Direction | null {
  if (k.alt || k.meta) return null;
  if (k.ctrl) return k.key === "Home" ? "up" : k.key === "End" ? "down" : null;
  switch (k.key) {
    case "ArrowUp":
    case "PageUp":
    case "Home":
      return "up";
    case "ArrowDown":
    case "PageDown":
    case "End":
      return "down";
    case "ArrowLeft":
      return "left";
    case "ArrowRight":
      return "right";
    case " ":
      return k.shift ? "up" : "down";
  }
  return null;
}

/**
 * forward reports whether a key the page did not cancel goes to the native
 * UI. It does not when it is Tab, which moves the focus, a modifier on its
 * own, a key of the input method's, one typed with AltGr, one that types
 * or edits in a text field, activates a control or changes it, copies or
 * selects (Ctrl+C, Ctrl+A), scrolls something that can scroll that way
 * (scrolls), or Escape while a modal dialog, a popover or full screen is
 * open (modal), which closes it.
 */
export function forward(k: KeyPress, target: Target, scrolls: (d: Direction) => boolean, modal: boolean): boolean {
  if (k.composing || k.altGraph || k.key === "Tab" || k.key === "Dead" || k.key === "Process" || MODIFIERS.has(k.key)) return false;
  if (k.key === "Escape") return !modal;
  if (/^F\d+$/.test(k.key)) return true;
  const chord = (k.ctrl || k.meta) && !(k.ctrl && k.alt); // Ctrl+Alt is AltGr, which types
  if (chord && EDIT_CODES.has(k.keyCode)) return false;
  if (target === "text") {
    return chord && !MOVES.has(k.key) && k.key !== "Backspace" && k.key !== "Delete" && k.key !== "Insert";
  }
  if (target !== "other" && !chord && !k.alt && (k.key === "Enter" || k.key === " ")) return false;
  if (target === "adjustable" && !chord && !k.alt && MOVES.has(k.key)) return false;
  const d = scrollDirection(k);
  return !d || !scrolls(d);
}
