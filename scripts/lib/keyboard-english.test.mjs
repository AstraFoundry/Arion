import test from 'node:test';
import assert from 'node:assert/strict';
import { appendFileSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './pinned-file.mjs';
import { writeEnglishDictionary } from './keyboard-english.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const manifest = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));

test('pinned English sources generate bounded frequency-ranked prefixes and preserve exact words offline', () => {
  const staging = mkdtempSync(path.join(os.tmpdir(), 'arion-english-'));
  try {
    mkdirSync(path.join(staging, 'data'));
    mkdirSync(path.join(staging, 'licenses/scowl'), { recursive: true });
    mkdirSync(path.join(staging, 'licenses/frequencywords'), { recursive: true });
    for (const spec of manifest.files.filter(spec => /^licenses\/(scowl|frequencywords)\//.test(spec.to))) {
      copyFileSync(pinnedFile(spec), path.join(staging, spec.to));
    }
    const output = path.join(staging, 'data/arion_english.dict.yaml');
    writeEnglishDictionary(staging, manifest.english);
    const dictionary = readFileSync(output, 'utf8');
    for (const word of ['artificial', 'hello', 'computer', 'software', 'can', 'an', 'you', 'hi', 'bailing', 'jingling']) {
      assert.match(dictionary, new RegExp(`^${word}\\t${word}\\t[0-9]+$`, 'm'));
    }
    const byCode = new Map();
    for (const row of dictionary.split('...\n')[1].trimEnd().split('\n')) {
      const [word, code] = row.split('\t');
      assert.match(code, /^[a-z]{2,}$/);
      const words = byCode.get(code) ?? [];
      words.push(word);
      byCode.set(code, words);
    }
    assert.ok(byCode.size > 20000, 'Common tiers retain normal inflected vocabulary.');
    for (const [code, words] of byCode) {
      assert.ok(words.length <= 4, 'The native filter has a fixed upper bound.');
      assert.equal(new Set(words).size, words.length);
      assert.ok(words.every(word => word.toLowerCase().startsWith(code)));
      if (code.length === 2) assert.deepEqual(words, [code, code[0].toUpperCase() + code.slice(1)]);
    }
    assert.equal(byCode.get('artifi')[0], 'artificial');
    assert.equal(byCode.get('hel')[0], 'help', 'Common words lead rare words, rather than alphabetical order.');
    assert.ok(byCode.get('hel').includes('hello'));
    assert.deepEqual(byCode.get('can').slice(0, 2), ['can', 'Can'], 'Exact word and case choice remain first.');
    assert.equal(byCode.has('he'), true);
    assert.equal(byCode.has('ar'), false, 'Two-letter prefixes do not trigger completion.');
    assert.equal(byCode.has('qqzzxx'), false);
    writeEnglishDictionary(staging, manifest.english);
    assert.equal(readFileSync(output, 'utf8'), dictionary, 'Generation is deterministic.');
    const spec = manifest.files.find(spec => spec.to.endsWith('english-words.10'));
    const changed = path.join(staging, spec.to);
    appendFileSync(changed, '\nchanged\n');
    assert.throws(() => pinnedFile({ ...spec, path: changed }), /size or SHA-256 mismatch/);
  } finally {
    rmSync(staging, { recursive: true, force: true });
  }
});
