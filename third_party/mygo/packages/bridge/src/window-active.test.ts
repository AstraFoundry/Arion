import { expect, test } from "bun:test";
import { createRuntime } from "./runtime";
import { INACTIVE_ATTRIBUTE, markWindowActive } from "./window-active";

/** An element that keeps its attributes as toggleAttribute sets them. */
function element() {
  const attrs = new Set<string>();
  return {
    attrs,
    toggleAttribute(name: string, force?: boolean) {
      const on = force ?? !attrs.has(name);
      if (on) attrs.add(name);
      else attrs.delete(name);
      return on;
    },
  };
}

test("a page starts active, and its <html> follows the event", () => {
  const { runtime, internal } = createRuntime(
    { platform: "darwin", windowId: 1, version: "0.0.0-test", secret: "s", locale: "en", dir: "ltr" },
    () => {},
  );
  const root = element();
  const seen: boolean[] = [];
  runtime.on<boolean>("mygo:window-active", (active) => {
    seen.push(runtime.windowActive);
    markWindowActive(root, active);
  });
  expect(runtime.windowActive).toBe(true);

  internal.receive({ t: "event", n: "mygo:window-active", p: false });
  expect(runtime.windowActive).toBe(false);
  expect(root.attrs.has(INACTIVE_ATTRIBUTE)).toBe(true);
  // Told again, as a page loads in an inactive window: no listener hears it.
  internal.receive({ t: "event", n: "mygo:window-active", p: false });
  // A payload that is no state changes nothing.
  internal.receive({ t: "event", n: "mygo:window-active", p: "yes" });
  expect(runtime.windowActive).toBe(false);

  internal.receive({ t: "event", n: "mygo:window-active", p: true });
  expect(runtime.windowActive).toBe(true);
  expect(root.attrs.has(INACTIVE_ATTRIBUTE)).toBe(false);
  // The listeners read the new state.
  expect(seen).toEqual([false, true]);
});

test("a page without its <html> yet is left alone", () => {
  expect(() => markWindowActive(null, false)).not.toThrow();
});
