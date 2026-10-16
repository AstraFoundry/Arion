// Report actual text-model outcomes; these fixtures do not measure OCR or ASR.
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { normalize, errorCounts, distribution } from "../../../tools/evaluation/src/metrics.mjs";

const [fixtures, directory, output] = process.argv.slice(2);
const read = (p) => JSON.parse(fs.readFileSync(p,"utf8"));
const sha = (p) => crypto.createHash("sha256").update(fs.readFileSync(p)).digest("hex");
const lines = (name) => fs.readFileSync(path.join(directory,name),"utf8").trim().split("\n").filter(Boolean).map(JSON.parse);
const run = read(path.join(directory,"run.json"));
if (run.status !== "complete") throw new Error("Incomplete context run");
if (sha(path.join(fixtures,"freeze.json")) !== run.freezeSha256) throw new Error("Freeze changed");
for (const [name,m] of Object.entries(read(path.join(fixtures,"freeze.json")).files)) if (sha(path.join(fixtures,name)) !== m.sha256) throw new Error("Frozen input changed");
for (const [name,h] of Object.entries(run.outputHashes)) if (sha(path.join(directory,name)) !== h) throw new Error("Output changed");
const memory = read(path.join(directory,"frozen-memory.json"));
const teaching = lines("teaching.jsonl");
const feedback = read(path.join(fixtures,run.split,"feedback.json")).events;
if (read(path.join(directory,"initial-memory.json")).afterCount !== 0 || teaching.length !== feedback.length) throw new Error("Invalid cold-start feedback");
let count = 0;
for (const r of teaching) {
  const e = feedback.find((x) => x.id === r.id);
  if (!e || JSON.stringify(e) !== JSON.stringify(r.feedback) || r.persistence.beforeCount !== count) throw new Error("Unapproved feedback source");
  count = r.persistence.afterCount;
}
if (JSON.stringify(memory.entries) !== JSON.stringify(teaching.at(-1).persistence.entries) || memory.afterCount !== count) throw new Error("Persistence/reopen mismatch");
const inputs = read(path.join(fixtures,run.split,"inputs.json")).samples;
const labels = new Map(read(path.join(fixtures,run.split,"evaluation.json")).samples.map((x) => [x.id,x]));
const rows = lines("results.jsonl"), seen = new Set(), seenResponses = new Set(), scored = [];
for (const r of rows) {
  const key = r.caseId + "/" + r.arm, input = inputs.find((x) => x.id === r.caseId), label = labels.get(r.caseId);
  if (seen.has(key) || !input || !label || !run.arms.includes(r.arm) || JSON.stringify(input) !== JSON.stringify(r.input)) throw new Error("Invalid paired context case");
  seen.add(key);
  if (!r.response.ok || !r.response.tokenizerParity || seenResponses.has(r.response.id)) throw new Error("Invalid actual generation");
  seenResponses.add(r.response.id);
  const delivered = r.validation.verdict === "accepted" ? r.response.text : r.rendered.source;
  if (delivered !== r.delivered) throw new Error("Delivered output differs from recorded validator decision");
  if (r.selected.some((h) => !memory.entries.some((e) => e.id === h.id && e.to === h.target))) throw new Error("Unstored correction used");
  const required = [...input.text.matchAll(/\d+(?:\.\d+)?|[一二三四五六七八九十]+点|不想|不要|不能|\bnot\b|\bdon't\b|codex/giu)].map((m) => m[0].toLowerCase());
  const questionViolation = /[?？]/u.test(input.text) && !/[?？吗]/u.test(delivered);
  const protectedViolation = required.some((t) => !delivered.toLowerCase().includes(t));
  scored.push({caseId:r.caseId,arm:r.arm,...label,source:input.text,screen:input.context.Screen,
    modelSource:r.rendered.source,modelContext:r.rendered.modelContext,modelTerms:r.rendered.terms,
    rawText:r.response.text,verdict:r.validation.verdict,delivered,
    exact:normalize(delivered)===normalize(label.reference),rawExact:normalize(r.response.text)===normalize(label.reference),
    changed:normalize(delivered)!==normalize(input.text),...errorCounts(label.reference,delivered,label.unit),
    questionViolation,protectedViolation,truncated:r.response.truncated,generationMs:r.response.generationMs});
}
if (rows.length !== inputs.length*run.arms.length || rows.length !== run.observations || rows.length !== run.actualModelCalls) throw new Error("Incomplete context comparison");
const summary={status:"complete",split:run.split,evidence:run.evidence,cases:inputs.length,observations:rows.length,actualModelCalls:rows.length,warmupCalls:run.warmups,
  learning:{initialEntries:0,feedbackCases:feedback.length,storedEntries:memory.afterCount,reopenMatches:true},arms:{},perCase:scored,semanticReview:"single_author_exact_reference_proxy"};
for (const arm of run.arms) {
  const all=scored.filter((r)=>r.arm===arm),controls=all.filter((r)=>r.kind==="preserve_control"),correct=all.filter((r)=>r.correctInput);
  summary.arms[arm]={exactMatches:all.filter((r)=>r.exact).length,cases:all.length,rawExactMatches:all.filter((r)=>r.rawExact).length,
    branchMatches:all.filter((r)=>r.kind==="meaning_branch"&&r.exact).length,branchCases:all.filter((r)=>r.kind==="meaning_branch").length,
    controlFailures:controls.filter((r)=>!r.exact).length,controlCases:controls.length,
    correctInputRegressions:correct.filter((r)=>!r.exact).length,correctInputCases:correct.length,
    questionViolations:all.filter((r)=>r.questionViolation).length,protectedViolations:all.filter((r)=>r.protectedViolation).length,
    validatorRejections:all.filter((r)=>r.verdict!=="accepted").length,truncations:all.filter((r)=>r.truncated).length,
    generationMs:distribution(all.map((r)=>r.generationMs)),lexicalErrors:all.reduce((n,r)=>n+r.errors,0)};
}
fs.writeFileSync(output,JSON.stringify(summary,null,2)+"\n",{flag:"wx"});
console.log(JSON.stringify(summary.arms,null,2));
