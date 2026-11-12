/*
 * Mock recordings for the "Arion 0.1.5 to 0.1.6" post, drawn in the manner of
 * a screen-recording editor: a rounded recording on a wallpaper, an eased
 * camera that zooms to where the action is, a smoothed cursor with click
 * ripples, and a keystroke overlay. Every scene is an illustration; nothing is
 * recorded, no model runs and no assistant is opened.
 *
 * stage.<locale>.html sets window.ARION_COPY and loads this file; the URL hash
 * picks the scene (#shortcut, #capture, #polish, #textcolor, #files,
 * #selection, #disable, #thumbnails). The page plays only while it is in view
 * and, under reduced motion, shows one finished frame. The engine (clock,
 * camera, cursor, keystroke overlay and pill) follows the 0.1.4 post's
 * stage.js.
 */
(() => {
  "use strict";

  const C = window.ARION_COPY;
  const U = C.ui;
  const sceneName = location.hash.slice(1) || "shortcut";
  const still = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const W = 1184;
  const H = 712;

  /* ---------- DOM ---------- */

  function h(spec, props, ...kids) {
    const [tag, ...classes] = spec.split(".");
    const el = document.createElement(tag || "div");
    if (classes.length) el.className = classes.join(" ");
    for (const [key, value] of Object.entries(props || {})) {
      if (key === "text") el.textContent = value;
      else el.setAttribute(key, value);
    }
    for (const kid of kids.flat()) if (kid != null && kid !== false) el.append(kid);
    return el;
  }

  function svg(markup, size = 16) {
    const holder = document.createElement("span");
    holder.innerHTML = `<svg viewBox="0 0 24 24" width="${size}" height="${size}" aria-hidden="true">${markup}</svg>`;
    return holder.firstChild;
  }

  const ICON = {
    wifi: '<path d="M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M2 9a15 15 0 0 1 20 0"/><circle cx="12" cy="19.5" r=".8"/>',
    battery: '<rect x="2" y="7" width="18" height="10" rx="2.5"/><path d="M22 11v2"/><path d="M5 10h10v4H5z" fill="currentColor"/>',
    check: '<path d="M5 12.5l4.5 4.5L19 7.5"/>',
    copy: '<rect x="8" y="8" width="12" height="12" rx="2.5"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>',
    pencil: '<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="M13.5 6.5l4 4"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    minus: '<path d="M5 12h14"/>',
    close: '<path d="M6 6l12 12M18 6L6 18"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5"/><circle cx="12" cy="7.8" r=".6" fill="currentColor"/>',
    chevron: '<path d="M6 9l6 6 6-6"/>',
    book: '<path d="M12 6.5C10 5 7 4.5 3.5 5v13c3.5-.5 6.5 0 8.5 1.5 2-1.5 5-2 8.5-1.5V5C17 4.5 14 5 12 6.5z"/><path d="M12 6.5v13"/>',
    ret: '<path d="M20 5v6a3 3 0 0 1-3 3H5"/><path d="M9 10l-4 4 4 4"/>',
    search: '<circle cx="11" cy="11" r="6.5"/><path d="M20 20l-4.2-4.2"/>',
    file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
    text: '<path d="M5 6h14M5 12h14M5 18h9"/>',
    send: '<path d="M12 19V5M6 11l6-6 6 6"/>',
    claude: '<path d="M4 20L10.5 4h3L20 20h-3.4l-4.6-12-4.6 12z" fill="currentColor" stroke="none"/>',
    openai:
      '<path d="M12 3.5a4 4 0 0 1 4 3.2 4 4 0 0 1 3.4 5.8 4 4 0 0 1-3.4 5.8A4 4 0 0 1 8 17.3a4 4 0 0 1-3.4-5.8A4 4 0 0 1 8 6.7a4 4 0 0 1 4-3.2z"/><path d="M12 8.5l3 1.7v3.6L12 15.5l-3-1.7v-3.6z"/>',
  };

  const stage = h("div.stage", { "aria-hidden": "true" });
  const camera = h("div.camera");
  const desk = h("div.desk");
  const hud = h("div.hud");
  const card = h("div.card");
  camera.append(desk);
  stage.append(h("div.wall"), h("div.frame", null, camera, hud, card));
  document.body.append(stage);
  if (still) document.documentElement.classList.add("still");

  function fit() {
    stage.style.setProperty("--fit", String(Math.min(innerWidth / 1280, innerHeight / 800)));
  }
  addEventListener("resize", fit);
  fit();

  /* ---------- Clock: everything waits on frames that only advance in view ---------- */

  let playing = false;
  const jobs = new Set();
  let last = performance.now();

  function frame(now) {
    const dt = Math.min(now - last, 64);
    last = now;
    if (playing) {
      for (const job of [...jobs]) {
        job.t += dt;
        const p = job.dur > 0 ? Math.min(job.t / job.dur, 1) : 1;
        if (job.step) job.step(p);
        if (p >= 1) {
          jobs.delete(job);
          job.resolve();
        }
      }
    }
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);

  function run(dur, step) {
    if (still) {
      if (step) step(1);
      return Promise.resolve();
    }
    return new Promise((resolve) => jobs.add({ t: 0, dur, step, resolve }));
  }

  const wait = (ms) => run(ms);
  /** Under reduced motion the scene stops here and shows this frame. */
  const hold = () => (still ? new Promise(() => {}) : Promise.resolve());

  const ease = {
    inOut: (t) => (t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2),
    camera: (t) => (t < 0.5 ? 8 * t ** 4 : 1 - Math.pow(-2 * t + 2, 4) / 2),
  };

  /* ---------- Camera ---------- */

  let cam = { x: W / 2, y: H / 2, s: 1 };

  function clampCam({ x, y, s }) {
    const hw = W / (2 * s);
    const hh = H / (2 * s);
    return { s, x: Math.min(Math.max(x, hw), W - hw), y: Math.min(Math.max(y, hh), H - hh) };
  }

  function applyCam() {
    camera.style.transform = `translate(${W / 2}px, ${H / 2}px) scale(${cam.s}) translate(${-cam.x}px, ${-cam.y}px)`;
  }
  applyCam();

  function zoom(target, dur = 1150) {
    const from = { ...cam };
    const to = clampCam({ x: target.x ?? W / 2, y: target.y ?? H / 2, s: target.s ?? 1 });
    return run(dur, (p) => {
      const e = ease.camera(p);
      const s = Math.exp(Math.log(from.s) + (Math.log(to.s) - Math.log(from.s)) * e);
      cam = { s, x: from.x + (to.x - from.x) * e, y: from.y + (to.y - from.y) * e };
      applyCam();
    });
  }

  /** A point of an element in desk coordinates (fx, fy: fraction of its box). */
  function at(el, fx = 0.5, fy = 0.5) {
    const d = desk.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    const k = d.width / W;
    return { x: (r.left - d.left + r.width * fx) / k, y: (r.top - d.top + r.height * fy) / k };
  }

  const mid = (a, b) => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 });

  /* ---------- Cursor ---------- */

  const cursor = h("div.cursor");
  cursor.innerHTML =
    '<svg viewBox="0 0 26 26" width="26" height="26"><path d="M5 3.5v17.2l4.3-4.1 2.8 6.4 2.9-1.3-2.8-6.3 6-.1z" fill="#14110e" stroke="#fff" stroke-width="1.6" stroke-linejoin="round"/></svg>';
  let pos = { x: 0, y: 0 };

  function placeCursor(x, y) {
    pos = { x, y };
    cursor.style.translate = `${x - 5}px ${y - 3.5}px`;
  }

  /** Eased move along a gentle arc, the way recording editors smooth a pointer. */
  function move(point, dur = 900, onStep) {
    cursor.classList.remove("idle");
    const from = { ...pos };
    const dx = point.x - from.x;
    const dy = point.y - from.y;
    const cx = (from.x + point.x) / 2 - dy * 0.12;
    const cy = (from.y + point.y) / 2 + dx * 0.12;
    return run(dur, (p) => {
      const t = ease.inOut(p);
      const u = 1 - t;
      placeCursor(u * u * from.x + 2 * u * t * cx + t * t * point.x, u * u * from.y + 2 * u * t * cy + t * t * point.y);
      if (onStep) onStep(t);
    });
  }

  async function click() {
    cursor.classList.add("down");
    const ripple = h("div.ripple");
    ripple.style.left = `${pos.x}px`;
    ripple.style.top = `${pos.y}px`;
    desk.append(ripple);
    await wait(120);
    cursor.classList.remove("down");
    wait(520).then(() => ripple.remove());
  }

  /** Recording editors hide a pointer that stops moving while you type. */
  const idle = () => cursor.classList.add("idle");

  /* ---------- Keystroke overlay ---------- */

  function hudShow(keys, note) {
    const caps = keys.map((key) => h("kbd", { text: key }));
    hud.replaceChildren(...caps, ...(note ? [h("span.note", { text: note })] : []));
    hud.classList.add("shown");
    return caps;
  }

  const hudHide = () => hud.classList.remove("shown");

  async function press(keys, note, ms = 150) {
    const caps = hudShow(keys, note);
    caps.forEach((cap) => cap.classList.add("down"));
    await wait(ms);
    caps.forEach((cap) => cap.classList.remove("down"));
  }

  /* ---------- Annotation card ---------- */

  function note(copy, top = true) {
    card.classList.toggle("top", top);
    card.replaceChildren(h("span.k", { text: copy.k }), h("span.chip-scene", { text: copy.note }));
    card.classList.add("shown");
  }

  const noteHide = () => card.classList.remove("shown");

  /* ---------- Dictation pill ---------- */

  const BAR_HEIGHTS = [6, 10, 16, 12, 20, 14, 9, 17, 11, 7, 5];
  const bars = h("span.bars");
  BAR_HEIGHTS.forEach((height, index) => {
    const bar = h("span");
    bar.style.setProperty("--h", `${height}px`);
    bar.style.animationDelay = `${index * 60}ms`;
    bars.append(bar);
  });
  const draft = h("bdi");
  const doneText = h("span");
  const noteTitle = h("b");
  const noteHint = h("small");
  const pill = h(
    "div.pill",
    { "data-state": "listening" },
    h("span.live", null, h("span.dot"), bars, h("span.draft", null, draft)),
    h("span.wait", null, h("span.orbit", null, h("i"), h("i"), h("i"))),
    h("span.done", null, svg(ICON.check), doneText),
    h("span.note", null, svg(ICON.info), h("span", null, noteTitle, noteHint)),
  );
  const ruler = h("span");
  Object.assign(ruler.style, { position: "absolute", visibility: "hidden", whiteSpace: "pre", fontSize: "15px" });
  let pillWidth = 150;

  function setPillWidth(width) {
    pillWidth = width;
    pill.style.width = `${width}px`;
  }

  function textWidth(text) {
    ruler.textContent = text;
    return ruler.offsetWidth;
  }

  function pillListen() {
    pill.dataset.state = "listening";
    draft.textContent = "";
    setPillWidth(150);
    pill.classList.add("shown");
  }

  /** The pill widens in steps with the draft and never narrows mid-session. */
  function pillDraft(text) {
    draft.textContent = text;
    const want = Math.min(Math.ceil((textWidth(text) + 136) / 48) * 48, 560);
    if (want > pillWidth) setPillWidth(want);
  }

  function pillWait() {
    pill.dataset.state = "polishing";
    setPillWidth(110);
  }

  function pillDone(text) {
    pill.dataset.state = "inserted";
    doneText.textContent = text;
    setPillWidth(textWidth(text) + 78);
  }

  function pillNote(title, hint) {
    pill.dataset.state = "note";
    noteTitle.textContent = title;
    noteHint.textContent = hint;
    pill.classList.add("tall");
    setPillWidth(Math.max(textWidth(title), textWidth(hint) * 0.87) + 84);
  }

  const pillHide = () => pill.classList.remove("shown");

  /* ---------- Shared pieces ---------- */

  const caret = () => h("span.caret");

  function menubar(app) {
    return h(
      "div.menubar",
      null,
      h("b", { text: app }),
      C.menu.map((item) => h("span", { text: item })),
      h("span.spacer"),
      h("span.mark"),
      svg(ICON.wifi),
      svg(ICON.battery),
      h("span", { text: C.clock }),
    );
  }

  function appWindow(title, ...content) {
    return h(
      "div.window",
      null,
      h("div.titlebar", null, h("span.lights", null, h("i"), h("i"), h("i")), h("span.title", { text: title })),
      h("div.body", null, content),
    );
  }

  function message({ who, text, me }, enter) {
    return h(
      `div.msg${me ? ".me" : ""}${enter ? ".enter" : ""}`,
      null,
      h("span.avatar", { text: who.slice(0, 1) }),
      h("div", null, h("div.who", { text: who }), h("div.text", { text })),
    );
  }

  function mount(...nodes) {
    desk.replaceChildren(...nodes, pill, ruler, cursor);
  }

  /** Puts a node above the windows but under the pill and cursor. */
  const lay = (node) => desk.insertBefore(node, pill);

  function reset() {
    jobs.clear();
    hudHide();
    card.classList.remove("shown", "top");
    pill.classList.remove("shown", "tall");
    cursor.classList.remove("idle");
    cam = { x: W / 2, y: H / 2, s: 1 };
    applyCam();
  }

  function chatApp() {
    const copy = C.chat;
    const typed = h("span");
    const placeholder = h("span.placeholder", { text: copy.composer });
    const composer = h("div.composer", null, placeholder);
    const thread = h("div.thread", null, copy.thread.map((m) => message(m)));
    mount(
      menubar(copy.app),
      appWindow(
        copy.app,
        h("div.sidebar", null, copy.channels.map((name, i) => h(i === 0 ? "span.on" : "span", { text: name }))),
        h("div.chat", null, thread, composer),
      ),
    );
    return { typed, placeholder, composer, thread };
  }

  async function focusComposer(app) {
    await move(at(app.composer, 0.3, 0.5), 950);
    await click();
    app.composer.classList.add("focused");
    app.placeholder.replaceWith(app.typed, caret());
    await wait(150);
    idle();
  }

  /* ---------- Arion Settings ---------- */

  function settingsWindow(navOn, title) {
    const page = h("div.spage", null, h("div.ptitle", { text: title }));
    const nav = U.nav.map((name, i) => h(i === navOn ? "span.on" : "span", { text: name }));
    const win = appWindow(U.settingsTitle, h("div.snav", null, nav), page);
    win.classList.add("settings");
    mount(menubar("Arion"), win);
    return { win, page };
  }

  function srow(label, control, extra) {
    return h("div.srow", null, h("span.slabel", { text: label }), extra, control);
  }

  function select(text, width = 200) {
    const el = h("span.input.select", { style: `width: ${width}px` }, h("span", { text }));
    return el;
  }

  function toggle(on) {
    return h(`span.switch${on ? ".on" : ""}`, null, h("i"));
  }

  /** Opens a select's menu, moves to option `pick` and chooses it. */
  async function choose(field, labels, pick, decorate) {
    await move(at(field, 0.6, 0.5), 700);
    await click();
    const options = labels.map((label, i) => {
      const option = h(i === 0 ? "span.sel" : "span", null, decorate ? decorate(i) : null, h("span", { text: label }));
      return option;
    });
    const menu = h("div.menu", { style: `width: ${field.offsetWidth}px` }, options);
    field.append(menu);
    await wait(450);
    const choice = options[pick];
    await move(at(choice, 0.4, 0.5), 520);
    options.forEach((option) => option.classList.toggle("sel", option === choice));
    await wait(250);
    await click();
    menu.remove();
  }

  /* 0.1.5: a + beside the dictation key adds a second one. */
  async function shortcutScene() {
    const S = U.shortcut;
    const { page } = settingsWindow(2, S.title);
    const field = (text, recording) =>
      h(
        `span.kfield${recording ? ".recording" : ""}`,
        null,
        h("span", { text }),
        svg(recording ? ICON.close : ICON.pencil, 15),
      );
    const plus = h("span.ibtn", null, svg(ICON.plus, 15));
    const first = srow(S.label, h("span.kctl", null, field(S.first), plus), h("span.i", null, svg(ICON.info, 13)));
    const group = h("div.group", null, first);
    page.append(group);
    placeCursor(980, 620);

    await wait(500);
    await zoom({ ...at(group, 0.55, 1.1), s: 1.6 });
    await move(at(plus, 0.5, 0.5), 900);
    await wait(200);
    await click();
    plus.remove();
    const recorder = field(S.recording, true);
    const second = h("div.srow.extra", null, h("span.slabel"), h("span.kctl", null, recorder, h("span.ibtn.ghost")));
    group.append(second);
    await wait(700);
    idle();
    await press([C.keys.second], null, 260);
    await wait(150);
    hudHide();
    recorder.replaceWith(field(S.second));
    second.querySelector(".ghost").replaceWith(h("span.ibtn", null, svg(ICON.minus, 15)));
    await wait(500);
    note(C.notes.shortcut, false);
    card.querySelector(".k").after(h("span.keys", null, h("kbd", { text: S.first }), h("kbd", { text: S.second })));
    await wait(1600);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  /* 0.1.5: the microphone opens first, and a long draft keeps its end in view. */
  async function captureScene() {
    const copy = C.chat;
    const app = chatApp();
    placeCursor(960, 230);

    await wait(500);
    await focusComposer(app);
    await zoom({ ...mid(at(app.composer, 0.5, 0.2), at(pill, 0.5, 0.5)), s: 1.5 });

    const [key] = hudShow([copy.key]);
    key.classList.add("down");
    pillListen();
    await wait(250);
    note(C.notes.capture);
    for (const partial of copy.partials) {
      pillDraft(partial);
      await wait(620);
    }
    await wait(500);
    noteHide();
    key.classList.remove("down");
    await wait(160);
    hudHide();
    app.typed.textContent = copy.final;
    app.typed.className = "reveal";
    pillDone(U.pasted);
    await wait(1100);
    pillHide();
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  /* 0.1.5: a fallback says why, and reasoning can be allowed when required. */
  async function polishScene() {
    const copy = C.chat;
    const P = U.polish;
    const app = chatApp();
    placeCursor(960, 230);

    await wait(500);
    await focusComposer(app);
    await zoom({ ...mid(at(app.composer, 0.5, 0.2), at(pill, 0.5, 0.5)), s: 1.6 });
    const [key] = hudShow([copy.key]);
    key.classList.add("down");
    pillListen();
    for (const partial of copy.short) {
      pillDraft(partial);
      await wait(520);
    }
    key.classList.remove("down");
    await wait(160);
    hudHide();
    pillWait();
    await wait(900);
    app.typed.textContent = copy.shortFinal;
    app.typed.className = "reveal";
    pillDone(U.pasted);
    await wait(900);
    pillNote(P.noteTitle, P.noteHint);
    await wait(2600);
    pillHide();
    await zoom({ s: 1 }, 900);

    const { page } = settingsWindow(5, U.nav[5]);
    page.firstChild.replaceChildren(h("span.back", { text: "‹" }), h("span", { text: P.cloudTitle }));
    const sw = toggle(false);
    const group = h(
      "div.group",
      null,
      h(
        "div.srow.tall",
        null,
        h("span.pmark", { text: "R" }),
        h("span.slabel", null, "OpenRouter", h("span.sub", { text: P.keyLabel })),
        h("span.btn", { text: P.disconnect }),
      ),
      srow(P.modelLabel, select("openai/gpt-5-mini", 240)),
      srow(P.reasoning, sw, h("span.i", null, svg(ICON.info, 13))),
    );
    page.append(h("div.glabel", { text: P.inUse }), group);
    placeCursor(700, 600);
    await wait(400);
    await zoom({ ...at(group, 0.5, 0.8), s: 1.5 });
    await move(at(sw, 0.5, 0.5), 900);
    await wait(150);
    await click();
    sw.classList.add("on");
    await wait(500);
    note(C.notes.polish, false);
    await wait(1800);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  /* ---------- Candidate bar ---------- */

  function candidateBar() {
    const items = C.candidates.map((text, i) => h(i === 0 ? "span.candidate.pick" : "span.candidate", null, h("small", { text: String(i + 1) }), text));
    const bar = h("div.cbar", null, h("div.preedit", { text: C.pinyin }), h("div.candidates", null, items));
    return { bar, pick: items[0] };
  }

  function paintPick(pick, fill, text) {
    pick.style.background = fill;
    pick.style.color = text === "white" ? "#FFFFFF" : "#14110F";
  }

  /* 0.1.5: the highlighted candidate's text follows the fill or a pinned color. */
  async function textColorScene() {
    const K = U.keyboard;
    const { page } = settingsWindow(3, K.title);
    const automatic = (color) => K.automatic.replace("{value}", color === "white" ? K.white : K.black);
    const swatch = (fill) => h("i.swatch", { style: `background: ${fill}` });
    const textField = select(automatic("black"), 220);
    const accentField = h("span.input.select", { style: "width: 220px" }, swatch(K.accents[0][1]), h("span", { text: K.accents[0][0] }));
    const group = h(
      "div.group",
      null,
      srow(K.pageSize, select("6", 140)),
      srow(K.size, select(K.sizeValue, 140)),
      srow(K.appearance, select(K.appearanceValue, 140)),
      srow(K.textColor, textField),
      srow(K.accent, accentField),
    );
    page.append(h("div.glabel", { text: K.group }), group);
    const { bar, pick } = candidateBar();
    lay(bar);
    paintPick(pick, K.accents[0][1], "black");
    placeCursor(900, 640);

    await wait(500);
    await zoom({ ...mid(at(group, 0.7, 0.8), at(bar, 0.5, 0.5)), s: 1.35 });
    const [name, fill, recommended] = K.accents[K.pick];
    await choose(accentField, K.accents.map(([label]) => label), K.pick, (i) => swatch(K.accents[i][1]));
    accentField.replaceChildren(swatch(fill), h("span.reveal", { text: name }));
    textField.firstChild.textContent = automatic(recommended);
    textField.firstChild.className = "reveal";
    paintPick(pick, fill, recommended);
    await wait(1300);

    await choose(textField, [automatic(recommended), K.black, K.white], 1);
    textField.firstChild.textContent = K.black;
    paintPick(pick, fill, "black");
    await wait(900);
    await move({ x: pos.x + 80, y: pos.y + 140 }, 600);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  /* 0.1.5: file history is a separate opt-in with one confirmation. */
  async function filesScene() {
    const F = U.files;
    const { page } = settingsWindow(7, F.title);
    const sw = toggle(false);
    const group = h(
      "div.group",
      null,
      srow(F.history, toggle(true), h("span.i", null, svg(ICON.info, 13))),
      srow(F.shortcut, h("span.kfield", null, h("span", { text: C.keys.clipboard.join("") }), svg(ICON.pencil, 15)), h("span.i", null, svg(ICON.info, 13))),
      srow(F.size, select(F.sizeValue, 140), h("span.i", null, svg(ICON.info, 13))),
      srow(F.record, sw, h("span.i", null, svg(ICON.info, 13))),
    );
    page.append(group);
    placeCursor(980, 620);

    await wait(500);
    await zoom({ ...at(group, 0.6, 0.85), s: 1.5 });
    await move(at(sw, 0.5, 0.5), 900);
    await click();
    await wait(200);
    const confirm = h("span.btn.primary", { text: F.confirm });
    const dialog = h(
      "div.dialog",
      null,
      h("div.dhead", null, h("b", { text: F.confirmTitle }), svg(ICON.close, 15)),
      h("p", { text: F.explanation }),
      h("p", { text: F.mac }),
      h("div.dfoot", null, h("span.btn.plain", { text: F.cancel }), confirm),
    );
    lay(dialog);
    await zoom({ ...at(dialog, 0.5, 0.55), s: 1.55 }, 1000);
    await move(at(confirm, 0.5, 0.5), 900);
    await wait(250);
    await click();
    dialog.classList.add("leave");
    await wait(200);
    dialog.remove();
    sw.classList.add("on");
    await zoom({ ...at(group, 0.6, 0.85), s: 1.5 }, 900);
    note(C.notes.files, false);
    await wait(1800);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  /* ---------- Selection actions ---------- */

  function browser() {
    const A = C.article;
    const words = A.selected.split(" ").map((word, i, all) => h("span.w", { text: i < all.length - 1 ? `${word} ` : word }));
    const target = h("span.target", null, words);
    const second = A.second.split(" ").map((word, i, all) => h("span.w", { text: i < all.length - 1 ? `${word} ` : word }));
    const win = appWindow(
      A.browser,
      h(
        "div.page",
        null,
        h("div.url", { text: `${A.host}/local-first` }),
        h("article", null, h("h2", { text: A.heading }), h("p", null, A.before, target, A.after), h("p", null, second)),
      ),
    );
    mount(menubar(A.browser), win);
    return { words, second, win };
  }

  /** Drags across `words`, highlighting each as the pointer passes it. */
  async function drag(words) {
    await move(at(words[0], 0.02, 0.55), 900);
    cursor.classList.add("down");
    await wait(120);
    const end = at(words[words.length - 1], 0.98, 0.55);
    await move(end, 1100, () => {
      for (const word of words) word.classList.toggle("hl", at(word, 0, 0.5).x <= pos.x);
    });
    words.forEach((word) => word.classList.add("hl"));
    cursor.classList.remove("down");
  }

  function selectionBar(words) {
    const S = U.selection;
    const button = (icon, label, chevron) =>
      h("span.sb", null, svg(icon, 16), h("span", { text: label }), chevron ? svg(ICON.chevron, 13) : null);
    const claude = button(ICON.claude, S.claude, true);
    const chatgpt = button(ICON.openai, S.chatgpt, true);
    const x = h("span.sb.x", null, svg(ICON.close, 15));
    const bar = h("div.sbar", null, claude, chatgpt, h("i.sep"), button(ICON.copy, S.copy), button(ICON.book, S.vocabulary), h("i.sep"), x);
    bar.style.left = `${Math.min(...words.map((word) => at(word, 0, 1).x)) - 8}px`;
    bar.style.top = `${at(words[words.length - 1], 0, 1).y + 10}px`;
    lay(bar);
    return { bar, claude, x };
  }

  /** A popup of the bar, created at its final size beside the control that opened it. */
  function popup(bar, rows, end) {
    const menu = h(`div.spop${end ? ".end" : ""}`, null, rows);
    const b = at(bar, 0, 1);
    menu.style.top = `${b.y + 4}px`;
    if (end) menu.style.left = `${at(bar, 1, 1).x - 224}px`;
    else menu.style.left = `${b.x}px`;
    lay(menu);
    return menu;
  }

  /* 0.1.6: select text, rest on an assistant, choose a prompt. */
  async function selectionScene() {
    const S = U.selection;
    const { words } = browser();
    placeCursor(860, 600);

    await wait(500);
    await zoom({ ...mid(at(words[0], 0, 1), at(words[words.length - 1], 1, 2)), s: 1.45 });
    await drag(words);
    await wait(150);
    const { bar, claude } = selectionBar(words);
    await wait(500);
    await zoom({ ...at(bar, 0.55, 2.4), s: 1.35 }, 900);
    await move(at(claude, 0.5, 0.55), 700);
    claude.classList.add("hover");
    await wait(300);
    claude.classList.add("open");
    const rows = S.prompts.map((title, i) => h(i === 0 ? "span.sel" : "span", null, h("span", { text: title }), i === 0 ? svg(ICON.ret, 14) : null));
    const ask = h("span.ask", { text: S.ask });
    const menu = popup(bar, [...rows, ask]);
    await wait(700);
    const choice = rows[C.pick];
    await move(at(choice, 0.3, 0.5), 650);
    rows.forEach((row) => row.classList.toggle("sel", row === choice));
    await wait(300);
    await click();
    menu.classList.add("leave");
    bar.classList.add("leave");
    await wait(180);
    menu.remove();
    bar.remove();

    const prompt = `${C.prompts[C.pick]}\n\n${C.article.selected}`;
    const composer = h("div.ccomposer", null, h("span.reveal", { text: prompt }), h("span.sendbtn", null, svg(ICON.send, 15)));
    const chat = h("div.claudewin", null, h("div.titlebar", null, h("span.lights", null, h("i"), h("i"), h("i")), h("span.title", { text: C.claude.app })), h("div.cbody", null, composer));
    lay(chat);
    await zoom({ ...at(chat, 0.5, 0.6), s: 1.35 }, 1000);
    card.classList.add("top");
    const query = encodeURIComponent(prompt).slice(0, 30);
    card.replaceChildren(h("span.k", { text: C.notes.selection.k }), h("code", { text: `claude://claude.ai/new?q=${query}…` }), h("span.chip-scene", { text: C.notes.selection.note }));
    card.classList.add("shown");
    await wait(2200);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2000);
  }

  /* 0.1.6: the X opens the disable menu; a disabled site gets no bar. */
  async function disableScene() {
    const S = U.selection;
    const A = C.article;
    const { words, second } = browser();
    placeCursor(860, 600);

    await wait(500);
    const picked = words.slice(0, 5);
    await zoom({ ...mid(at(picked[0], 0, 1), at(picked[4], 1, 2)), s: 1.45 });
    await drag(picked);
    await wait(150);
    const { bar, x } = selectionBar(picked);
    await wait(500);
    await zoom({ ...at(bar, 0.7, 2.6), s: 1.6 }, 900);
    await move(at(x, 0.5, 0.55), 800);
    x.classList.add("hover");
    await wait(300);
    const rows = [S.pause, S.site.replace("{site}", A.host), S.app.replace("{app}", A.browser), S.off].map((t) => h("span", { text: t }));
    const menu = popup(bar, rows, true);
    await wait(700);
    await move(at(rows[1], 0.3, 0.5), 650);
    rows[1].classList.add("sel");
    await wait(300);
    await click();
    menu.classList.add("leave");
    bar.classList.add("leave");
    await wait(180);
    menu.remove();
    bar.remove();
    words.forEach((word) => word.classList.remove("hl"));
    await wait(500);

    await zoom({ ...mid(at(second[0], 0, 1), at(second[5], 1, 2)), s: 1.45 }, 900);
    await drag(second.slice(0, 6));
    await wait(400);
    note(C.notes.disable, true);
    await wait(2000);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2000);
  }

  /* 0.1.6: a copied image file shows its picture in clipboard history. */
  async function thumbnailScene() {
    const D = C.doc;
    const typed = h("span");
    const caretEl = caret();
    const anchor = h("span.anchor");
    const line = h("p.live", null, D.before, typed, caretEl, anchor);
    caretEl.style.display = "none";
    mount(menubar(D.app), appWindow(D.app, h("div.doc", null, h("h2", { text: D.title }), D.lines.map((text) => h("p", { text })), line)));
    placeCursor(940, 560);

    await wait(500);
    await move(at(anchor, 0, 0.6), 900);
    await click();
    caretEl.style.display = "";
    await wait(150);
    idle();
    await zoom({ ...at(anchor, 0.5, 4), s: 1.6 });
    await press(C.keys.clipboard, null, 220);
    await wait(200);
    hudHide();

    const icons = { text: ICON.text, file: ICON.file };
    const rows = C.rows.map((row) =>
      h(
        "div.row",
        null,
        row.kind === "image" ? h("span.thumb.photo") : svg(icons[row.kind], 15),
        h("span.label", { text: row.text }),
        row.tag ? h("span.tag", { text: row.tag }) : null,
        h("span.ret", null, svg(ICON.ret, 15)),
      ),
    );
    const picker = h("div.picker", null, h("header", null, h("span.q", { text: U.clipboard.label }), svg(ICON.search, 15)), rows);
    anchor.append(picker);
    rows[0].classList.add("sel");
    await wait(800);
    await move(at(rows[0].firstChild, 0.5, 0.5), 700);
    await wait(450);
    const flyout = h("div.flyout", null, h("span.thumb.photo"));
    picker.append(flyout);
    await zoom({ ...mid(at(picker, 0.3, 0.5), at(flyout, 1, 0.5)), s: 1.3 }, 900);
    await wait(1400);
    await move({ x: pos.x + 40, y: pos.y + 120 }, 500);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  const scenes = {
    shortcut: shortcutScene,
    capture: captureScene,
    polish: polishScene,
    textcolor: textColorScene,
    files: filesScene,
    selection: selectionScene,
    disable: disableScene,
    thumbnails: thumbnailScene,
  };

  /* ---------- Playback: only while visible ---------- */

  let started = false;
  async function loop() {
    for (;;) {
      reset();
      await scenes[sceneName]();
    }
  }

  function setPlaying(visible) {
    playing = visible && !document.hidden;
    document.documentElement.classList.toggle("paused", !playing);
    if (playing && !started) {
      started = true;
      loop();
    }
  }

  let visible = false;
  new IntersectionObserver((entries) => {
    visible = entries[entries.length - 1].isIntersecting;
    setPlaying(visible);
  }, { threshold: 0.35 }).observe(document.body);
  document.addEventListener("visibilitychange", () => setPlaying(visible));
  if (still) {
    started = true;
    reset();
    scenes[sceneName]();
  }
})();
