import test from "node:test";
import assert from "node:assert/strict";
import { validatePreflight } from "../phonetic/preflight.mjs";

const config = { models: ["local"], variants: ["text", "ipa_only"] };
const meta = { kind: "development-output-contract-smoke", status: "complete", testCases: 0, promptControllerHash: "frozen" };
const ids = new Set(["dev-zh", "dev-en", "dev-mixed"]);
const cases = () => config.variants.flatMap(variant => ["zh", "en", "mixed"].map(language => ({
  caseId: `dev-${language}`, split: "dev", model: "local", variant, language,
  response: { text: "plain words", truncated: false },
})));

test("preflight separates missing phoneme capability from text output-contract failures", () => {
  const rows = cases();
  rows.find(row => row.variant === "ipa_only").response.truncated = true;
  assert.equal(validatePreflight(meta, rows, config, "frozen", ids).negativeControlFailures, 1);
  rows.find(row => row.variant === "text").response.text = '{"asr_text":"wrapped"}';
  assert.throws(() => validatePreflight(meta, rows, config, "frozen", ids), /output contract/);
});

test("preflight rejects test leakage, changed prompts and incomplete evidence", () => {
  const rows = cases();
  rows[0].split = "test";
  assert.throws(() => validatePreflight(meta, rows, config, "frozen", ids), /development/);
  assert.throws(() => validatePreflight(meta, cases(), config, "changed", ids), /exact prompt/);
  assert.throws(() => validatePreflight(meta, cases().slice(1), config, "frozen", ids), /Incomplete/);
});
