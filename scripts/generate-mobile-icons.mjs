#!/usr/bin/env node
// Writes the mobile apps' icons from vendored SVGs (packages/mobile-icons):
// an iOS asset catalog in ArionKit and Android vector drawables in the design
// module. --check fails when a generated file differs from its source.
import { existsSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const manifest = JSON.parse(readFileSync(join(root, "packages/mobile-icons/icons.json"), "utf8"));
const sources = [join(root, "packages/mobile-icons/svg"), join(root, "apps/desktop/ui/shared/icons/svg")];
const iosDir = join(root, "apps/ios/ArionKit/Sources/ArionKit/Resources/Icons.xcassets");
const androidDir = join(root, "apps/android/design/src/main/res/drawable");
const check = process.argv.includes("--check");

// Pressed and active keys show a filled icon. It is derived from the official
// SVG, never drawn: "-fill" fills the closed outlines and drops open strokes;
// "-cut" keeps only the open strokes, which the app knocks out of the fill.
const derived = new Map();
for (const name of manifest.filled ?? []) {
  const svg = source(name);
  const elements = [...svg.matchAll(/<path\b[^>]*\/>/g)].map((m) => m[0]);
  const closed = (el) => /[zZ]\s*"/.test(el);
  const frame = (body) => svg.replace(/<path\b[\s\S]*<\/svg>/, `${body}\n</svg>`);
  derived.set(`${name}-fill`, frame(elements.filter(closed).map((el) => el.replace("<path", '<path fill="currentColor"')).join("\n  ")));
  const open = elements.filter((el) => !closed(el));
  if (open.length) derived.set(`${name}-cut`, frame(open.join("\n  ")));
}

function source(name) {
  if (derived?.has(name)) return derived.get(name);
  for (const dir of sources) {
    const file = join(dir, `${name}.svg`);
    if (existsSync(file)) return readFileSync(file, "utf8");
  }
  throw new Error(`icon ${name} is in neither ${sources.join(" nor ")}`);
}

const attrs = (text) => Object.fromEntries([...text.matchAll(/([\w:-]+)="([^"]*)"/g)].map((m) => [m[1], m[2]]));
const n = (v) => Number.parseFloat(v ?? "0");

/**
 * Path data with every number separated, so parsers that do not read the
 * compact arc flags of minified SVGs ("a1 1 0 01-2 0") see the same path.
 */
function spaced(d) {
  const tokens = d.match(/[a-zA-Z]|-?(?:\d+\.?\d*|\.\d+)(?:e-?\d+)?/g) ?? [];
  const out = [];
  let command = "";
  let index = 0;
  for (let i = 0; i < tokens.length; i++) {
    const t = tokens[i];
    if (/[a-zA-Z]/.test(t)) {
      command = t;
      index = 0;
      out.push(t);
      continue;
    }
    // Arc flags (the 4th and 5th arguments) are single digits that may run together.
    if (/[aA]/.test(command) && (index % 7 === 3 || index % 7 === 4) && /^[01]./.test(t)) {
      const rest = t;
      out.push(rest[0]);
      index++;
      tokens.splice(i + 1, 0, rest.slice(1));
      continue;
    }
    out.push(t);
    index++;
  }
  return out.join(" ");
}

/** One drawing element as path data; shapes become arcs and lines. */
function pathData(tag, a) {
  switch (tag) {
    case "path":
      return spaced(a.d);
    case "circle": {
      const [cx, cy, r] = [n(a.cx), n(a.cy), n(a.r)];
      return `M${cx - r},${cy}a${r},${r} 0 1,0 ${2 * r},0a${r},${r} 0 1,0 ${-2 * r},0`;
    }
    case "ellipse": {
      const [cx, cy, rx, ry] = [n(a.cx), n(a.cy), n(a.rx), n(a.ry)];
      return `M${cx - rx},${cy}a${rx},${ry} 0 1,0 ${2 * rx},0a${rx},${ry} 0 1,0 ${-2 * rx},0`;
    }
    case "rect": {
      const [x, y, w, h] = [n(a.x), n(a.y), n(a.width), n(a.height)];
      const rx = Math.min(n(a.rx ?? a.ry), w / 2);
      const ry = Math.min(n(a.ry ?? a.rx), h / 2);
      if (!rx && !ry) return `M${x},${y}h${w}v${h}h${-w}z`;
      return `M${x + rx},${y}h${w - 2 * rx}a${rx},${ry} 0 0 1 ${rx},${ry}v${h - 2 * ry}a${rx},${ry} 0 0 1 ${-rx},${ry}h${-(w - 2 * rx)}a${rx},${ry} 0 0 1 ${-rx},${-ry}v${-(h - 2 * ry)}a${rx},${ry} 0 0 1 ${rx},${-ry}z`;
    }
    case "line":
      return `M${n(a.x1)},${n(a.y1)}L${n(a.x2)},${n(a.y2)}`;
    case "polyline":
    case "polygon": {
      const points = a.points.trim().split(/[\s,]+/).map(Number);
      let d = `M${points[0]},${points[1]}`;
      for (let i = 2; i < points.length; i += 2) d += `L${points[i]},${points[i + 1]}`;
      return tag === "polygon" ? `${d}z` : d;
    }
    default:
      throw new Error(`unsupported SVG element <${tag}>`);
  }
}

/** The SVG as an Android vector drawable, keeping stroke or fill per element. */
function vector(name, svg) {
  const top = attrs(svg.match(/<svg([\s\S]*?)>/)[1]);
  const [, , vw, vh] = (top.viewBox ?? "0 0 24 24").split(/\s+/).map(Number);
  const stroked = top.stroke && top.stroke !== "none";
  const body = svg.replace(/<!--[\s\S]*?-->/g, "").replace(/<title>[\s\S]*?<\/title>/g, "");
  const paths = [...body.matchAll(/<(path|circle|ellipse|rect|line|polyline|polygon)\b([\s\S]*?)\/?>/g)].map(([, tag, raw]) => {
    const a = attrs(raw);
    const fill = a.fill ?? (stroked ? "none" : top.fill ?? "#000000");
    const color = (c) => (c === "currentColor" ? "#FF000000" : c);
    const lines = [`    <path`, `        android:pathData="${pathData(tag, a)}"`];
    if (fill !== "none") lines.push(`        android:fillColor="${color(fill)}"`);
    if ((a["fill-rule"] ?? top["fill-rule"]) === "evenodd") lines.push(`        android:fillType="evenOdd"`);
    if (stroked) {
      lines.push(
        `        android:strokeColor="${color(top.stroke)}"`,
        `        android:strokeWidth="${top["stroke-width"] ?? 2}"`,
        `        android:strokeLineCap="${top["stroke-linecap"] ?? "butt"}"`,
        `        android:strokeLineJoin="${top["stroke-linejoin"] ?? "miter"}"`,
      );
    }
    return `${lines.join("\n")} />`;
  });
  return `<?xml version="1.0" encoding="utf-8"?>
<!-- Generated by scripts/generate-mobile-icons.mjs from ${name}.svg. Do not edit. -->
<vector xmlns:android="http://schemas.android.com/apk/res/android"
    android:width="24dp"
    android:height="24dp"
    android:viewportWidth="${vw}"
    android:viewportHeight="${vh}">
${paths.join("\n")}
</vector>
`;
}

/** The SVG for Xcode: fixed size, no comments, current color as black for template rendering. */
function iosSVG(svg) {
  return svg
    .replace(/\sd="([^"]*)"/g, (_, d) => ` d="${spaced(d)}"`)
    .replace(/<!--[\s\S]*?-->\s*/g, "")
    .replace(/<title>[\s\S]*?<\/title>/g, "")
    .replace(/currentColor/g, "#000000")
    .replace(/\s(width|height)="1em"/g, ' $1="24"')
    .replace(/\sstyle="[^"]*"/g, "")
    .replace(/\sclass="[^"]*"/g, "");
}

const camel = (name) => name.replace(/-(\w)/g, (_, c) => c.toUpperCase());
const files = new Map();
files.set(join(iosDir, "Contents.json"), `${JSON.stringify({ info: { author: "xcode", version: 1 } }, null, 2)}\n`);
const all = [...manifest.icons.map((name) => [name, true]), ...[...derived.keys()].map((name) => [name, true]), ...manifest.original.map((name) => [name, false])];
for (const [name, template] of all) {
  const svg = source(name);
  const set = join(iosDir, `${camel(name)}.imageset`);
  files.set(join(set, `${name}.svg`), iosSVG(svg));
  const properties = { "preserves-vector-representation": true };
  if (template) properties["template-rendering-intent"] = "template";
  files.set(join(set, "Contents.json"), `${JSON.stringify({ images: [{ filename: `${name}.svg`, idiom: "universal" }], info: { author: "xcode", version: 1 }, properties }, null, 2)}\n`);
  files.set(join(androidDir, `icon_${name.replace(/-/g, "_")}.xml`), vector(name, svg));
}
const swift = join(root, "apps/ios/ArionKit/Sources/ArionKit/Design/Icon.swift");
files.set(swift, `// Generated by scripts/generate-mobile-icons.mjs. Do not edit.
import UIKit

/// The apps' icons, generated from the vendored Lucide, LobeHub and brand SVGs.
public enum Icon: String, CaseIterable, Sendable {
${all.map(([name]) => `    case ${camel(name)}`).join("\n")}

    /// Brand marks that keep their own colors.
    public var original: Bool {
        switch self {
${manifest.original.map((name) => `        case .${camel(name)}: true`).join("\n")}
        default: false
        }
    }

    /// The icon as a template image (brand marks keep their colors).
    public var image: UIImage {
        guard let image = UIImage(named: rawValue, in: .module, compatibleWith: nil) else {
            preconditionFailure("Icon \\(rawValue) is missing from Icons.xcassets")
        }
        return image
    }
}
`);

const pascal = (name) => name.replace(/(^|-)(\w)/g, (_, __, c) => c.toUpperCase());
files.set(join(root, "apps/android/design/src/main/kotlin/si/astrafoundry/arion/design/ArionIcon.kt"), `// Generated by scripts/generate-mobile-icons.mjs. Do not edit.
package si.astrafoundry.arion.design

/** The apps' icons, generated from the vendored Lucide, LobeHub and brand SVGs. */
enum class ArionIcon(val res: Int, val original: Boolean = false) {
${all.map(([name, template]) => `    ${pascal(name)}(R.drawable.icon_${name.replace(/-/g, "_")}${template ? "" : ", original = true"}),`).join("\n")}
}
`);

let drift = [];
const generatedAndroid = new Set([...files.keys()].filter((f) => f.startsWith(androidDir)));
if (check) {
  for (const [file, text] of files) if (!existsSync(file) || readFileSync(file, "utf8") !== text) drift.push(file);
  if (existsSync(androidDir)) {
    for (const f of readdirSync(androidDir)) {
      const file = join(androidDir, f);
      if (f.startsWith("icon_") && !generatedAndroid.has(file)) drift.push(file);
    }
  }
  if (drift.length) {
    console.error(`Mobile icons are out of date; run pnpm mobile:generate:\n${drift.map((f) => `  ${f}`).join("\n")}`);
    process.exit(1);
  }
} else {
  rmSync(iosDir, { recursive: true, force: true });
  if (existsSync(androidDir)) for (const f of readdirSync(androidDir)) if (f.startsWith("icon_")) rmSync(join(androidDir, f));
  for (const [file, text] of files) {
    mkdirSync(dirname(file), { recursive: true });
    writeFileSync(file, text);
  }
}
