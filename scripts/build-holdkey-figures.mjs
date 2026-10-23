// Builds the onboarding dictation key figure with the Anatomy kit
// (.agents/skills/anatomy): an isometric keyboard on a desk mat, wired to a
// desk microphone. The chosen dictation key is the one lit part; pressed, its
// cap sinks into the deck and the microphone's ring lights.
//
// MyGo draws SVG without CSS variables, attribute selectors, filters or
// text, so every color is written inline, the halo is stacked translucent
// outlines and keycap legends are stroked paths.
//
//   node scripts/build-holdkey-figures.mjs            write the SVGs
//   node scripts/build-holdkey-figures.mjs --preview  also a contact sheet in .workspace/
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import * as k from "../.agents/skills/anatomy/kit/iso-kit.mjs";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const OUT = join(ROOT, "apps/desktop/ui/shared/assets/holdkey");

const WIDTH = 300;
const HEIGHT = 300;
const STROKE = 0.75;

// Sunbeam neutrals arranged as the kit's tones. Paper is the pane's Card
// token, so the figure has no background of its own.
const PALETTES = {
  dark: {
    paper: "#221e1b",
    shades: ["#1c1916", "#201c19", "#25211d", "#2a2521"],
    top: "#2e2925",
    litTop: "#3d3418",
    litShades: ["#2a2412", "#2e2814", "#332c16", "#383018"],
    hi: "#dfdad3",
    mid: "#7d756e",
    lo: "#4d4640",
    faint: "#38322d",
    lit: "#ffcc00",
    live: "#eb6347",
    dot: "#efece6",
    halo: "#000000",
    haloOpacity: 0.16,
    hole: "#141210",
  },
  light: {
    paper: "#ffffff",
    shades: ["#e0dbd3", "#e8e4dd", "#f0ede8", "#f7f5f1"],
    top: "#ffffff",
    litTop: "#fff3c2",
    litShades: ["#f0dc8c", "#f4e39f", "#f8eab2", "#fbf0c4"],
    hi: "#38322e",
    mid: "#988f86",
    lo: "#c0b9af",
    faint: "#e3dfd8",
    lit: "#a87e00",
    live: "#e03f1f",
    dot: "#38322e",
    halo: "#726b65",
    haloOpacity: 0.07,
    hole: "#d6d0c7",
  },
};

// World units: U is one key pitch, about 18 mm.
const U = 18;
const GAP = 1.5;
const MARGIN = 7;
const CASE_H = 7;
const CAP_LIFT = 1.4;
const CAP_H = 2.4;
const TRAVEL = 1.2;
const ROW_D = [0.62, 1, 1, 1, 1, 1];

// Rows of [id, width in U]. Only the bottom row's modifiers carry legends;
// the arrow keys are half height.
const UPPER = [
  [["esc", 1.5], ...Array.from({ length: 12 }, (_, i) => [`f${i + 1}`, 1]), ["power", 1.5]],
  [["grave", 1], ...Array.from({ length: 12 }, (_, i) => [`n${i}`, 1]), ["delete", 2]],
  [["tab", 1.5], ...Array.from({ length: 12 }, (_, i) => [`q${i}`, 1]), ["backslash", 1.5]],
  [["caps", 1.75], ...Array.from({ length: 11 }, (_, i) => [`a${i}`, 1]), ["return", 2.25]],
  [["shift", 2.25], ...Array.from({ length: 10 }, (_, i) => [`z${i}`, 1]), ["rshift", 2.75]],
];
const LAYOUTS = {
  darwin: [
    ...UPPER,
    [["fn", 1, "fn"], ["lctrl", 1, "control"], ["lopt", 1, "option"], ["lcmd", 1.25, "command"], ["space", 5.5], ["right_command", 1.25, "command"], ["right_option", 1, "option"], ["arrows", 3]],
  ],
  windows: [
    ...UPPER,
    [["lctrl", 1.25, "Ctrl"], ["lwin", 1.25, "win"], ["lalt", 1.25, "Alt"], ["space", 5.25], ["right_alt", 1, "Alt"], ["menu", 1, "menu"], ["right_ctrl", 1, "Ctrl"], ["arrows", 3]],
  ],
};
const KEYS = { darwin: ["fn", "right_option", "right_command"], windows: ["right_alt", "right_ctrl"] };
// Homing bumps sit on F and J: the fourth and seventh letter of the home row.
const HOMING = new Set(["a3", "a6"]);

