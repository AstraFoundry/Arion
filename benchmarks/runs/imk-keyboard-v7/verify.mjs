// Frozen-evidence verification only: no network, runtime, Keychain or OS mutation.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.dirname(fileURLToPath(import.meta.url));
const read = file => readFileSync(path.join(root, file));
const json = file => JSON.parse(read(file));
const archive = json('archive.json');
const paths = new Set();
for (const entry of archive.files) {
  assert(!path.isAbsolute(entry.path) && !entry.path.split('/').includes('..'));
  assert(!paths.has(entry.path)); paths.add(entry.path);
  const bytes = read(entry.path);
  assert.equal(bytes.length, entry.sizeBytes, entry.path);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), entry.sha256, entry.path);
}
function inventory(directory) {
  return readdirSync(directory).flatMap(name => {
    const file = path.join(directory, name);
    return statSync(file).isDirectory() ? inventory(file) : [path.relative(root, file).split(path.sep).join('/')];
  });
}
assert.deepEqual(inventory(root).filter(file => file !== 'archive.json').sort(), [...paths].sort());
const before = json(archive.functional.before);
assert.equal(before.passed, false); assert.equal(before.cases.length, 23);
assert.equal(before.cases.filter(test => test.passed).length, 22);
const failed = before.cases.find(test => !test.passed);
assert.equal(failed.case, 'literal umlaut Pinyin uses the declared schema alias');
assert.equal(failed.failure, 'engine handles synthetic letter ü');
for (const [file, arch, translated] of [[archive.functional.native, 'arm64', 'none'], [archive.functional.translated, 'x86_64', 'Rosetta 2']]) {
  const report = json(file);
  assert(report.passed && report.cases.length === 23 && report.cases.every(test => test.passed));
  assert.equal(report.execution.hostArchitecture, 'arm64'); assert.equal(report.execution.executableArchitecture, arch);
  assert.equal(report.execution.translation, translated);
  assert.equal(report.execution.physicalIntelHost, false); assert.equal(report.execution.installedIMEHostDispatch, false);
  const literal = report.cases.find(test => test.case === failed.case);
  assert.equal(literal.observed.text, '绿色');
  assert(literal.trace[0].state.candidates.includes('绿色'));
}
for (const binding of archive.functional.sourceBindings) {
  assert(!path.isAbsolute(binding.frozenPath));
  if (binding.frozenPath.startsWith('../')) assert(/^\.\.\/imk-keyboard-v[1-6]\//.test(binding.frozenPath));
  const bytes = read(binding.frozenPath); assert.equal(bytes.length, binding.sizeBytes);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), binding.sha256, binding.originalPath);
}
assert(read('checks/unsigned-host-literal-umlaut-build.log').toString().includes('Built apps/desktop/resources/darwin/ArionInput.app'));
assert(read('checks/unsigned-host-literal-umlaut-verify.log').toString().includes('(unsealed-development)'));
const assessment = json(archive.candidateProbe.assessment);
for (const file of assessment.files) assert.equal(createHash('sha256').update(read('candidate-probe/' + file.path)).digest('hex'), file.sha256);
assert(assessment.outcome.startsWith('scope-blocked')); assert.equal(assessment.processExit, 1);
assert.equal(assessment.remainingProbeProcessQueryExit, 1); assert.deepEqual(assessment.remainingProbePIDs, []);
const probe = json('candidate-probe/report.json');
assert.equal(probe.passed, false); assert.equal(probe.serverCreated, 1); assert.equal(probe.candidatesCreated, 1); assert.equal(probe.dataSet, true);
assert(probe.observations.every(item => item.identifier === 0 && item.selectedIdentifier === 0 && item.selectedText === null));
for (const file of ['setup-failed-crash.json', 'class-setup-failed-crash.json']) assert.equal(json('candidate-probe/' + file).exception.type, 'EXC_BAD_ACCESS');
const evidence = json(archive.static.ledger);
assert.equal(evidence.summary.newIntervalBytesHashed, 2228); assert.equal(evidence.summary.newInstructionsValidated, 557);
assert.equal(evidence.summary.allIntervalBytesHashed, 7808); assert.equal(evidence.summary.allInstructionsValidated, 1952);
assert.equal(evidence.intervals.length, 6);
for (const interval of evidence.intervals) {
  assert.equal(interval.instructionsValidated * 4, interval.sizeBytes);
  assert.deepEqual(interval.unrecognizedInstructionSlots, []);
  assert(Number.isInteger(interval.universalFileOffset));
}
assert.equal(evidence.artifacts.main.universalSha256, '4b3836dcb8a3e06b01b4c95390c757b71f623c1f7c4141c059e49284f7f6442d');
assert.equal(evidence.artifacts.engine.universalSha256, 'd1cebd7c9a9d888acc2a6d2f4aad372cb3b31570ff68a7e631cba930517f5db8');
assert(evidence.observations.dispatch.includes('UI refresh')); assert(evidence.observations.decoderBoundary.includes('unknown'));
console.log(`PASS: ${archive.files.length} SHA-256 mappings and ${archive.functional.sourceBindings.length} source bindings; literal-ü failure retained, native/Rosetta 23/23; inactive candidate probe remains inconclusive; focused static provenance retained without binary/disassembly carriers.`);
