import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import crypto from "node:crypto";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import test from "node:test";
import assert from "node:assert/strict";
import { scoreText } from "../live/score-text.mjs";
import { scoreAudio } from "../live/score-audio.mjs";

const sample = (id, sourceText, extra = {}) => ({
  id, language: "zh", sourceText, references: [extra.reference ?? sourceText], requiredStrings: extra.required ?? [],
  forbiddenStrings: extra.forbidden ?? [], protected: [], mode: extra.mode ?? "clean", scenario: "chat", approvedMemory: [],
});
const corpus = {
  samples: [
    sample("repair", "周三开会，不对，周四开会。", { reference: "周四开会。", required: ["周四"], forbidden: ["周三"] }),
    sample("copy", "下周三交稿。", { required: ["下周三"] }),
  ],
};

test("text scoring separates proposals, delivered text and worsening", () => {
  const rows = [
    { caseId: "repair", verdict: "accepted", proposal: "周四开会。", text: "周四开会。", generationMs: 300, live: [{ verdict: "accepted", totalMs: 120 }] },
    { caseId: "copy", verdict: "content_dropped", proposal: "交稿。", text: "下周三交稿。", generationMs: 200, live: [] },
  ];
  const s = scoreText(corpus, rows);
  assert.equal(s.sourcePass, 1);
  assert.equal(s.proposalPass, 1);
  assert.equal(s.deliveredPass, 2);
  assert.equal(s.worsened, 0);
  assert.deepEqual(s.rejected, { content_dropped: 1 });
  assert.equal(s.liveEdits, 1);
});

test("text scoring breaks results down by expected scene", () => {
  const scenes = {
    samples: [
      { ...sample("mail", "张总你好，合同看完了。", { required: ["\n\n"] }), scenario: "email" },
      sample("chat", "我七点到。", { required: ["七点"] }),
    ],
  };
  const rows = [
    { caseId: "mail", scene: "email", verdict: "accepted", proposal: "张总你好：\n\n合同看完了。", text: "张总你好：\n\n合同看完了。", generationMs: 300 },
    { caseId: "chat", scene: "email", verdict: "accepted", proposal: "我七点到。", text: "我七点到。", generationMs: 200 },
  ];
  const s = scoreText(scenes, rows);
  assert.equal(s.sceneAgreement, 1);
  assert.deepEqual(s.byScene.email, { cases: 1, sourcePass: 0, deliveredPass: 1, worsened: 0, sceneAgreement: 1 });
  assert.deepEqual(s.byScene.chat, { cases: 1, sourcePass: 1, deliveredPass: 1, worsened: 0, sceneAgreement: 0 });
});

test("text scoring rejects incomplete coverage", () => {
  assert.throws(() => scoreText(corpus, [{ caseId: "repair", verdict: "accepted", text: "x" }]));
});

test("audio scoring reports acoustic error, latency and previews before stop", () => {
  const manifest = { samples: [
    { id: "repair", segments: 3, speech: "周三开会|不对|周四开会" },
    { id: "copy", segments: 1, speech: "下周三交稿" },
  ] };
  const rows = [
    { caseId: "repair", rawText: "周三开会。不对。周四开会。", sourceText: "周三开会。不对。周四开会。", finalText: "周四开会。", rewrite: "accepted",
      stopToFinalMs: 600, firstPartialMs: 900, firstPreviewMs: 2000, speechEndMs: 5000, previewEvents: 3 },
    { caseId: "copy", rawText: "下周三交搞。", sourceText: "下周三交搞。", finalText: "下周三交搞。", rewrite: "accepted",
      stopToFinalMs: 400, firstPartialMs: 700, firstPreviewMs: null, speechEndMs: 1500, previewEvents: 0 },
  ];
  const s = scoreAudio(manifest, corpus, rows);
  assert.equal(s.finalPass, 2);
  assert.equal(s.improved, 1);
  assert.equal(s.multiSegment, 1);
  assert.equal(s.previewBeforeStop, 1);
  assert.equal(s.asrErrorRate["zh:cer"].errors, 1);
  assert.equal(s.stopToFinalMs.p50, 500);
});

