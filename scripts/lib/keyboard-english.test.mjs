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

test('pinned English sources generate exact lowercase and sentence-case word pairs offline', () => {
  const staging = mkdtempSync(path.join(os.tmpdir(), 'arion-english-'));
  try {
    mkdirSync(path.join(staging, 'data'));
    mkdirSync(path.join(staging, 'licenses/scowl'), { recursive: true });
    for (const spec of manifest.files.filter(spec => spec.to.startsWith('licenses/scowl/'))) {
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
      assert.deepEqual(words, [code, code[0].toUpperCase() + code.slice(1)], 'The bounded Rime filter receives exactly two case variants.');
    }
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