const ROWS_D = ROW_D.reduce((a, b) => a + b, 0) * U;
const CASE = { x: -MARGIN, y: -MARGIN, w: 15 * U + MARGIN * 2, d: ROWS_D + MARGIN * 2, r: 6 };
const MIC = { x: 12.6 * U, y: -2.7 * U };
const MAT = { x: CASE.x - 14, y: MIC.y - 30, w: CASE.w + 28, d: CASE.y + CASE.d + 14 - (MIC.y - 30), r: 5 };
const MAT_H = 1.6;
const PORT = { x: 10.4 * U, y: CASE.y, z: 3.6 };
const MIC_BASE_R = 21;
const MIC_TOP = 150;

const P = k.fitProjection(
  [...k.boxCorners(MAT, 0, MAT_H), [MIC.x, MIC.y, MIC_TOP + 4], [MIC.x + 10, MIC.y, MIC_TOP + 2]],
  WIDTH,
  HEIGHT,
  { pad: 6, azimuth: 32, elevation: 42 },
);
const at = (point) => k.iso(point, P);
const depth = (x, y, z = 0) => k.depthOf([x, y, z], P);
const num = (value) => (Math.round(value * 10) / 10).toString();

// Painting with a style sheet of classes: MyGo matches class selectors but
// not CSS variables or attribute selectors. Fills: p paper, s0–s3 side
// shades, t tops, l0–l3 and lt the lit part, h key openings, hl halo.
// Strokes: the tones hi, mi, lo, fa, the lit accent li and the live ring lv.
function styleOf(pal) {
  const rules = [
    [".p", `fill:${pal.paper}`],
    ...pal.shades.map((c, i) => [`.s${i}`, `fill:${c}`]),
    [".t", `fill:${pal.top}`],
    ...pal.litShades.map((c, i) => [`.l${i}`, `fill:${c}`]),
    [".lt", `fill:${pal.litTop}`],
    [".h", `fill:${pal.hole}`],
    [".hl", `fill:${pal.halo};fill-opacity:${pal.haloOpacity}`],
    [".d", `fill:${pal.dot}`],
    [".hi", `stroke:${pal.hi}`],
    [".mi", `stroke:${pal.mid}`],
    [".lo", `stroke:${pal.lo}`],
    [".fa", `stroke:${pal.faint}`],
    [".li", `stroke:${pal.lit}`],
    [".lv", `stroke:${pal.live}`],
    [".pa", `stroke:${pal.paper}`],
  ];
  return `<style>${rules.map(([sel, body]) => `${sel}{${body}}`).join("")}</style>`;
}

const TONES = { hi: "hi", mid: "mi", lo: "lo", faint: "fa", lit: "li", live: "lv", paper: "pa" };
const fill = (d, cls) => (d ? `<path class="${cls}" d="${d}"/>` : "");
const line = (d, tone, { width = STROKE, opacity } = {}) =>
  d ? `<path class="${TONES[tone]}"${width !== STROKE ? ` stroke-width="${width}"` : ""}${opacity ? ` stroke-opacity="${opacity}"` : ""} d="${d}"/>` : "";
