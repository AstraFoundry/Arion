import { describe, expect, test } from "bun:test";
import { contextTarget, type TargetElement } from "./context-menu";

/** An element matching the selectors of its tags, inside parent. */
const el = (tags: string[], props: Partial<TargetElement> = {}, parent: TargetElement | null = null): TargetElement => {
  const self: TargetElement = {
    ...props,
    closest(selector) {
      const matches = selector.split(",").some((s) => tags.includes(s.replace(/\[href\]$/, "")) && (!s.endsWith("[href]") || !!props.href));
      return matches ? self : (parent?.closest(selector) ?? null);
    },
  };
  return self;
};

describe("contextTarget", () => {
  test("tells a link and its text, from inside it", () => {
    const a = el(["a"], { href: "https://example.com/", innerText: "  Example  " });
    expect(contextTarget(el(["span"], {}, a), "")).toEqual({ l: "https://example.com/", lt: "Example" });
    expect(contextTarget(el(["a"], { textContent: "No href" }), "")).toEqual({});
  });

  test("tells an SVG link's href", () => {
    expect(contextTarget(el(["a"], { href: { baseVal: "/doc" }, textContent: "Doc" }), "")).toEqual({ l: "/doc", lt: "Doc" });
  });

  test("tells an image in a link", () => {
    const a = el(["a"], { href: "https://example.com/" });
    const img = el(["img"], { currentSrc: "", src: "https://example.com/cat.png" }, a);
    expect(contextTarget(img, "")).toEqual({ l: "https://example.com/", i: "https://example.com/cat.png" });
  });

  test("tells a video's source", () => {
    expect(contextTarget(el(["video"], { currentSrc: "https://example.com/a.mp4" }), "")).toEqual({ m: "https://example.com/a.mp4" });
  });

  test("tells the document's selection", () => {
    expect(contextTarget(el(["p"]), "Hello")).toEqual({ s: "Hello" });
    expect(contextTarget(null, "Hello")).toEqual({ s: "Hello" });
  });

  test("tells a text field's own selection, and that it is editable", () => {
    const input = el(["input"], { type: "text", value: "Hello world", selectionStart: 6, selectionEnd: 11 });
    expect(contextTarget(input, "elsewhere")).toEqual({ e: true, s: "world" });
    const area = el(["textarea"], { value: "abc", selectionStart: 1, selectionEnd: 1 });
    expect(contextTarget(area, "elsewhere")).toEqual({ e: true, s: "elsewhere" });
  });

  test("a read-only or disabled field, or a button, is not editable", () => {
    expect(contextTarget(el(["input"], { type: "text", readOnly: true }), "")).toEqual({});
    expect(contextTarget(el(["textarea"], { disabled: true }), "")).toEqual({});
    expect(contextTarget(el(["input"], { type: "checkbox" }), "x")).toEqual({ s: "x" });
  });

  test("never tells a password", () => {
    const pw = el(["input"], { type: "password", value: "hunter2", selectionStart: 0, selectionEnd: 7 });
    expect(contextTarget(pw, "")).toEqual({ e: true });
  });

  test("editable content", () => {
    expect(contextTarget(el(["div"], { isContentEditable: true }), "")).toEqual({ e: true });
  });
});
