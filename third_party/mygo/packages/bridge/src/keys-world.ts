// Entry point of the script MyGo's WebView2 backend runs in an isolated
// world of every document of a web view in a window of native UI, top
// frame and frames alike (DevTools: Page.addScriptToEvaluateOnNewDocument
// with a worldName). It sees the page's keydown events, and after the
// page's handlers ran, tells Go of those the page left unhandled (keys.ts)
// through the binding Runtime.addBinding gives that world alone. The
// page's scripts reach neither the world nor the binding, and their
// synthetic events are not trusted: they cannot make up a key.
import { type Direction, forward, type Target } from "./keys";

// The binding's name, as internal/windows names it.
const BINDING = "mygoUnhandledKey";

// Inputs that take no text.
const NOT_TEXT = new Set(["button", "checkbox", "color", "file", "hidden", "image", "radio", "range", "reset", "submit"]);

// Controls that Enter and Space act on, and those the arrows change too.
const CONTROLS = 'a[href],area[href],button,summary,input,[role="button"],[role="link"],[role="checkbox"],[role="tab"]';
const ADJUSTABLE = 'input[type="range"],input[type="radio"],audio[controls],video[controls],[role="slider"],[role="radiogroup"]';

(() => {
  const w = window as any;
  if (w[BINDING + "Installed"]) return;
  w[BINDING + "Installed"] = true;

  // Where the user last pressed, where keys scroll from while nothing has
  // the focus, as in Chrome.
  let pressed: Element | null = null;
  addEventListener("pointerdown", (e) => (pressed = element(e.composedPath()[0])), true);

  addEventListener(
    "keydown",
    (e: KeyboardEvent) => {
      if (!e.isTrusted) return;
      // What has the focus, inside open shadow roots.
      let el = element(e.composedPath()[0]);
      for (let a = el?.shadowRoot?.activeElement; a; a = a.shadowRoot?.activeElement) el = a;
      if (el === document.body || el === document.documentElement) el = null;
      const press = {
        key: e.key,
        keyCode: e.keyCode,
        ctrl: e.ctrlKey,
        alt: e.altKey,
        shift: e.shiftKey,
        meta: e.metaKey,
        altGraph: e.getModifierState("AltGraph"),
        composing: e.isComposing || e.keyCode === 229,
      };
      // Decided as the key comes, before the page scrolls with it.
      const from = el ?? pressed;
      if (!forward(press, targetOf(el), (d) => scrolls(from, d), modal())) return;
      const mods = (e.shiftKey ? 1 : 0) | (e.ctrlKey ? 2 : 0) | (e.altKey ? 4 : 0) | (e.metaKey ? 8 : 0);
      // The page's handlers run in any order: look once all did.
      setTimeout(() => {
        if (e.defaultPrevented) return;
        try {
          w[BINDING](JSON.stringify({ k: e.keyCode, m: mods, r: e.repeat }));
        } catch {
          // No binding: the document is going.
        }
      });
    },
    true,
  );

  function element(n: EventTarget | null | undefined): Element | null {
    return n instanceof Element ? n : null;
  }

  function targetOf(el: Element | null): Target {
    if (!el) return "other";
    if ((el as HTMLElement).isContentEditable || el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) return "text";
    if (el instanceof HTMLInputElement && !NOT_TEXT.has(el.type)) return "text";
    return el.closest(ADJUSTABLE) ? "adjustable" : el.closest(CONTROLS) ? "control" : "other";
  }

  // Whether the element or one around it, up to the page, scrolls in the
  // direction, as the key would.
  function scrolls(from: Element | null, d: Direction): boolean {
    const root = document.scrollingElement;
    for (let n = from; n && n !== root && n !== document.body; ) {
      const style = getComputedStyle(n);
      const overflow = d === "up" || d === "down" ? style.overflowY : style.overflowX;
      if ((overflow === "auto" || overflow === "scroll" || overflow === "overlay") && room(n, d)) return true;
      n = n.parentElement ?? ((n.getRootNode() as ShadowRoot).host || null);
    }
    if (!root || !room(root, d)) return false;
    const hidden = (el: Element | null) => !!el && getComputedStyle(el)[d === "up" || d === "down" ? "overflowY" : "overflowX"] === "hidden";
    return !hidden(document.documentElement) && !hidden(document.body);
  }

  function room(n: Element, d: Direction): boolean {
    switch (d) {
      case "up":
        return n.scrollTop > 0;
      case "down":
        return n.scrollTop + n.clientHeight < n.scrollHeight - 1;
      default:
        // Either way, whatever the direction of the text.
        return n.scrollWidth > n.clientWidth + 1;
    }
  }

  // Whether Escape closes something of the page's own: a modal dialog, full
  // screen, or a popover.
  function modal(): boolean {
    try {
      return !!document.querySelector(":modal, :popover-open") || !!document.fullscreenElement;
    } catch {
      return !!document.fullscreenElement;
    }
  }
})();
