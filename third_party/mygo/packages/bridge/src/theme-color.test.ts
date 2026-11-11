import { describe, expect, test } from "bun:test";
import { fillStyleHex, type Meta, pick, reporter } from "./theme-color";

const meta = (attrs: Record<string, string>): Meta => ({ getAttribute: (n) => attrs[n] ?? null });
// Colors named as the test likes; anything else is not a color.
const parse = (v: string) => ({ red: "#ff0000ff", blue: "#0000ffff" })[v.trim()] ?? null;

describe("pick", () => {
  const dark = (media: string) => media === "(prefers-color-scheme: dark)";

  test("takes the first meta whose media matches", () => {
    const metas = [
      meta({ content: "red", media: "(prefers-color-scheme: light)" }),
      meta({ content: "blue", media: " (prefers-color-scheme: dark) " }),
    ];
    expect(pick(metas, dark, parse)).toBe("#0000ffff");
    expect(pick(metas, (m) => m === "(prefers-color-scheme: light)", parse)).toBe("#ff0000ff");
  });

  test("an absent or empty media matches", () => {
    expect(pick([meta({ content: "red" })], () => false, parse)).toBe("#ff0000ff");
    expect(pick([meta({ content: "red", media: " " })], () => false, parse)).toBe("#ff0000ff");
  });

  test("skips metas that are not colors", () => {
    expect(pick([meta({ content: "nope" }), meta({}), meta({ content: " blue " })], dark, parse)).toBe("#0000ffff");
    expect(pick([meta({ content: "nope" })], dark, parse)).toBeNull();
    expect(pick([], dark, parse)).toBeNull();
  });
});

describe("fillStyleHex", () => {
  test("reads what engines serialize", () => {
    expect(fillStyleHex("#1E90FF")).toBe("#1e90ffff");
    expect(fillStyleHex("rgba(255, 0, 0, 0.5)")).toBe("#ff000080");
    expect(fillStyleHex("rgba(0, 0, 0, 0)")).toBe("#00000000");
    expect(fillStyleHex("rgb(1 2 3 / 1)")).toBe("#010203ff");
    expect(fillStyleHex("rgb(1, 2, 3)")).toBe("#010203ff");
  });

  test("leaves other color spaces to the canvas", () => {
    expect(fillStyleHex("oklch(0.7 0.1 200)")).toBeNull();
    expect(fillStyleHex("color(display-p3 1 0 0)")).toBeNull();
  });
});

describe("reporter", () => {
  test("tells changes only", () => {
    const sent: string[] = [];
    const r = reporter((c) => sent.push(c));
    r.report("#ff0000ff");
    r.report("#ff0000ff");
    r.ready("#ff0000ff");
    r.report("#0000ffff");
    r.report(null);
    r.report(null);
    expect(sent).toEqual(["#ff0000ff", "#0000ffff", ""]);
  });

  test("tells of no color once the DOM is ready", () => {
    const sent: string[] = [];
    const r = reporter((c) => sent.push(c));
    r.report(null); // at document start: the previous page's may be the same
    expect(sent).toEqual([]);
    r.ready(null);
    expect(sent).toEqual([""]);
  });

  test("tells of a color removed before the DOM is ready", () => {
    const sent: string[] = [];
    const r = reporter((c) => sent.push(c));
    r.report("#ff0000ff");
    r.report(null);
    r.ready(null);
    expect(sent).toEqual(["#ff0000ff", ""]);
  });
});