test("cleanup and multilingual constraint counts remain separate from error rates", () => {
  const multilingual = { samples: [
    { ...sample("fr", "Euh, le rapport est prêt.", { required: ["rapport"], forbidden: ["Euh"] }), language: "fr" },
    { ...sample("en", "I might need it.", { required: ["might"] }), language: "en" },
  ] };
  const rows = [
    { caseId: "fr", source: "Euh, le rapport est prêt.", cleanup: { version: "speech-spans-v1", text: "le rapport est prêt." }, verdict: "accepted", proposal: "Le rapport est prêt.", text: "Le rapport est prêt.", generationMs: 100 },
    { caseId: "en", source: "I might need it.", verdict: "uncertainty_dropped", proposal: "I need it.", text: "I might need it.", generationMs: 200 },
  ];
  const s = scoreText(multilingual, rows);
  assert.equal(s.sourcePass, 1);
  assert.equal(s.cleanupPass, 2);
  assert.equal(s.cleanupChanged, 1);
  assert.equal(s.deliveredPass, 2);
  assert.equal(s.referenceEditRatioMean, 0);
  assert.deepEqual(s.byLanguage.fr, { cases: 1, sourcePass: 0, cleanupPass: 1, proposalPass: 1, deliveredPass: 1, worsened: 0 });
});

test("live report rejects checksum changes instead of publishing altered observations", async () => {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), "arion-live-report-"));
  try {
    const corpusPath = path.join(dir, "corpus.json");
    const rowsPath = path.join(dir, "rows.jsonl");
    const configPath = path.join(dir, "runs.json");
    const out = path.join(dir, "summary.json");
    const corpusBytes = JSON.stringify({ samples: [sample("copy", "hello")] });
    const rowsBytes = JSON.stringify({ caseId: "copy", source: "hello", proposal: "hello", text: "hello", verdict: "accepted", generationMs: 1 }) + "\n";
    const hash = (s) => crypto.createHash("sha256").update(s).digest("hex");
    await fs.writeFile(corpusPath, corpusBytes);
    await fs.writeFile(rowsPath, rowsBytes);
    await fs.writeFile(configPath, JSON.stringify({ runs: [{ id: "fixture-only", kind: "text", corpus: corpusPath, results: rowsPath, corpusSha256: hash(corpusBytes), resultsSha256: hash(rowsBytes) }] }));
    const run = () => spawnSync(process.execPath, [fileURLToPath(new URL("../live/report.mjs", import.meta.url)), configPath, out], { encoding: "utf8" });
    await fs.writeFile(rowsPath, rowsBytes.replaceAll("hello", "other"));
    assert.match(run().stderr, /results checksum changed/);
    await assert.rejects(fs.access(out));
    await fs.writeFile(rowsPath, rowsBytes);
    await fs.writeFile(corpusPath, corpusBytes + " ");
    assert.match(run().stderr, /corpus checksum changed/);
    await fs.writeFile(corpusPath, corpusBytes);
    assert.equal(run().status, 0);
    assert.notEqual(run().status, 0);
  } finally { await fs.rm(dir, { recursive: true, force: true }); }
});

test("audio reports distinguish legacy unknown filler policy and reject mixed treatments", () => {
  const manifest = { samples: [{ id: "copy", segments: 1, speech: "下周三交稿" }] };
  const text = { samples: [corpus.samples[1]] };
  const row = { caseId: "copy", rawText: "下周三交稿", sourceText: "下周三交稿", finalText: "下周三交稿", rewrite: "accepted", stopToFinalMs: 1, firstPartialMs: 1, firstPreviewMs: null, speechEndMs: 1, previewEvents: 0 };
  assert.equal(scoreAudio(manifest, text, [row]).removeFillers, null);
  assert.equal(scoreAudio(manifest, text, [{ ...row, removeFillers: true }]).removeFillers, true);
  const extended = { samples: [...manifest.samples, { ...manifest.samples[0], id: "other" }] };
  assert.throws(() => scoreAudio(extended, text, [{ ...row, removeFillers: true }, { ...row, caseId: "other", removeFillers: false }]), /Mixed filler policies/);
});
