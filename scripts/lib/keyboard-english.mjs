import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

const lexical = (a, b) => a < b ? -1 : a > b ? 1 : 0;
const eligible = /^[A-Za-z]{2,128}$/;
export const englishLexiconHeader = 'ARION_ENGLISH\t2\t';

export function writeEnglishLexicon(staging, manifest) {
  const frequencies = new Map();
  for (const row of readFileSync(path.join(staging, manifest.english.frequencies), 'utf8').trimEnd().split('\n')) {
    const [word, count] = row.split(' ');
    if (!word || !/^\d+$/.test(count)) throw new Error('Invalid pinned English frequency row.');
    frequencies.set(word, Number(count));
  }
  const words = new Map();
  for (const source of manifest.files.filter(file => file.english)) {
    if (!['esdb', 'cspell'].includes(source.english)) throw new Error('Unknown English word-source format.');
    for (const row of readFileSync(path.join(staging, source.to), 'utf8').split('\n')) {
      const display = (source.english === 'cspell' ? row.replace(/\s+#.*$/, '') : row).trim();
      if (!eligible.test(display)) continue;
      const key = display.toLowerCase();
      const entry = words.get(key) ?? { key, displays: new Map() };
      const technical = source.english === 'cspell';
      entry.displays.set(display, technical || entry.displays.get(display) === true);
      words.set(key, entry);
    }
  }
  if (!words.size) throw new Error('The pinned English lexicon is empty.');
  const ordered = [...words.values()].sort((a, b) =>
    (frequencies.get(b.key) ?? 0) - (frequencies.get(a.key) ?? 0) ||
    a.key.length - b.key.length || lexical(a.key, b.key));
  const rows = [];
  let displayCount = 0;
  for (const entry of ordered) {
    // Source-native case always survives; derive sentence case only for ordinary lowercase words.
    if (entry.displays.has(entry.key) && ![...entry.displays].some(([text, technical]) => technical && text !== entry.key)) {
      const title = entry.key[0].toUpperCase() + entry.key.slice(1);
      if (!entry.displays.has(title)) entry.displays.set(title, false);
    }
    const displays = [...entry.displays].sort(([a, at], [b, bt]) =>
      Number(bt) - Number(at) || Number(b === entry.key) - Number(a === entry.key) || lexical(a, b));
    displayCount += displays.length;
    rows.push({ key: entry.key, displays: displays.map(([display]) => display), rank: rows.length,
      count: frequencies.get(entry.key) ?? 0 });
  }
  rows.sort((a, b) => lexical(a.key, b.key));
  writeFileSync(path.join(staging, 'data/arion_english.lexicon'), [
    `${englishLexiconHeader}${rows.length}`,
    ...rows.map(({ key, displays, rank, count }) => `${key}\t${rank}\t${count}\t${displays.join(',')}`), '',
  ].join('\n'));
  return { words: words.size, displays: displayCount };
}
