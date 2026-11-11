/**
 * The highlights, in reading order. Each id names its copy under
 * `tour.<id>` and its mock.
 */
export const CHAPTERS = ["offline", "mixed", "learning", "polish", "keyboard"] as const;

export type ChapterId = (typeof CHAPTERS)[number];

/**
 * Columns each card spans in the three-column grid from `lg` up. Each row
 * fills three columns: a narrow card beside a wide one, then three narrow.
 */
export const CHAPTER_SPANS: Record<ChapterId, 1 | 2> = {
  offline: 1,
  mixed: 2,
  learning: 1,
  polish: 1,
  keyboard: 1,
};

/**
 * Milliseconds each mock spends in a step before the next; a mock has
 * `length + 1` steps and plays once its card scrolls into view.
 */
export const MOCK_DELAYS: Record<ChapterId, readonly number[]> = {
  // network on → network off → dictation inserted
  offline: [900, 1300],
  // one step per streaming hypothesis, then the committed sentence
  mixed: [600, 650, 650, 650, 800],
  // misrecognized → selected → corrected → learned
  learning: [1000, 700, 700],
  // heard → polishing → typed
  polish: [900, 900],
  // first syllable → full Pinyin with candidates → committed → Shift switches mode
  keyboard: [700, 800, 900, 800],
};