function solid(paths, { tone = "hi", crease = "faint", lit = false, flat = false, inner = "" } = {}) {
  const parts = [fill(paths.fill, "p")];
  if (!flat) {
    paths.shades.forEach((d, shade) => parts.push(fill(d, `${lit ? "l" : "s"}${shade}`)));
    parts.push(fill(paths.top, lit ? "lt" : "t"));
  }
  if (crease !== "none") parts.push(line(paths.crease, crease), line(paths.bevel, crease));
  parts.push(inner, line(paths.outline, lit ? "lit" : tone));
  return parts.join("");
}
const dots = (points, { size = 0.5, opacity = 0.45 } = {}) =>
  `<g class="d" fill-opacity="${opacity}">${points.map(([x, y]) => `<circle cx="${num(x)}" cy="${num(y)}" r="${size}"/>`).join("")}</g>`;
// A soft contact shadow: the footprint grown in steps, each faint.
const halo = (plan, z) => Array.from({ length: 5 }, (_, i) => fill(k.planOutline(k.insetPlan(plan, -i * 2.2), z, P), "hl")).join("");

// Stroke glyphs in an em box, y up, as polylines; width is the advance.
const arc = (cx, cy, rx, ry, from, to, steps = 10) =>
  Array.from({ length: steps + 1 }, (_, i) => {
    const a = ((from + ((to - from) * i) / steps) * Math.PI) / 180;
    return [cx + rx * Math.cos(a), cy + ry * Math.sin(a)];
  });
const GLYPHS = {
  f: { w: 0.42, lines: [[[0.14, 0], [0.14, 0.78], ...arc(0.32, 0.78, 0.18, 0.2, 180, 90, 4).slice(1)], [[0.02, 0.62], [0.36, 0.62]]] },
  n: { w: 0.56, lines: [[[0.06, 0], [0.06, 0.62]], [[0.06, 0.4], ...arc(0.27, 0.4, 0.21, 0.22, 180, 0, 8), [0.48, 0]]] },
  C: { w: 0.7, lines: [arc(0.38, 0.5, 0.32, 0.5, 50, 310, 14)] },
  t: { w: 0.36, lines: [[[0.14, 0.86], [0.14, 0.16], ...arc(0.28, 0.16, 0.14, 0.16, 180, 270, 3).slice(1)], [[0.02, 0.62], [0.32, 0.62]]] },
  r: { w: 0.38, lines: [[[0.06, 0], [0.06, 0.62]], [[0.06, 0.36], ...arc(0.3, 0.36, 0.24, 0.26, 180, 90, 4).slice(1)]] },
  l: { w: 0.2, lines: [[[0.08, 0], [0.08, 1]]] },
  A: { w: 0.72, lines: [[[0, 0], [0.34, 1], [0.68, 0]], [[0.12, 0.34], [0.56, 0.34]]] },
};
const SYMBOLS = {
  // ⌘: a square with a loop at each corner.
  command: [
    [[0.3, 0.3], [0.7, 0.3], [0.7, 0.7], [0.3, 0.7], [0.3, 0.3]],
    ...[[0.3, 0.3], [0.7, 0.3], [0.7, 0.7], [0.3, 0.7]].map(([cx, cy]) => {
      const out = [cx < 0.5 ? 1 : -1, cy < 0.5 ? 1 : -1];
      const start = Math.atan2(0, out[0]) * (180 / Math.PI);
      const center = [cx - out[0] * 0.14, cy - out[1] * 0.14];
      const sweep = out[0] * out[1] > 0 ? -270 : 270;
      return arc(center[0], center[1], 0.14, 0.14, start, start + sweep, 12);
    }),
  ],
  // ⌥: a shifted rail and a short bar.
  option: [[[0.04, 0.74], [0.34, 0.74], [0.68, 0.26], [0.96, 0.26]], [[0.6, 0.74], [0.96, 0.74]]],
  // ⌃
  control: [[[0.22, 0.36], [0.5, 0.72], [0.78, 0.36]]],
  // The Windows logo: four panes.
  win: [
    [[0.22, 0.22], [0.47, 0.22], [0.47, 0.47], [0.22, 0.47], [0.22, 0.22]],
    [[0.53, 0.22], [0.78, 0.22], [0.78, 0.47], [0.53, 0.47], [0.53, 0.22]],
    [[0.22, 0.53], [0.47, 0.53], [0.47, 0.78], [0.22, 0.78], [0.22, 0.53]],
    [[0.53, 0.53], [0.78, 0.53], [0.78, 0.78], [0.53, 0.78], [0.53, 0.53]],
  ],
  // The menu key: a page of three lines.
  menu: [[[0.24, 0.16], [0.76, 0.16], [0.76, 0.84], [0.24, 0.84], [0.24, 0.16]], [[0.36, 0.64], [0.64, 0.64]], [[0.36, 0.5], [0.64, 0.5]], [[0.36, 0.36], [0.64, 0.36]]],
};

