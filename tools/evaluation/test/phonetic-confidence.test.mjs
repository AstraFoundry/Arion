import test from "node:test";
import assert from "node:assert/strict";
import { riskAuc, scoreConfidence } from "../phonetic/confidence.mjs";

test("error-risk AUC handles direction, ties and missing classes", () => {
  assert.equal(riskAuc([{ error: true, risk: 2 }, { error: false, risk: 1 }]), 1);
  assert.equal(riskAuc([{ error: true, risk: 1 }, { error: false, risk: 2 }]), 0);
  assert.equal(riskAuc([{ error: true, risk: 1 }, { error: false, risk: 1 }]), 0.5);
  assert.equal(riskAuc([{ error: true, risk: 1 }]), null);
});

test("confidence scoring excludes silence and counts missing scores explicitly", () => {
  const samples = ["wrong", "right", "missing", "silence"].map(id => ({ id, reference: id === "silence" ? "" : "hello", unit: "wer" }));
  const evidence = new Map([
    ["wrong", { asrText: "yellow", segments: [{ logScores: [-2] }] }],
    ["right", { asrText: "hello", segments: [{ logScores: [-0.1] }] }],
    ["missing", { asrText: "", segments: [] }],
  ]);
  const result = scoreConfidence(samples, evidence, -1);
  assert.equal(result.samples, 2);
  assert.equal(result.missingScores, 1);
  assert.equal(result.truePositive, 1);
  assert.equal(result.trueNegative, 1);
  assert.equal(result.auc, 1);
});
