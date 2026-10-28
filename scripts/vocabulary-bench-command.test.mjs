import test from "node:test";
import assert from "node:assert/strict";
import * as fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { createHash } from "node:crypto";
import { evaluate, percentile, verify } from "./vocabulary-bench-command.mjs";

const input = [{ id: "alias", source: "heard", previous: "", entries: [{ From: "heard", To: "Brand" }] }];
const expected = [{ id: "alias", corrected: "Brand", terms: ["Brand"] }];
const row = terms => ({ id: "alias", source: "heard", corrected: "Brand", terms, latencyNs: Array.from({ length: 100 }, (_, i) => (i + 1) * 1000) });
const arms = () => ({ baseline: [row(["heard → Brand", "other → Brand"])], candidate: [row(["Brand"])] });

test("canonical recall, raw pair exposure and duplicated budget remain distinct", () => {
  const report = evaluate(input, expected, arms());
  assert.equal(report.candidateAcceptancePassed, true);
  assert.equal(report.metrics.baseline.exactTerms, 0);
  assert.equal(report.metrics.baseline.canonicalRecall, 1);
  assert.equal(report.metrics.baseline.aliasExposingTermSlots, 2);
  assert.equal(report.metrics.baseline.duplicateCanonicalBudgetSlots, 1);
  assert.equal(report.metrics.candidate.aliasExposingTermSlots, 0);
  assert.deepEqual(report.metrics.candidate.latencyUs, { p50: 50, p95: 95, p99: 99 });
  assert.equal(percentile([5, 1, 3, 2, 4], .5), 3);
});

test("missing, duplicate, reordered and malformed observations cannot report complete", () => {
  for (const bad of [[], [row(["Brand"]), row(["Brand"])], [{ ...row(["Brand"]), id: "other" }], [{ ...row(["Brand"]), latencyNs: [1] }]]) {
    assert.throws(() => evaluate(input, expected, { ...arms(), candidate: bad }));
  }
  assert.throws(() => evaluate(input, [], arms()));
});

test("incorrect corrections and terms fail acceptance without dropping observations", () => {
  const candidate = [{ ...row(["Other"]), corrected: "Other" }];
  const report = evaluate(input, expected, { ...arms(), candidate });
  assert.equal(report.candidateAcceptancePassed, false);
  assert.equal(report.metrics.candidate.canonicalRecall, 0);
  assert.equal(report.metrics.candidate.unexpectedCanonicalTerms, 1);
  assert.equal(report.cases.candidate.length, 1);
});

test("archive verification retains failed status and rejects altered or extra bytes", async t => {
  const directory = await fs.mkdtemp(path.join(os.tmpdir(), "arion-vocabulary-archive-test-"));
  t.after(() => fs.rm(directory, { recursive: true, force: true }));
  const bytes = JSON.stringify({ status: "failed", error: "controlled failure" });
  await fs.writeFile(path.join(directory, "status.json"), bytes);
  await fs.writeFile(path.join(directory, "manifest.json"), JSON.stringify({ schema: "vocabulary-recall-run-v1", files: {
    "status.json": createHash("sha256").update(bytes).digest("hex"),
  } }));
  await verify(directory);
  await fs.writeFile(path.join(directory, "status.json"), JSON.stringify({ status: "completed" }));
  await assert.rejects(verify(directory), /SHA-256 mismatch/);
  await fs.writeFile(path.join(directory, "status.json"), bytes);
  await fs.writeFile(path.join(directory, "unlisted.json"), "{}");
  await assert.rejects(verify(directory), /inventory/);
});