// A legend's polylines laid on a keycap top centred on (cx, cy): em units
// run along +x and up the cap toward the back (-y).
function legend(text, cx, cy, z, size) {
  let lines;
  let width;
  if (SYMBOLS[text]) {
    lines = SYMBOLS[text].map((pts) => pts.map(([x, y]) => [x - 0.5, y - 0.5]));
    width = 1;
  } else {
    lines = [];
    let pen = 0;
    for (const ch of text) {
      const g = GLYPHS[ch];
      lines.push(...g.lines.map((pts) => pts.map(([x, y]) => [x + pen, y])));
      pen += g.w + 0.08;
    }
    width = pen - 0.08;
    lines = lines.map((pts) => pts.map(([x, y]) => [x - width / 2, y - 0.4]));
  }
  return lines.map((pts) => k.pathOf(pts.map(([x, y]) => at([cx + x * size, cy - y * size, z])))).join("");
}

// The caps of a layout with their plans, in painting order.
function capsOf(platform) {
  const caps = [];
  let y = 0;
  LAYOUTS[platform].forEach((row, r) => {
    const d = ROW_D[r] * U;
    let x = 0;
    for (const [id, width, mark] of row) {
      const w = width * U;
      if (id === "arrows") {
        // ← ↑↓ →: the inverted T in half-height caps.
        const half = d / 2;
        const unit = w / 3;
        caps.push({ id: "left", plan: { x, y: y + half, w: unit, d: half }, mark: "left" });
        caps.push({ id: "up", plan: { x: x + unit, y, w: unit, d: half }, mark: "up" });
        caps.push({ id: "down", plan: { x: x + unit, y: y + half, w: unit, d: half }, mark: "down" });
        caps.push({ id: "right", plan: { x: x + unit * 2, y: y + half, w: unit, d: half }, mark: "right" });
      } else {
        caps.push({ id, plan: { x, y, w, d }, mark, homing: HOMING.has(id), touch: id === "power" && platform === "darwin" });
      }
      x += w;
    }
    y += d;
  });
  for (const cap of caps) {
    const { x, y, w, d } = cap.plan;
    cap.key = { x: x + GAP / 2, y: y + GAP / 2, w: w - GAP, d: d - GAP, r: 1.6 };
  }
  return caps.sort((a, b) => depth(a.key.x + a.key.w, a.key.y + a.key.d) - depth(b.key.x + b.key.w, b.key.y + b.key.d));
}

// Where a cap's opening is cut in the deck.
const openingOf = (cap) => k.planOutline(k.insetPlan(cap.key, -0.5), CASE_H, P, 3);
const holes = (caps) => caps.map((cap) => fill(openingOf(cap), "h")).join("") + line(caps.map(openingOf).join(""), "faint");

