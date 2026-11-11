import { describe, expect, test } from "bun:test";
import { type Direction, forward, type KeyPress, scrollDirection } from "./keys";

/** A key press: key, with its key code, and the modifiers in mods. */
const press = (key: string, mods: Partial<KeyPress> = {}): KeyPress => ({
  key,
  keyCode: key.length === 1 ? key.toUpperCase().charCodeAt(0) : 0,
  ctrl: false,
  alt: false,
  shift: false,
  meta: false,
  altGraph: false,
  composing: false,
  ...mods,
});

const never = () => false;
const always = () => true;

describe("forward", () => {
  test("gives the native UI letters and chords the page left", () => {
    expect(forward(press("j"), "other", never, false)).toBe(true);
    expect(forward(press("k", { ctrl: true }), "other", never, false)).toBe(true);
    expect(forward(press("k", { meta: true }), "other", never, false)).toBe(true);
    expect(forward(press("Enter"), "other", never, false)).toBe(true);
    expect(forward(press("F5"), "text", never, false)).toBe(true);
  });

  test("keeps Tab, modifiers on their own and the input method's keys", () => {
    for (const key of ["Tab", "Shift", "Control", "Alt", "AltGraph", "Meta", "CapsLock", "Dead", "Process"]) {
      expect(forward(press(key), "other", never, false)).toBe(false);
    }
    expect(forward(press("j", { composing: true }), "other", never, false)).toBe(false);
  });

  test("keeps what AltGr types, outside text fields too", () => {
    // AltGr+Q types @ on a German layout; Chromium tells neither Ctrl nor
    // Alt with it, but the key code of Q.
    const at = press("@", { keyCode: 81, altGraph: true });
    expect(forward(at, "other", never, false)).toBe(false);
    expect(forward(at, "text", never, false)).toBe(false);
    expect(forward(press("q", { ctrl: true, alt: true, altGraph: true }), "other", never, false)).toBe(false);
  });

  test("keeps what types or edits in a text field, not other chords", () => {
    for (const k of [press("j"), press("J", { shift: true }), press("Backspace"), press("ArrowLeft"), press(" ")]) {
      expect(forward(k, "text", never, false)).toBe(false);
    }
    for (const key of ["ArrowLeft", "Backspace", "Delete", "Home"]) {
      expect(forward(press(key, { ctrl: true }), "text", never, false)).toBe(false);
    }
    // AltGr, which Windows tells as Ctrl+Alt, types.
    expect(forward(press("q", { ctrl: true, alt: true }), "text", never, false)).toBe(false);
    expect(forward(press("k", { ctrl: true }), "text", never, false)).toBe(true);
    expect(forward(press("Escape"), "text", never, false)).toBe(true);
  });

  test("keeps the keys of edit commands everywhere", () => {
    for (const key of ["a", "c", "v", "x", "y", "z"]) {
      expect(forward(press(key, { ctrl: true }), "other", never, false)).toBe(false);
    }
    expect(forward(press("z", { ctrl: true, shift: true }), "other", never, false)).toBe(false);
  });

  test("keeps what activates or changes a control", () => {
    expect(forward(press("Enter"), "control", never, false)).toBe(false);
    expect(forward(press(" "), "control", never, false)).toBe(false);
    expect(forward(press("Enter", { ctrl: true }), "control", never, false)).toBe(true);
    // The arrows scroll from a link or a button, and change a slider.
    expect(forward(press("ArrowDown"), "control", never, false)).toBe(true);
    expect(forward(press("ArrowDown"), "adjustable", never, false)).toBe(false);
    expect(forward(press("j"), "adjustable", never, false)).toBe(true);
  });

  test("keeps the keys that scroll the page, while it can scroll that way", () => {
    const only = (d: Direction) => (got: Direction) => got === d;
    expect(forward(press("ArrowDown"), "other", always, false)).toBe(false);
    expect(forward(press("ArrowDown"), "other", never, false)).toBe(true);
    expect(forward(press(" "), "other", only("down"), false)).toBe(false);
    expect(forward(press(" ", { shift: true }), "other", only("down"), false)).toBe(true);
    expect(forward(press("Home", { ctrl: true }), "other", only("up"), false)).toBe(false);
  });

  test("keeps Escape while the page shows a dialog, a popover or full screen", () => {
    expect(forward(press("Escape"), "other", never, true)).toBe(false);
    expect(forward(press("Escape"), "other", never, false)).toBe(true);
  });
});

describe("scrollDirection", () => {
  test("tells the way a key scrolls", () => {
    expect(scrollDirection(press("PageDown"))).toBe("down");
    expect(scrollDirection(press("ArrowLeft"))).toBe("left");
    expect(scrollDirection(press(" ", { shift: true }))).toBe("up");
    expect(scrollDirection(press("End", { ctrl: true }))).toBe("down");
    expect(scrollDirection(press("ArrowDown", { ctrl: true }))).toBe(null);
    expect(scrollDirection(press("ArrowDown", { alt: true }))).toBe(null);
    expect(scrollDirection(press("j"))).toBe(null);
  });
});
