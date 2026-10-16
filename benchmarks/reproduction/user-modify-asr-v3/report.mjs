// Score the causal, checksum-verified teaching/persistence/recognition chain.
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { errorCounts, distribution } from "../../../tools/evaluation/src/metrics.mjs";

const [fixtures, directory, output] = process.argv.slice(2);
const read = (p) => JSON.parse(fs.readFileSync(p, "utf8"));
const sha = (p) => crypto.createHash("sha256").update(fs.readFileSync(p)).digest("hex");
const lines = (name) => fs.readFileSync(path.join(directory, name), "utf8").trim().split("\n").filter(Boolean).map(JSON.parse);
const fail = (message) => { throw new Error(message); };
const run = read(path.join(directory, "run.json"));
if (run.status !== "complete") fail("Incomplete run");
if (sha(path.join(fixtures, "freeze.json")) !== run.initialFreezeSha256) fail("Initial freeze changed");
const frozen = read(path.join(fixtures, "freeze.json"));
for (const [name, identity] of Object.entries(frozen.files)) {
  const p = path.join(fixtures, name);
  if (sha(p) !== identity.sha256 || fs.statSync(p).size !== identity.sizeBytes) fail("Frozen input changed: " + name);
}
for (const [name, hash] of Object.entries(run.inputHashes)) if (sha(path.join(fixtures, name)) !== hash) fail("Run input changed");
for (const [name, hash] of Object.entries(run.outputHashes)) if (sha(path.join(directory, name)) !== hash) fail("Output changed: " + name);