function capSvg(cap, { lit = false, pressed = false } = {}) {
  const z = CASE_H + (pressed ? CAP_LIFT - TRAVEL : CAP_LIFT);
  const surface = z + CAP_H;
  const plan = cap.key;
  const parts = [solid(k.slabOf(plan, z, CAP_H, P, 3, 0.5), { tone: "mid", crease: "faint", lit })];
  parts.push(line(k.planOutline(k.insetPlan(plan, 1.5), surface, P, 3), lit ? "lit" : "faint", lit ? { opacity: 0.45 } : {}));
  const cx = plan.x + plan.w / 2;
  const cy = plan.y + plan.d / 2;
  if (cap.homing) parts.push(line(k.lineOnTop([cx - 2.2, plan.y + plan.d - 3.2], [cx + 2.2, plan.y + plan.d - 3.2], surface, P), "lo", { width: 1 }));
  if (cap.touch) parts.push(line(k.ring(cx, cy, 4.4, surface, P, 28), "lo"));
  if (cap.mark) {
    const arrow = { up: [[-1, -1], [0, 1], [1, -1]], down: [[-1, 1], [0, -1], [1, 1]], left: [[1, -1], [-1, 0], [1, 1]], right: [[-1, -1], [1, 0], [-1, 1]] }[cap.mark];
    if (arrow) {
      parts.push(line(k.pathOf(arrow.map(([u, v]) => at([cx + u * 1.6, cy - v * 1.3, surface])), true), "mid"));
    } else {
      const size = SYMBOLS[cap.mark] ? 7.5 : 4.6;
      parts.push(line(legend(cap.mark, cx, cy, surface, size), lit ? "lit" : "mid", { width: lit ? 1.05 : 0.85 }));
    }
  }
  return parts.join("");
}

function keyboard(platform) {
  const out = [halo(CASE, 0)];
  out.push(solid(k.slabOf(CASE, MAT_H, CASE_H - MAT_H, P, 6, 1.2), { tone: "mid", inner: line(k.sideSeam(CASE, MAT_H + 2.2, P, 6), "faint") }));
  // The deck's milled edge and, on the right side, the power slider.
  out.push(line(k.planOutline(k.insetPlan(CASE, 3.2), CASE_H, P, 6), "faint"));
  const slider = { x: CASE.x + CASE.w, y: CASE.y + 8, w: 1, d: 9, r: 0.8 };
  out.push(solid(k.slabOf(slider, 3, 2.4, P, 3), { tone: "lo", crease: "none" }));
  out.push(solid(k.slabOf({ x: slider.x + 1, y: slider.y + 1.5, w: 0.8, d: 3.4, r: 0.4 }, 3.4, 1.6, P, 2), { tone: "mid", crease: "none" }));
  const caps = capsOf(platform);
  out.push(holes(caps));
  for (const cap of caps) out.push(capSvg(cap));
  // Screws that hold the deck to the case, at the back corners.
  out.push(dots([at([CASE.x + 3.6, CASE.y + 3.6, CASE_H]), at([CASE.x + CASE.w - 3.6, CASE.y + 3.6, CASE_H])], { size: 0.55 }));
  return out.join("");
}

// The screen box a cap covers when raised.
function boundsOf(cap) {
  const points = k.boxCorners(cap.key, CASE_H, CASE_H + CAP_LIFT + CAP_H).map(at);
  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  return { x0: Math.min(...xs), x1: Math.max(...xs), y0: Math.min(...ys), y1: Math.max(...ys) };
}
const overlaps = (a, b) => a.x0 < b.x1 && b.x0 < a.x1 && a.y0 < b.y1 && b.y0 < a.y1;

// The chosen key over the base: the lit cap, raised or pressed, and the
// caps around it again in depth order, so the base's cap is fully covered
// and nearer caps still hide it.
function chosenKey(platform, chosen, pressed) {
  const caps = capsOf(platform);
  const target = caps.find((cap) => cap.id === chosen);
  const box = boundsOf(target);
  const near = caps.filter((cap) => overlaps(boundsOf(cap), box));
  const out = [];
  // Pressed, the raised cap is gone: the bare deck shows where it was.
  if (pressed) out.push(fill(k.slabOf(target.key, CASE_H, CAP_LIFT + CAP_H, P, 3).fill, "t"));
  out.push(holes(near));
  for (const cap of near) out.push(capSvg(cap, cap === target ? { lit: true, pressed } : {}));
  return out.join("");
}

