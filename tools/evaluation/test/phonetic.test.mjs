import test from "node:test";
import assert from "node:assert/strict";
import { evidenceInput, promptFor, scoreRow, summarizeRows, requireCoverage } from "../phonetic/design.mjs";

const evidence = { asrText: "许洋", pinyin: "xu3 yang2", phones: "ɕ y a ŋ", segments: [{ tokens: ["许", "洋"], logScores: [-1, -0.1] }], reference: "NEVER SEND THIS", context: "NEVER SEND CONTEXT" };

test("chat role boundaries use actual newline bytes and copy examples return plain text", () => {
  const prompt = promptFor(evidence, "text", -0.5);
  assert.ok(prompt.startsWith("<|im_start|>system\n"));
  assert.ok(prompt.includes("<|im_end|>\n<|im_start|>user\n"));
  assert.ok(prompt.includes("<|im_start|>assistant\nLeave the silver folder unchanged.<|im_end|>"));
  assert.ok(prompt.endsWith("<|im_start|>assistant\n<think>\n\n</think>\n\n"));
  assert.ok(!prompt.includes("NEVER SEND"));
});

test("phonetic treatments expose exactly their permitted evidence", () => {
  assert.deepEqual(evidenceInput(evidence, "text", -0.5), { asr_text: "许洋" });
  assert.deepEqual(evidenceInput(evidence, "ipa_only", -0.5), { audio_derived_ipa: "ɕ y a ŋ" });
  const input = evidenceInput(evidence, "text_pinyin_scores", -0.5);
  assert.deepEqual(input.token_log_scores, [["许", -1, true], ["洋", -0.1, false]]);
  assert.equal(input.text_derived_pinyin, "xu3 yang2");
  assert.equal(input.reference, undefined);
  assert.equal(input.context, undefined);
  assert.equal(input.audio_derived_ipa, undefined);
});

test("scoring distinguishes recovery from corruption and pools corpus errors", () => {
  const sample = { reference: "徐洋", unit: "cer" };
  const recovered = scoreRow(sample, evidence, { text: "徐洋" });
  const corrupted = scoreRow(sample, { ...evidence, asrText: "徐洋" }, { text: "许阳" });
  assert.equal(recovered.improved, true);
  assert.equal(recovered.raw.errors, 1);
  assert.equal(corrupted.corrupted, true);
  const response = { generationMs: 1, truncated: false };
  const aggregate = summarizeRows([{ unit: "cer", metrics: recovered, response }, { unit: "cer", metrics: corrupted, response }]);
  assert.equal(aggregate.rates.cer.rate, 0.5);
  assert.equal(aggregate.corrupted, 1);
});

test("reports refuse missing, duplicate and invented treatment observations", () => {
  const samples = [{ id: "case" }];
  const raw = { caseId: "case", model: "raw", variant: "raw" };
  const row = { caseId: "case", model: "local", variant: "text" };
  assert.doesNotThrow(() => requireCoverage([raw, row], samples, ["local"], ["text"]));
  assert.throws(() => requireCoverage([raw], samples, ["local"], ["text"]), /Incomplete/);
  assert.throws(() => requireCoverage([raw, row, row], samples, ["local"], ["text"]), /duplicate/);
  assert.throws(() => requireCoverage([raw, { ...row, variant: "invented" }], samples, ["local"], ["text"]), /Unexpected/);
});