const split = path.join(fixtures, run.split);
const teachingInputs = read(path.join(split, "teaching.json")).events;
const future = read(path.join(split, "recognition.json")).samples;
const labels = read(path.join(split, "evaluation.json"));
const controls = read(path.join(split, "edit-controls.json")).events;
const initial = read(path.join(directory, "initial-memory.json"));
const snapshot = read(path.join(directory, "frozen-memory.json"));
const memory = snapshot.entries ?? [];
if (initial.afterCount !== 0) fail("Memory was not empty");
if (sha(path.join(directory, "frozen-memory.json")) !== run.frozenMemorySha256) fail("Memory snapshot changed");
if (snapshot.ciphertextSha256 !== run.frozenCiphertextSha256 || memory.length !== snapshot.afterCount) fail("Reloaded memory identity mismatch");
const cipherPath = path.join(directory, "vocabulary.enc");
if (run.frozenCiphertextSha256 && sha(cipherPath) !== run.frozenCiphertextSha256) fail("Encrypted vocabulary changed");
const calls = lines("calls.jsonl"), rows = lines("results.jsonl"), teaching = lines("teaching.jsonl");
const byCall = new Map(calls.map((c) => [c.callId, c]));
if (byCall.size !== calls.length || calls.length !== run.actualModelCalls) fail("Invalid actual-call count");
const quote = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const present = (text, term) => {
  const a = text.normalize("NFKC").toLowerCase(), b = term.normalize("NFKC").toLowerCase();
  return /[\u3400-\u9fff]/u.test(b) ? a.includes(b) : new RegExp("(?<![a-z0-9])" + [...b.replace(/\s/gu, "")].map(quote).join("\\s*") + "(?![a-z0-9])", "u").test(a);
};
const byTeaching = new Map(teachingInputs.map((x) => [x.id, x]));
const teachingLabels = new Map(labels.teaching.map((x) => [x.id, x]));
const seenTeaching = new Set();
let previousCount = 0;
for (const r of teaching) {
  if (seenTeaching.has(r.id)) fail("Duplicate teaching event");
  seenTeaching.add(r.id);
  if (r.persistence.beforeCount !== previousCount || r.persistence.afterCount !== (r.persistence.entries ?? []).length) fail("Broken persistence count chain");
  previousCount = r.persistence.afterCount;
  if (r.callId !== undefined) {
    const input = byTeaching.get(r.id), actual = byCall.get(r.callId);
    if (!input || !actual || actual.caseId !== r.id || actual.stage !== "teaching" || actual.context !== "" || actual.text !== r.edit.Inserted) fail("Teaching did not use actual ASR");
    if (r.edit.Settled !== "Entry: " + input.userEditedText + " | end") fail("Unapproved teaching correction");
  } else {
    const control = controls.find((x) => "edit-" + x.id === r.id);
    if (!control || control.kind !== r.kind || control.inserted !== r.edit.Inserted || control.settled !== r.edit.Settled) fail("Unknown edit control");
  }
}
if (teaching.length !== teachingInputs.length + controls.length || teaching.length !== run.teachingCases + run.editControlCases) fail("Missing teaching events");
if (previousCount !== snapshot.afterCount || JSON.stringify(teaching.at(-1).persistence.entries) !== JSON.stringify(snapshot.entries)) fail("Restart snapshot differs from persisted learning");
for (const e of memory) {
  if (!e.approved || e.origin !== "learned_edit") fail("Memory contains an unapproved or preauthored entry");
  if (!teaching.some((r) => r.persistence.extraction.Pairs?.some((p) => p.From === e.from && p.To === e.to))) fail("Memory lacks an actual edit source");
}
const by = new Map();
for (const r of rows) {
  const key = r.caseId + "/" + r.arm;
  if (by.has(key)) fail("Duplicate condition observation");
  by.set(key, r);
  const actual = r.callIds.map((id) => byCall.get(id));
  if (actual.some((c) => !c || c.caseId !== r.caseId) || actual.at(-1).text !== r.text) fail("Output is not its actual producing call");
  if (!future.some((x) => x.id === r.caseId) || !run.arms.includes(r.arm)) fail("Unknown inference input");
  if (r.selected.length > run.config.retrievalLimit || r.selected.some((h) => !memory.some((e) => e.id === h.id && e.to === h.target))) fail("Unstored recognition context");
  const terms = r.selected.map((h) => h.target).join(", ");
  const expected = terms ? (r.arm === "v2_hint" ? "The relevant terms are " + terms + "." : "Vocabulary: " + terms + ".") : "";
  if (r.context !== expected || actual.at(-1).context !== r.context) fail("Context not destination-only or call mismatch");
  const first = actual[0];
  if (first.stage !== "first_pass" || first.text !== r.firstPassText) fail("Retrieval lacks first-pass provenance");
  const twoPass = r.route === "audio_redecode";
  if (actual.length !== (twoPass ? 2 : 1) || Boolean(r.context) !== twoPass) fail("Invalid conditional re-decoding route");
  const measured = first.elapsedMs + r.retrievalMs + (twoPass ? actual[1].elapsedMs : 0);
  if (Math.abs(measured - r.pipelineMs) > 1e-6) fail("Invalid total compute time");
}
if (rows.length !== future.length * run.arms.length || rows.length !== run.observations || future.length !== run.recurrenceCases) fail("Incomplete paired run");
const allEntities = [...new Set([...labels.teaching.map((x) => x.term), ...memory.map((e) => e.to)])];
const wrong = (text, reference) => allEntities.filter((t) => present(text, t) && !present(reference, t));
const perCase = [];
for (const s of labels.samples) {
  const b = by.get(s.id + "/none");
  if (!b) fail("Missing baseline");
  const bs = errorCounts(s.reference, b.text, s.unit), bw = wrong(b.text, s.reference);
  for (const arm of run.arms) {
    const r = by.get(s.id + "/" + arm);
    if (!r) fail("Missing paired arm");
    const score = errorCounts(s.reference, r.text, s.unit);
    perCase.push({ ...r, ...s, ...score, baselineErrors: bs.errors, deltaErrors: score.errors - bs.errors,
      entityHit: s.term ? present(r.text, s.term) : null,
      baselineEntityHit: s.term ? present(b.text, s.term) : null,
      hasStoredTarget: s.term ? memory.some((e) => present(e.to, s.term)) : null,
      retrievalTargetHit: s.term ? r.selected.some((h) => present(h.target, s.term)) : null,
      newWrongEntities: wrong(r.text, s.reference).filter((t) => !bw.includes(t)) });
  }
}
const negatives = teaching.filter((x) => x.kind === "negative"), risks = teaching.filter((x) => x.kind === "semantic_risk");
const prior = teaching.filter((x) => x.callId !== undefined).map((r) => ({ id: r.id, term: teachingLabels.get(r.id).term,
  baselineEntityHit: present(r.edit.Inserted, teachingLabels.get(r.id).term),
  exactTargetLearned: (r.persistence.extraction.Pairs ?? []).some((p) => present(p.To, teachingLabels.get(r.id).term)),
  verdict: r.persistence.extraction.Verdict, pairs: r.persistence.extraction.Pairs ?? [] }));
