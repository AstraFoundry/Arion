import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { constraints, indexUnique, buildReport } from './report.mjs';

test('duplicate stage observations are rejected', () => {
  assert.throws(() => indexUnique([{ id: 'same' }, { id: 'same' }], 'id'), /Duplicate/);
});
test('English filler checks do not match inside document', () => {
  const result = constraints({ language: 'en', task: 'copy' }, { required: ['document'], forbidden: ['um'] }, 'Keep the document.');
  assert.equal(result.pass, true);
});
test('case and authored numeric aliases are accepted without losing negation', () => {
  const label = { required: ['eight', 'not'], forbidden: ['twelve'], aliases: { eight: ['8'], twelve: ['12'] } };
  assert.equal(constraints({ language: 'en' }, label, 'NOT 8').pass, true);
  assert.equal(constraints({ language: 'en' }, label, 'not 12').pass, false);
});
test('truncation and missing downstream observations cannot pass', () => {
  const label = { required: [], forbidden: [] };
  assert.equal(constraints({ language: 'zh' }, label, '', true).pass, false);
  assert.equal(constraints({ language: 'zh' }, label, null).pass, false);
});
test('punctuation-only silence output is still a strict false emission', () => {
  const result = constraints({ language: 'zh', task: 'silence' }, { required: [], forbidden: [] }, '。');
  assert.equal(result.pass, false);
  assert.equal(result.silenceViolation, true);
});
test('an incomplete run cannot be reported as complete', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'chain-report-'));
  try {
    fs.writeFileSync(path.join(directory, 'run.json'), JSON.stringify({ status: 'failed' }));
    assert.throws(() => buildReport(directory), /Incomplete/);
  } finally { fs.rmSync(directory, { recursive: true }); }
});
test('completed status cannot hide modified observations', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'chain-tamper-'));
  try {
    fs.writeFileSync(path.join(directory, 'observations.jsonl'), 'changed\n');
    fs.writeFileSync(path.join(directory, 'run.json'), JSON.stringify({ status: 'completed', outputHashes: { 'observations.jsonl': 'incorrect-digest' } }));
    assert.throws(() => buildReport(directory), /Changed output/);
  } finally { fs.rmSync(directory, { recursive: true }); }
});
