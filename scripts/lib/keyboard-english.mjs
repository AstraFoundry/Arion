import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

// Source tiers are dictionary coverage classes, not measured word frequencies.
export function writeEnglishDictionary(staging, english) {
  const words = new Map();
  for (const level of english.levels) {
    const source = readFileSync(path.join(staging, 'licenses/scowl', `english-words.${level}`), 'utf8');
    for (const word of source.trimEnd().split('\n')) {
      // Apostrophes remain Pinyin separators; single letters remain abbreviations.
      if (!/^[a-z]{2,}$/.test(word)) continue;
      if (!words.has(word)) words.set(word, 100 - level);
    }
  }
  if (!words.size) throw new Error('The pinned English word lists are empty.');
  const rows = [...words].sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0).flatMap(([word, weight]) => [
    `${word}\t${word}\t${weight}`,
    `${word[0].toUpperCase()}${word.slice(1)}\t${word}\t${weight - 1}`,
  ]);
  writeFileSync(path.join(staging, 'data/arion_english.dict.yaml'), [
    `# Derived from pinned SCOWL ${english.version}; see licenses/scowl/Copyright.`,
    '# ASCII words of two or more letters, with sentence-initial case variants.',
    '---', 'name: arion_english', "version: '1'", 'sort: by_weight',
    'use_preset_vocabulary: false', '...', ...rows, '',
  ].join('\n'));
}
