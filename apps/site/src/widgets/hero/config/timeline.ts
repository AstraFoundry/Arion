/** How many partial hypotheses the pill shows before the final text. */
export const PARTIALS = 5;

/**
 * Steps of the hero demo, in order. One loop walks through the highlights
 * in the same document: a dictation, a correction Arion learns, a polished
 * dictation, then Pinyin typed with the input method and a Shift switch to
 * English.
 */
export const HERO_STEP = {
  idle: 0,
  listening: 1,
  firstPartial: 2,
  inserted: 2 + PARTIALS,
  learnTyped: 3 + PARTIALS,
  learnSelected: 4 + PARTIALS,
  learnCorrected: 5 + PARTIALS,
  learned: 6 + PARTIALS,
  polishHeard: 7 + PARTIALS,
  polishHeardAll: 8 + PARTIALS,
  polishing: 9 + PARTIALS,
  polished: 10 + PARTIALS,
  pinyinStart: 11 + PARTIALS,
  pinyin: 12 + PARTIALS,
  pinyinCommitted: 13 + PARTIALS,
  switched: 14 + PARTIALS,
} as const;

/** Milliseconds spent in each step before the next one; the last step is `switched`. */
export const HERO_DELAYS: readonly number[] = [
  900, // idle
  1100, // listening
  ...Array.from({ length: PARTIALS }, () => 420),
  1600, // inserted
  900, // learnTyped
  700, // learnSelected
  700, // learnCorrected
  1800, // learned
  700, // polishHeard
  1100, // polishHeardAll
  1000, // polishing
  1800, // polished
  600, // pinyinStart
  1000, // pinyin
  800, // pinyinCommitted
];

/** How long the finished document rests before the loop starts over. */
export const HERO_LOOP_REST = 2600;

/** The streaming draft at `step`: a growing prefix of the sentence. */
export function partialText(sentence: string, step: number): string {
  const chars = Array.from(sentence);
  const shown = step - HERO_STEP.firstPartial + 1;
  if (shown <= 0) return "";
  return chars.slice(0, Math.ceil((chars.length * shown) / (PARTIALS + 1))).join("");
}

/** The first half of a spoken sentence, as the pill streams it. */
export function halfText(sentence: string): string {
  const chars = Array.from(sentence);
  return chars.slice(0, Math.ceil(chars.length / 2)).join("");
}