function mat() {
  return [
    halo(MAT, 0),
    solid(k.slabOf(MAT, 0, MAT_H, P, 6, 0.6), { tone: "lo" }),
    // The stitched hem, 4 in from the edge, as a run of short stitches.
    `<g stroke-dasharray="1.6 1.4">${line(k.planOutline(k.insetPlan(MAT, 4), MAT_H, P, 6), "lo")}</g>`,
  ].join("");
}

const PLUG = { x: MIC.x - MIC_BASE_R - 6, y: MIC.y - 2.6, w: 7, d: 5.2, r: 1.2 };

function cable() {
  const z = MAT_H + 1.1;
  const end = [PLUG.x, PLUG.y + PLUG.d / 2];
  const route = [
    [PORT.x, PORT.y - 6, PORT.z],
    [PORT.x, PORT.y - 12, z + 0.6],
    [PORT.x + 1, PORT.y - 20, z],
    [end[0] - 12, end[1] + 14, z],
    [end[0] - 6, end[1] + 4, z],
    [end[0], end[1], z],
  ];
  const d = k.pathOf(route.map(at));
  const out = [];
  // The plug's strain relief behind the keyboard, then the cord on the mat.
  out.push(solid(k.slabOf({ x: PORT.x - 2.2, y: PORT.y - 7, w: 4.4, d: 7, r: 1.2 }, PORT.z - 1.6, 3.2, P, 3), { tone: "lo", crease: "none" }));
  out.push(line(d, "lo", { width: 2.6 }), line(d, "paper", { width: 1.4 }), line(d, "faint", { width: 0.5 }));
  return out.join("");
}

const MIC_BASE_TOP = MAT_H + 1.6 + 6;
const MIC_SHELL = MIC_BASE_TOP + 87;
const MIC_R = 13;
const MIC_RING = MIC_SHELL + 16;

function microphone() {
  const { x, y } = MIC;
  const top = MIC_BASE_TOP;
  const out = [halo({ x: x - MIC_BASE_R, y: y - MIC_BASE_R, w: MIC_BASE_R * 2, d: MIC_BASE_R * 2, r: MIC_BASE_R }, MAT_H)];
  // A rubber ring under a weighted base with a bolt circle and a gain knob.
  out.push(solid(k.cylinder(x, y, MIC_BASE_R - 1.4, MAT_H, 1.6, P, 48), { tone: "lo", crease: "none" }));
  out.push(solid(k.cylinder(x, y, MIC_BASE_R, MAT_H + 1.6, 6, P, 56, 1), { tone: "mid" }));
  out.push(line(k.ring(x, y, MIC_BASE_R - 4, top, P, 48), "faint"));
  const bolts = Array.from({ length: 6 }, (_, i) => (i / 6) * Math.PI * 2 + 0.3).map((a) => at([x + Math.cos(a) * (MIC_BASE_R - 7), y + Math.sin(a) * (MIC_BASE_R - 7), top]));
  out.push(dots(bolts, { size: 0.55 }));
  // The cord's connector shell, plugged into the base's side.
  out.push(solid(k.slabOf(PLUG, MAT_H + 1.8, 3.6, P, 3), { tone: "mid", crease: "none" }));
  const knob = [x + 8, y + 9];
  out.push(solid(k.cylinder(knob[0], knob[1], 3.2, top, 4, P, 28, 0.4), { tone: "mid", inner: line(k.knurl(knob[0], knob[1], 3.2, top, top + 4, 28, P), "lo") }));
  out.push(line(k.lineOnTop(knob, [knob[0] + 2, knob[1] + 2], top + 4, P), "hi"));

  // Stem with a clamp collar and thumb screw.
  out.push(solid(k.cylinder(x, y, 2.6, top, 84, P, 24), { tone: "mid" }));
  const collar = top + 66;
  out.push(solid(k.cylinder(x, y, 4.2, collar, 5, P, 32, 0.5), { tone: "mid" }));
  out.push(solid(k.cylinder(x + 4.4, y + 2.4, 1.4, collar + 1.6, 1.8, P, 14), { tone: "lo", crease: "none" }));

  // The capsule: a neck, a shell with a status ring and a grille, a cap.
  out.push(solid(k.cylinder(x, y, MIC_R - 2, top + 84, MIC_SHELL - top - 84, P, 40), { tone: "mid" }));
  out.push(solid(k.cylinder(x, y, MIC_R, MIC_SHELL, 44, P, 56, 0.6), { tone: "mid", inner: line(k.sideArc(x, y, MIC_R, MIC_SHELL + 9, P, 36), "faint") }));
  const grille = [];
  for (let z = MIC_RING + 3; z < MIC_SHELL + 42; z += 2.1) grille.push(k.sideArc(x, y, MIC_R, z, P, 36));
  out.push(line(grille.join(""), "lo", { width: 0.55 }));
  out.push(ring(false));
  const capTop = MIC_SHELL + 44;
  out.push(solid(k.cylinder(x, y, MIC_R - 1, capTop, 2, P, 48, 0.6), { tone: "mid" }));
  out.push(line(k.ring(x, y, MIC_R - 5, capTop + 2, P, 40) + k.ring(x, y, MIC_R - 8.5, capTop + 2, P, 28), "faint"));
  out.push(dots([at([x, y, capTop + 2])], { size: 0.7, opacity: 0.6 }));
  return out.join("");
}