const uniqueTerms = [...new Set(labels.teaching.map((x) => x.term))];
const teachingMisses = prior.filter((r) => !r.baselineEntityHit);
const lifecycle = fs.existsSync(path.join(directory, "lifecycle.json")) ? read(path.join(directory, "lifecycle.json")) : null;
const summary = { status: "complete", split: run.split, cases: future.length, observations: rows.length,
  actualModelCalls: calls.length, warmupCalls: lines("warmups.jsonl").length, selectionSha256: run.selectionSha256,
  learning: { initialEntries: initial.afterCount, teachingCases: teachingInputs.length, teachingEntityMisses: teachingMisses.length,
    missesProducingTargetPairs: teachingMisses.filter((r) => r.exactTargetLearned).length,
    distinctTerms: uniqueTerms.length, storedDistinctTerms: uniqueTerms.filter((t) => memory.some((e) => present(e.to, t))).length,
    storedEntries: memory.length, allLearnedPairs: teaching.reduce((n,r) => n+(r.persistence.extraction.Pairs?.length ?? 0),0),
    negativeEditCases: negatives.length, negativeEditsLearned: negatives.filter((r) => r.persistence.extraction.Pairs?.length).length,
    semanticRiskCases: risks.length, semanticRiskEditsLearned: risks.filter((r) => r.persistence.extraction.Pairs?.length).length,
    persistedSnapshots: teaching.length, finalReopenMatches: true, ciphertextUnchangedDuringRecurrence: true,
    undoRemovedEntry: lifecycle ? lifecycle.undo.afterCount === memory.length-1 : null,
    reversalOutcomes: lifecycle?.reversal.learning?.map((x) => x.Outcome) ?? [],
    mutationMs: distribution(teaching.map((r) => r.persistence.mutationMs)), perTeaching: prior },
  arms: {}, perCase, semanticReview: "not_independently_reviewed" };
for (const arm of run.arms) {
  const all = perCase.filter((r) => r.arm === arm), speech = all.filter((r) => r.referenceUnits > 0);
  const primary = speech.filter((r) => r.kind === "recurring"), controls = speech.filter((r) => r.kind !== "recurring");
  const rates = {};
  for (const lang of ["zh", "en", "mixed"]) {
    const g = speech.filter((r) => r.language === lang), errors = g.reduce((n,r) => n+r.errors,0), units = g.reduce((n,r) => n+r.referenceUnits,0);
    rates[lang] = { errors, referenceUnits: units, rate: units ? errors/units : null };
  }
  const baselineMisses = primary.filter((r) => !r.baselineEntityHit);
  summary.arms[arm] = { primaryEntityHits: primary.filter((r) => r.entityHit).length, primaryEntityCases: primary.length,
    storedTargetCases: primary.filter((r) => r.hasStoredTarget).length, storedTargetHits: primary.filter((r) => r.hasStoredTarget && r.entityHit).length,
    baselineEntityMisses: baselineMisses.length, baselineMissesRecovered: baselineMisses.filter((r) => r.entityHit).length,
    baselineEntityHitsLost: primary.filter((r) => r.baselineEntityHit && !r.entityHit).length,
    retrievalHitsAmongBaselineMisses: baselineMisses.filter((r) => r.retrievalTargetHit).length,
    newWrongEntityCases: all.filter((r) => r.newWrongEntities.length).length,
    baselineCorrectCases: speech.filter((r) => r.baselineErrors === 0).length,
    baselineCorrectRegressed: speech.filter((r) => r.baselineErrors === 0 && r.errors > 0).length,
    controlCases: controls.length, controlWorsened: controls.filter((r) => r.deltaErrors > 0).length,
    naturalCases: controls.filter((r) => r.kind === "natural_control").length,
    naturalWorsened: controls.filter((r) => r.kind === "natural_control" && r.deltaErrors > 0).length,
    improved: speech.filter((r) => r.deltaErrors < 0).length, worsened: speech.filter((r) => r.deltaErrors > 0).length,
    lexicalErrors: speech.reduce((n,r) => n+r.errors,0),
    nonspeechEmissions: all.filter((r) => r.kind === "nonspeech" && r.text.trim()).length,
    truncations: all.filter((r) => r.tokenLimitReached).length,
    redecodeCases: all.filter((r) => r.route === "audio_redecode").length,
    rates, pipelineMs: distribution(all.map((r) => r.pipelineMs)),
    routedExtraMs: distribution(all.filter((r) => r.route === "audio_redecode").map((r) => r.retrievalMs+r.secondPassMs)) };
}
fs.writeFileSync(output, JSON.stringify(summary,null,2)+"\n", {flag:"wx"});
console.log(JSON.stringify({ learning: { ...summary.learning, perTeaching: undefined }, arms: summary.arms },null,2));
