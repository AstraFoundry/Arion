// Replay retained native observations; this command does not execute an OS input method.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {englishAdmissionFixtures} from './lib/english-admission-fixtures.mjs';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const directory=path.join(root,'benchmarks/runs/ime-english-admission-v1');
const provenance=JSON.parse(readFileSync(path.join(directory,'provenance.json'),'utf8'));
for(const file of provenance.files) {
  assert.equal(path.basename(file.file),file.file,'Unsafe archived member');
  const bytes=readFileSync(path.join(directory,file.file));
  assert.equal(createHash('sha256').update(bytes).digest('hex'),file.sha256,`Changed native evidence: ${file.file}`);
}
const report=JSON.parse(readFileSync(path.join(directory,'native-final.json'),'utf8'));
assert.equal(report.passed,true);
assert.equal(report.execution.dirty,false);
let count=0;
for(const file of englishAdmissionFixtures(root)) {
  const split=path.basename(file,'.tsv');
  const rows=readFileSync(file,'utf8').split('\n').filter(row=>row && !row.startsWith('#')).map(row=>row.split('\t'));
  const group=report.cases.find(item=>item.case===`English admission ${split} on native menus`);
  assert.equal(group?.passed,true,split);
  assert.equal(group.trace.length,rows.length*2);
  let step=0,commits=0,text='';
  for(const page of [5,9]) for(const [id,input,expected,limit,chinese] of rows) {
    const observed=group.trace[step++];
    assert.equal(observed.step,`${id} page ${page}`);
    assert.equal(observed.state.insertions,commits,id);
    const candidates=observed.state.candidates;
    assert.ok(candidates.length<=page,id);
    assert.ok(candidates.filter(word=>/^[A-Za-z]{2,}$/.test(word)).length<=Number(limit),id);
    if(chinese==='1') assert.ok(candidates[0]?.codePointAt(0)>127,id);
    if(expected!=='-') { assert.ok(candidates.includes(expected),`${input} → ${expected}`); ++commits; text+=expected; }
    ++count;
  }
  assert.equal(group.observed.insertions,commits,split);
  assert.equal(group.observed.text,text,split);
  assert.equal(group.observed.marked,false,split);
}
assert.equal(count,provenance.result.inputPageCases);
const windows = readFileSync(path.join(directory, 'windows-keyboard-standard-user.stdout.log'), 'utf8');
const traces = windows.split(/\r?\n/).filter(line => line.startsWith('ADMISSION\t'));
const seen = new Map(traces.map(line => {
  const [, id, page, ...candidates] = line.split('\t');
  return [`${id}:${page}`, candidates];
}));
assert.equal(traces.length, count);
assert.equal(seen.size, count);
for (const file of englishAdmissionFixtures(root)) {
  const rows = readFileSync(file, 'utf8').split('\n').filter(row => row && !row.startsWith('#'));
  for (const page of [5, 9]) for (const row of rows) {
    const [id, input, expected, limit, chinese] = row.split('\t');
    const candidates = seen.get(`${id}:${page}`);
    assert.ok(candidates, `Missing Windows observation: ${id}:${page}`);
    assert.ok(candidates.length <= page, id);
    assert.ok(candidates.filter(word => /^[A-Za-z]{2,}$/.test(word)).length <= Number(limit), id);
    if (chinese === '1') assert.ok(candidates[0]?.codePointAt(0) > 127, id);
    if (expected !== '-') assert.ok(candidates.includes(expected), `${input} → ${expected}`);
  }
}
assert.ok(windows.includes('PASS english-admission-development-and-validation-five-nine-pages'));
console.log(`Verified ${count} archived admission checks per platform, macOS exact-once totals and Windows native commit assertions; no OS execution or latency measurement.`);