// The capsule's status ring: dark at rest, live while the key is held.
const ring = (live) => line(k.sideArc(MIC.x, MIC.y, MIC_R, MIC_RING, P, 44), live ? "live" : "faint", { width: live ? 2 : 1.3 });

const svgOf = (theme, body) =>
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${WIDTH} ${HEIGHT}" width="${WIDTH}" height="${HEIGHT}">${styleOf(PALETTES[theme])}<g fill="none" stroke-width="${STROKE}" stroke-linecap="round" stroke-linejoin="round">${body}</g></svg>\n`;

rmSync(OUT, { recursive: true, force: true });
mkdirSync(OUT, { recursive: true });
const written = [];
const write = (name, theme, svg) => {
  writeFileSync(join(OUT, name), svg);
  written.push({ name, theme, svg });
};
for (const [platform, keys] of Object.entries(KEYS)) {
  for (const theme of Object.keys(PALETTES)) {
    write(`${platform}-${theme}.svg`, theme, svgOf(theme, [mat(), microphone(), cable(), keyboard(platform)].join("")));
    for (const key of keys) {
      write(`${key}-${theme}.svg`, theme, svgOf(theme, chosenKey(platform, key, false)));
      write(`${key}-${theme}-pressed.svg`, theme, svgOf(theme, ring(true) + chosenKey(platform, key, true)));
    }
  }
}
const total = written.reduce((sum, w) => sum + w.svg.length, 0);
console.log(`wrote ${written.length} figures to ${OUT} (${Math.round(total / 1024)} KB in all)`);

if (process.argv.includes("--preview")) {
  // Each base with its chosen keys, stacked as the app stacks them.
  const dir = join(ROOT, ".workspace/figures");
  mkdirSync(dir, { recursive: true });
  const layer = (name) => `<img src="${join(OUT, name)}" style="position:absolute;inset:0">`;
  const cells = [];
  for (const [platform, keys] of Object.entries(KEYS))
    for (const theme of Object.keys(PALETTES))
      for (const key of keys)
        for (const pressed of [false, true])
          cells.push(`<figure style="margin:0;position:relative;width:${WIDTH}px;height:${HEIGHT}px;background:${PALETTES[theme].paper}">${layer(`${platform}-${theme}.svg`)}${layer(`${key}-${theme}.svg`)}${pressed ? layer(`${key}-${theme}-pressed.svg`) : ""}</figure>`);
  writeFileSync(join(dir, "holdkey.html"), `<!doctype html><meta charset="utf-8"><body style="margin:0;display:flex;flex-wrap:wrap;gap:8px;background:#777">${cells.join("")}`);
  console.log(`preview ${join(dir, "holdkey.html")}`);
}
