/*
 * Mock recordings for the "Arion 0.1.1" post, drawn in the manner of a
 * screen-recording editor: a rounded recording on a wallpaper, an eased
 * camera that zooms to where the action is, a smoothed cursor with click
 * ripples, and a keystroke overlay. Every scene is an illustration; nothing
 * is recorded and no model runs.
 *
 * stage.<locale>.html sets window.ARION_COPY and loads this file; the URL
 * hash picks the scene (#dictation, #polish, #translate, #keyboard,
 * #clipboard). The page plays only while it is in view and, under reduced
 * motion, shows one finished frame.
 */
(() => {
  "use strict";

  const C = window.ARION_COPY;
  const sceneName = location.hash.slice(1) || "dictation";
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
    search: '<circle cx="11" cy="11" r="6.5"/><path d="M20 20l-4.2-4.2"/>',
    ret: '<path d="M20 5v6a3 3 0 0 1-3 3H5"/><path d="M9 10l-4 4 4 4"/>',
    file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
    text: '<path d="M5 6h14M5 12h14M5 18h9"/>',
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

  /** A camera target that keeps `el`'s left edge `margin` inside the view. */
  function fromLeft(el, s, dy = 0, margin = 56) {
    const start = at(el, 0, 0.5);
    return { x: start.x - margin + W / (2 * s), y: start.y + dy, s };
  }

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
  function move(point, dur = 900) {
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
  const pill = h(
    "div.pill",
    { "data-state": "listening" },
    h("span.live", null, h("span.dot"), bars, h("span.draft", null, draft)),
    h("span.wait", null, h("span.orbit", null, h("i"), h("i"), h("i"))),
    h("span.done", null, svg(ICON.check), doneText),
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
    const want = Math.min(Math.ceil((textWidth(text) + 136) / 48) * 48, 620);
    if (want > pillWidth) setPillWidth(want);
  }

  function pillPolish() {
    pill.dataset.state = "polishing";
    setPillWidth(96);
  }

  function pillDone(text) {
    pill.dataset.state = "inserted";
    doneText.textContent = text;
    setPillWidth(textWidth(text) + 78);
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

  function reset() {
    jobs.clear();
    hudHide();
    card.classList.remove("shown");
    pill.classList.remove("shown");
    cursor.classList.remove("idle");
    cam = { x: W / 2, y: H / 2, s: 1 };
    applyCam();
  }

  /* ---------- Scenes ---------- */

  /** Chat reply by voice; `translate` routes it through the polishing model. */
  async function chatScene(copy, translate) {
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
    placeCursor(960, 230);

    await wait(500);
    await move(at(composer, 0.3, 0.5), 950);
    await click();
    composer.classList.add("focused");
    placeholder.replaceWith(typed, caret());
    await wait(150);
    idle();
    await zoom({ ...mid(at(composer, 0.5, 0.2), at(pill, 0.5, 0.5)), s: 1.7 });

    const [key] = hudShow([copy.key], copy.hold);
    key.classList.add("down");
    pillListen();
    await wait(650);
    for (const partial of copy.partials) {
      pillDraft(partial);
      await wait(460);
    }
    await wait(350);
    key.classList.remove("down");
    await wait(160);
    hudHide();

    if (translate) {
      pillPolish();
      await wait(1250);
    }
    typed.textContent = copy.final;
    typed.className = "reveal";
    pillDone(C.inserted);
    await wait(1100);

    await press([C.keys.enter]);
    thread.append(message({ who: copy.me, text: copy.final, me: true }, true));
    typed.textContent = "";
    pillHide();
    await wait(250);
    hudHide();
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  async function polishScene() {
    const copy = C.polish;
    const content = h("div.content");
    mount(
      menubar(copy.app),
      appWindow(
        copy.app,
        h(
          "div.mail",
          null,
          h("div.field", null, h("span.label", { text: copy.to }), h("span.chip", { text: copy.toValue })),
          h("div.field", null, h("span.label", { text: copy.subject }), h("span", { text: copy.subjectValue })),
          content,
        ),
      ),
    );
    placeCursor(980, 200);
    await wait(500);
    await move(at(content, 0.18, 0.12), 950);
    await click();
    content.append(caret());
    await wait(150);
    idle();
    await zoom({ x: W / 2, y: H, s: 1.25 });

    const [key] = hudShow([copy.key], copy.hold);
    key.classList.add("down");
    pillListen();
    await wait(650);
    for (const partial of copy.partials) {
      pillDraft(partial);
      await wait(520);
    }
    await wait(350);
    key.classList.remove("down");
    await wait(160);
    hudHide();
    pillPolish();
    await wait(1300);

    content.replaceChildren(h("span.reveal", { text: copy.final }), caret());
    pillDone(C.inserted);
    await wait(1100);
    pillHide();
    await zoom({ s: 1 }, 1200);

    const parts = copy.heard.map(([text, cut]) => (cut ? h("s", { text }) : h("span", { text })));
    card.replaceChildren(
      h("span.k", { text: copy.heardLabel }),
      h("span", null, parts),
      h("span.chip-scene", { text: copy.scene }),
    );
    card.classList.add("shown");
    await wait(500);
    for (const cut of card.querySelectorAll("s")) {
      cut.classList.add("on");
      await wait(320);
    }
    await hold();
    await wait(1400);
    await wait(1800);
  }

  /* Language-neutral Pinyin data: keys, preedit and the native candidate page. */
  const PINYIN = [
    { key: "s", pre: "s", cands: ["是", "说", "上", "所", "三"] },
    { key: "h", pre: "sh", cands: ["是", "说", "上", "时", "事"] },
    { key: "i", pre: "shi", cands: ["是", "时", "事", "使", "十"] },
    { key: "y", pre: "shi y", cands: ["是一", "事业", "实验", "使用", "时有"] },
    { key: "a", pre: "shi ya", cands: ["实验", "誓言", "试验", "是呀", "食盐"] },
    { key: "n", pre: "shi yan", cands: ["实验", "誓言", "试验", "食盐", "失言"] },
  ];
  const PROMOTED = "誓言";

  function candidateBar(words) {
    return h(
      "span.candidates",
      null,
      words.map((word, i) => h(i === 0 ? "span.candidate.pick" : "span.candidate", null, h("small", { text: String(i + 1) }), word)),
    );
  }

  async function keyboardScene() {
    const copy = C.keyboard;
    const committed = document.createTextNode(copy.before);
    const barAnchor = h("span.anchor");
    const preedit = h("span.preedit");
    const caretEl = caret();
    const badgeAnchor = h("span.anchor");
    const line = h("p.live", null, committed, barAnchor, preedit, caretEl, badgeAnchor);
    caretEl.style.display = "none";
    mount(menubar(copy.app), appWindow(copy.app, h("div.doc", null, h("h2", { text: copy.title }), h("p", { text: copy.intro }), line)));
    placeCursor(900, 520);

    await wait(500);
    await move(at(badgeAnchor, 0, 0.6), 900);
    await click();
    caretEl.style.display = "";
    await wait(150);
    idle();
    await zoom(fromLeft(line, 1.8, 70));

    let bar = null;
    for (const step of PINYIN) {
      await press([step.key], null, 110);
      preedit.textContent = step.pre;
      const next = candidateBar(step.cands);
      if (bar) next.style.animation = "none";
      if (bar) bar.replaceWith(next);
      else barAnchor.append(next);
      bar = next;
      await wait(260);
    }
    hudHide();

    // The settled local ranking pass promotes a candidate that fits the text
    // before the caret; the native page rebuilds from the same candidates.
    await wait(380);
    const items = [...bar.children];
    const target = items.find((item) => item.lastChild.textContent === PROMOTED);
    const before = new Map(items.map((item) => [item, item.getBoundingClientRect().left]));
    bar.prepend(target);
    const ordered = [...bar.children];
    const k = desk.getBoundingClientRect().width / W / cam.s;
    ordered.forEach((item, i) => {
      item.querySelector("small").textContent = String(i + 1);
      item.classList.toggle("pick", i === 0);
      const dx = (before.get(item) - item.getBoundingClientRect().left) / (k * cam.s);
      item.style.transition = "none";
      item.style.transform = `translateX(${dx}px)`;
    });
    bar.getBoundingClientRect();
    ordered.forEach((item) => {
      item.style.transition = "";
      item.style.transform = "";
    });
    hudShow([], copy.rank);
    await wait(1300);
    hudHide();

    await press([C.keys.space]);
    committed.textContent += PROMOTED;
    preedit.textContent = "";
    bar.remove();
    await wait(600);

    await press([C.keys.shift]);
    const badge = h("span.badge", null, h("span.from", { text: copy.chinese }), h("span.to", { text: copy.english }));
    badgeAnchor.append(badge);
    await wait(40);
    badge.classList.add("flip");
    await wait(1200);
    hudHide();
    badge.remove();
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(1800);
  }

  async function clipboardScene() {
    const copy = C.clipboard;
    const typed = h("span");
    const caretEl = caret();
    const anchor = h("span.anchor");
    const line = h("p.live", null, copy.before, typed, caretEl, anchor);
    caretEl.style.display = "none";
    mount(
      menubar(copy.app),
      appWindow(copy.app, h("div.doc", null, h("h2", { text: copy.title }), copy.lines.map((text) => h("p", { text })), line)),
    );
    placeCursor(940, 560);

    await wait(500);
    await move(at(anchor, 0, 0.6), 900);
    await click();
    caretEl.style.display = "";
    await wait(150);
    idle();
    await zoom(fromLeft(line, 1.6, 120));

    await press(C.keys.clipboard, null, 220);
    await wait(200);
    hudHide();

    const icons = { text: ICON.text, file: ICON.file, link: ICON.link };
    const rows = copy.rows.map((row) =>
      h(
        "div.row",
        null,
        row.kind === "image" ? h("span.thumb") : svg(icons[row.kind], 15),
        h("span.label", { text: row.text }),
        row.tag ? h("span.tag", { text: row.tag }) : null,
        h("span.ret", null, svg(ICON.ret, 15)),
      ),
    );
    const query = h("span.q");
    const searchIcon = svg(ICON.search, 15);
    const header = h("header", null, h("span.q", { text: copy.label }), searchIcon);
    const picker = h("div.picker", null, header, rows);
    anchor.append(picker);
    const select = (row) => rows.forEach((r) => r.classList.toggle("sel", r === row));
    select(rows[0]);
    await wait(700);

    const image = rows[copy.rows.findIndex((row) => row.kind === "image")];
    await move(at(image, 0.3, 0.5), 700);
    select(image);
    await wait(450);
    const flyout = h("div.flyout", null, h("span.thumb"));
    flyout.style.top = `${image.offsetTop}px`;
    picker.append(flyout);
    await wait(1300);

    await move(at(searchIcon, 0.5, 0.5), 650);
    flyout.remove();
    await click();
    header.replaceChildren(svg(ICON.search, 15), query, caret());
    for (const letter of copy.query) {
      await press([letter], null, 100);
      query.textContent += letter;
      const q = query.textContent.toLowerCase();
      rows.forEach((row) => {
        row.style.display = row.textContent.toLowerCase().includes(q) ? "" : "none";
      });
      select(rows.find((row) => row.style.display === ""));
      await wait(180);
    }
    await wait(500);
    await press([C.keys.enter]);
    const chosen = rows.find((row) => row.style.display === "");
    picker.remove();
    typed.textContent = chosen.textContent;
    typed.className = "reveal";
    await wait(400);
    hudHide();
    await move({ x: pos.x + 120, y: pos.y + 140 }, 700);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(1800);
  }

  const scenes = {
    dictation: () => chatScene(C.dictation, false),
    translate: () => chatScene(C.translate, true),
    polish: polishScene,
    keyboard: keyboardScene,
    clipboard: clipboardScene,
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
