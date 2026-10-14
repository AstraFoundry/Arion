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
