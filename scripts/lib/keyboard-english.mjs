import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

// Build bounded prefix aliases once, never enumerate completions on the key path.
export function writeEnglishDictionary(staging, english) {
  const frequencies = new Map();
  for (const row of readFileSync(path.join(staging, english.frequencies), 'utf8').trimEnd().split('\n')) {
    const [word, count] = row.split(' ');
    if (!word || !/^\d+$/.test(count)) throw new Error('Invalid pinned English frequency row.');
    frequencies.set(word, Number(count));
  }
  const words = new Map();
  for (const level of english.levels) {
    const source = readFileSync(path.join(staging, 'licenses/scowl', `english-words.${level}`), 'latin1');
    for (const word of source.trimEnd().split('\n')) {
      // Apostrophes remain Pinyin separators; single letters remain abbreviations.
      if (!/^[a-z]{2,}$/.test(word)) continue;
      if (!words.has(word)) words.set(word, 100 - level);
    }
  }
  if (!words.size) throw new Error('The pinned English word lists are empty.');
  const lexical = (a, b) => a < b ? -1 : a > b ? 1 : 0;
  const ranked = [...words.keys()].sort((a, b) =>
    (frequencies.get(b) ?? 0) - (frequencies.get(a) ?? 0) || words.get(b) - words.get(a) ||
    a.length - b.length || lexical(a, b));
  const prefixes = new Map();
  for (const word of ranked) {
    for (let length = 3; length <= word.length; length++) {
      const code = word.slice(0, length);
      const matches = prefixes.get(code) ?? [];
      if (matches.length < 3) matches.push(word);
      prefixes.set(code, matches);
    }
  }
  // Every complete word remains available, even outside its prefix's top three.
  for (const word of words.keys()) {
    const matches = prefixes.get(word) ?? [];
    prefixes.set(word, [word, ...matches.filter(match => match !== word)].slice(0, 3));
  }
  const rows = [...prefixes].sort(([a], [b]) => lexical(a, b)).flatMap(([code, matches]) => {
    const title = matches[0][0].toUpperCase() + matches[0].slice(1);
    const candidates = code === matches[0] ? [matches[0], title, ...matches.slice(1)] : [...matches, title];
    return candidates.map((word, index) => `${word}\t${code}\t${100 - index}`);
  });
  writeFileSync(path.join(staging, 'data/arion_english.dict.yaml'), [
    `# Derived from pinned SCOWL ${english.version}; see licenses/scowl/Copyright.`,
    '# Prefix ranking uses Hermit Dave FrequencyWords / OpenSubtitles 2018 counts (CC BY-SA 4.0).',
    '# Modified bounded prefix index, CC BY-SA 4.0; see licenses/frequencywords for source and license.',
    '# Two-letter exact words; prefixes from three letters; up to three words plus one case variant.',
    '---', 'name: arion_english', "version: '2'", 'sort: by_weight',
    'use_preset_vocabulary: false', '...', ...rows, '',
  ].join('\n'));
}
