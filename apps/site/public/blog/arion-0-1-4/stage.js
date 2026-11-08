/*
 * Mock recordings for the "Arion 0.1.2 to 0.1.4" post, drawn in the manner of a
 * screen-recording editor: a rounded recording on a wallpaper, an eased
 * camera that zooms to where the action is, a smoothed cursor with click
 * ripples, and a keystroke overlay. Every scene is an illustration; nothing
 * is recorded and no model runs.
 *
 * stage.<locale>.html sets window.ARION_COPY and loads this file; the URL
 * hash picks the scene (#hotwords, #compatible, #deepseek, #feedback). The
 * page plays only while it is in view and, under reduced motion, shows one
 * finished frame. The engine (clock, camera, cursor, keystroke overlay and
 * pill) follows the 0.1.1 post's stage.js.
 */
(() => {
  "use strict";

  const C = window.ARION_COPY;
  const sceneName = location.hash.slice(1) || "hotwords";
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
    card.classList.remove("shown", "top");
    pill.classList.remove("shown", "quiet");
    cursor.classList.remove("idle");
    cam = { x: W / 2, y: H / 2, s: 1 };
    applyCam();
  }

  /* ---------- Scenes ---------- */

  /** Types `text` into `el` one character at a time. */
  async function typeInto(el, text, ms = 38) {
    for (const letter of text) {
      el.textContent += letter;
      await wait(ms);
    }
  }

  /* 0.1.2: a quiet window is not sent, so the hotword list cannot echo back. */
  async function hotwordScene() {
    const copy = C.hotwords;
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
    await zoom({ ...mid(at(composer, 0.5, 0.2), at(pill, 0.5, 0.5)), s: 1.6 });

    const [key] = hudShow([copy.key], copy.hold);
    key.classList.add("down");
    pillListen();
    pill.classList.add("quiet");
    card.classList.add("top");
    card.replaceChildren(
      h("span.k", { text: copy.vocabLabel }),
      h("span.terms", null, copy.vocab.map((term) => h("span.term", { text: term }))),
      h("span.chip-scene", { text: copy.quiet }),
    );
    await wait(300);
    card.classList.add("shown");
    await wait(2200);
    card.classList.remove("shown");
    pill.classList.remove("quiet");
    for (const partial of copy.partials) {
      pillDraft(partial);
      await wait(480);
    }
    await wait(350);
    key.classList.remove("down");
    await wait(160);
    hudHide();
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

  /* ---------- Settings › Speech & Polish › Cloud model ---------- */

  function settingsWindow() {
    const page = h("div.spage");
    const nav = C.settings.nav.map((name, i) => h(i === C.settings.navOn ? "span.on" : "span", { text: name }));
    return { win: appWindow(C.settings.title, h("div.snav", null, nav), page), page };
  }

  function providerMark(provider) {
    return h("span.pmark", { text: provider.mark, style: `--c: ${provider.color}` });
  }

  /** The provider list; `connected` puts that provider and its model on top. */
  function providerList(connected, model) {
    const S = C.settings;
    const rows = new Map();
    const kids = [h("div.shead", null, h("span.back", { text: "‹" }), h("span", { text: S.cloudTitle }))];
    let select = null;
    let inUse = null;
    if (connected) {
      select = h("span.input.select", null, h("span", { text: model }));
      inUse = h(
        "div.group",
        null,
        h(
          "div.prow.tall",
          null,
          providerMark(connected),
          h("span.pname", null, connected.name, h("span.sub", { text: S.keyLabel })),
          h("span.btn", { text: S.disconnect }),
        ),
        h("div.prow.tall", null, h("span.pname", { text: S.modelLabel }), select),
      );
      kids.push(h("div.glabel", { text: S.inUse }), inUse);
    }
    const group = h("div.group");
    for (const provider of S.providers) {
      const row = h("div.prow", null, providerMark(provider), h("span.pname", { text: provider.name }), h("span.chev", { text: "›" }));
      rows.set(provider.id, row);
      group.append(row);
    }
    kids.push(h("div.glabel", { text: S.apiKeys }), group);
    return { node: h("div", null, kids), rows, select, inUse };
  }

  /** A provider's key page; compatible providers also ask for a Base URL. */
  function providerForm(provider) {
    const S = C.settings;
    const placeholder = (text) => h("span.placeholder", { text });
    const base = provider.compatible ? h("span.input", null, placeholder(S.basePlaceholder)) : null;
    const key = h("span.input", null, placeholder(S.keyPlaceholder));
    const connect = h("span.btn.primary", { text: S.connect });
    const node = h(
      "div",
      null,
      h("div.shead", null, h("span.back", { text: "‹" }), h("span", { text: provider.name })),
      h(
        "div.group.form",
        null,
        providerMark(provider),
        h("div.fcol", null, base ? [h("div.flabel", { text: S.baseLabel }), base] : null, h("div.frow", null, key, connect)),
      ),
    );
    return { node, base, key, connect };
  }

  async function focusField(field) {
    await move(at(field, 0.2, 0.5), 650);
    await click();
    field.classList.add("focused");
    field.replaceChildren(h("span"), caret());
    return field.firstChild;
  }

  /** Pick a provider, enter its details, connect, then choose a model. */
  async function providerScene(copy) {
    const provider = C.settings.providers.find((p) => p.id === copy.provider);
    const { win, page } = settingsWindow();
    let list = providerList(null);
    page.append(list.node);
    mount(menubar(C.settings.app), win);
    placeCursor(1000, 640);

    await wait(500);
    const row = list.rows.get(provider.id);
    await move(at(row, 0.3, 0.5), 950);
    row.classList.add("hover");
    await wait(200);
    await click();
    await wait(120);
    const form = providerForm(provider);
    page.replaceChildren(form.node);
    await wait(320);
    await zoom({ ...at(form.node.lastChild, 0.5, 0.6), s: 1.55 });

    if (form.base) {
      const text = await focusField(form.base);
      idle();
      await typeInto(text, copy.baseURL);
      form.base.classList.remove("focused");
      await wait(250);
    }
    await focusField(form.key);
    idle();
    await press(C.keys.paste, null, 200);
    form.key.firstChild.replaceWith(h("span.dots", { text: "•".repeat(22) }));
    await wait(450);
    hudHide();
    await move(at(form.connect, 0.5, 0.5), 600);
    await click();
    form.key.classList.remove("focused");
    form.connect.classList.add("busy");
    form.connect.replaceChildren(h("span.orbit", null, h("i"), h("i"), h("i")));
    if (copy.note) {
      card.classList.add("top");
      card.replaceChildren(h("span.k", { text: copy.noteLabel }), h("code", { text: copy.note }), h("span.chip-scene.chip-ok", { text: copy.status }));
      card.classList.add("shown");
    }
    await wait(1500);

    list = providerList(provider, copy.models[0]);
    page.replaceChildren(list.node);
    await wait(350);
    card.classList.remove("shown");
    await zoom({ ...at(list.inUse, 0.5, 0.9), s: 1.4 }, 1000);
    await move(at(list.select, 0.6, 0.5), 700);
    await click();
    const options = copy.models.map((name, i) => h(i === 0 ? "span.sel" : "span", { text: name }));
    const menu = h("div.menu", null, options);
    list.select.append(menu);
    await wait(500);
    const choice = options[copy.pick];
    await move(at(choice, 0.4, 0.5), 500);
    options.forEach((option) => option.classList.toggle("sel", option === choice));
    await wait(250);
    await click();
    menu.remove();
    list.select.firstChild.textContent = copy.models[copy.pick];
    list.select.firstChild.className = "reveal";
    await wait(700);
    await move({ x: pos.x + 90, y: pos.y + 120 }, 600);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2000);
  }

  /* 0.1.4: from the menu bar panel's ? to an issue filed by a coding agent. */
  async function feedbackScene() {
    const copy = C.feedback;
    const bar = menubar(copy.app);
    const mark = bar.querySelector(".mark");
    const ask = h("div.ask", null, h("span.placeholder", { text: copy.ask }));
    const convo = h("div.convo");
    const agent = appWindow(copy.agent, h("div.chat", null, convo, ask));
    agent.classList.add("agent");
    mount(bar, agent);
    placeCursor(700, 520);

    await wait(500);
    await move(at(mark, 0.5, 0.6), 1000);
    await click();
    mark.classList.add("open");
    const help = h("span.help", { text: "?" });
    const panel = h(
      "div.mpanel",
      null,
      h("div.top", null, h("span.logo"), h("span.name", { text: "Arion" }), h("span.ready", { text: copy.ready }), help),
      h("div.start", { text: copy.start }),
      h("div.foot", null, h("span", { text: copy.settings }), h("span", { text: copy.quit })),
    );
    const markAt = at(mark, 0.5, 1);
    panel.style.left = `${markAt.x - 260}px`;
    desk.insertBefore(panel, pill);
    await wait(200);
    await zoom({ ...at(panel, 0.6, 0.5), s: 1.7 }, 950);
    await move(at(help, 0.5, 0.5), 700);
    await wait(250);
    await click();
    panel.remove();
    mark.classList.remove("open");

    const copyButton = h("span.btn.primary", null, svg(ICON.copy, 15), h("span", { text: copy.copy }));
    const win = h(
      "div.fwin",
      null,
      h("div.titlebar", null, h("span.lights", null, h("i"), h("i"), h("i")), h("span.title", { text: copy.windowTitle })),
      h(
        "div.content",
        null,
        h("h3", { text: copy.title }),
        h("p", { text: copy.body }),
        h("div.prompt", null, h("span", { text: copy.prompt }), copyButton),
        h("span.link", { text: copy.yourself }),
      ),
    );
    desk.insertBefore(win, pill);
    await zoom({ ...at(win, 0.5, 0.55), s: 1.35 }, 1000);
    await move(at(copyButton, 0.5, 0.5), 800);
    await wait(200);
    await click();
    copyButton.replaceChildren(svg(ICON.check, 15), h("span", { text: copy.copied }));
    await wait(900);

    await zoom({ s: 1 }, 900);
    await move(at(ask, 0.3, 0.5), 900);
    await click();
    agent.classList.add("front");
    ask.classList.add("focused");
    ask.replaceChildren(caret());
    await zoom({ ...at(agent, 0.5, 0.62), s: 1.4 }, 1000);
    idle();
    await press(C.keys.paste, null, 200);
    ask.replaceChildren(h("span.reveal", { text: copy.prompt }));
    await wait(700);
    await press([C.keys.enter]);
    hudHide();
    ask.classList.remove("focused");
    ask.replaceChildren(h("span.placeholder", { text: copy.ask }));
    convo.append(h("div.mine", { text: copy.prompt }));
    await wait(500);
    const typing = h("div.typing", null, h("i"), h("i"), h("i"));
    convo.append(typing);
    await wait(1200);
    typing.replaceWith(h("div.reply", { text: copy.reply }));
    await wait(1300);
    convo.append(
      h(
        "div.issue",
        null,
        h("span.gh", { text: "#" }),
        h("div", null, h("span.opened", null, svg(ICON.check, 13), copy.opened), h("b", { text: copy.issue })),
      ),
    );
    await wait(1400);
    await zoom({ s: 1 }, 1200);
    await hold();
    await wait(2200);
  }

  const scenes = {
    hotwords: hotwordScene,
    compatible: () => providerScene(C.compatible),
    deepseek: () => providerScene(C.deepseek),
    feedback: feedbackScene,
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
