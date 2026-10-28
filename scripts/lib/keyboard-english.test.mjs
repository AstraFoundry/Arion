import test from 'node:test';
import assert from 'node:assert/strict';
import { appendFileSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './pinned-file.mjs';
import { writeEnglishLexicon } from './keyboard-english.mjs';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const manifest = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
const entries = directory => readFileSync(path.join(directory, 'data/arion_english.lexicon'), 'utf8')
  .trimEnd().split('\n').slice(1).map(line => { const [key, display, rank] = line.split('\t'); return { key, display, rank: Number(rank) }; });

test('complete pinned ESDB 70 and CSpell vocabulary retains case, frequency-independent membership and all prefix matches', () => {
  const staging = mkdtempSync(path.join(os.tmpdir(), 'arion-english-'));
  try {
    mkdirSync(path.join(staging, 'data'));
    for (const spec of manifest.files.filter(file => /^licenses\/(esdb|cspell|frequencywords)\//.test(file.to))) {
      const destination = path.join(staging, spec.to);
      mkdirSync(path.dirname(destination), { recursive: true });
      copyFileSync(pinnedFile(spec), destination);
    }
    const count = writeEnglishLexicon(staging, manifest);
    assert.ok(count.words > 145000);
    const rows = entries(staging);
    const words = new Set(rows.map(row => row.display));
    for (const word of ['artificial', 'computer', 'app', 'API', 'GitHub', 'JavaScript', 'TypeScript',
      'chatgpt', 'frontend', 'backend', 'workflow', 'plugin', 'config', 'asynchronous']) assert.ok(words.has(word), word);
    assert.ok(rows.filter(row => row.key.startsWith('con')).length > 1700, 'Later completions remain indexed.');
    assert.equal(rows.length, count.displays);
    assert.equal(new Set(rows.map(row => row.rank)).size, rows.length);
    assert.equal(new Set(rows.map(row => row.key + '\t' + row.display)).size, rows.length);
    const github = rows.filter(row => row.key === 'github').sort((a, b) => a.rank - b.rank);
    assert.equal(github[0].display, 'GitHub');
    const original = readFileSync(path.join(staging, 'data/arion_english.lexicon'));
    writeEnglishLexicon(staging, manifest);
    assert.deepEqual(readFileSync(path.join(staging, 'data/arion_english.lexicon')), original);
    const pin = manifest.files.find(file => file.english === 'cspell');
    const changed = path.join(staging, pin.to);
    appendFileSync(changed, '\nchanged\n');
    assert.throws(() => pinnedFile({ ...pin, path: changed }), /size or SHA-256 mismatch/);
  } finally { rmSync(staging, { recursive: true, force: true }); }
});

test('canonical technical case, exact dedup and unseen words survive normalization', () => {
  const staging = mkdtempSync(path.join(os.tmpdir(), 'arion-english-policy-'));
  try {
    mkdirSync(path.join(staging, 'data'));
    writeFileSync(path.join(staging, 'esdb'), 'react\nReact\nhello\nhelp\nartifice\n');
    writeFileSync(path.join(staging, 'technical'), 'React\nGitHub\nJavaScript # product\n# comment\nfoo+bar\ntwo words\n');
    writeFileSync(path.join(staging, 'frequency'), 'help 20\nhello 10\n');
    writeEnglishLexicon(staging, { english: { frequencies: 'frequency' }, files: [
      { to: 'esdb', english: 'esdb' }, { to: 'technical', english: 'cspell' },
    ] });
    const rows = entries(staging), words = rows.map(row => row.display);
    assert.equal(rows.filter(row => row.key === 'react').sort((a, b) => a.rank - b.rank)[0].display, 'React');
    assert.equal(words.filter(word => word === 'React').length, 1);
    assert.ok(words.includes('react') && words.includes('artifice') && words.includes('Hello'));
    assert.ok(words.includes('GitHub') && !words.includes('Github'));
    assert.ok(words.includes('JavaScript') && !words.includes('foo+bar') && !words.includes('two words'));
    assert.ok(rows.find(row => row.display === 'help').rank < rows.find(row => row.display === 'hello').rank);
  } finally { rmSync(staging, { recursive: true, force: true }); }
});
