/** How many partial hypotheses the pill shows before the final text. */
export const PARTIALS = 5;

/** Steps of the hero demo, in order. */
export const HERO_STEP = {
  idle: 0,
  listening: 1,
  firstPartial: 2,
  inserted: 2 + PARTIALS,
} as const;

/** Milliseconds spent in each step before the next one. */
export const HERO_DELAYS: readonly number[] = [
  900,
  1300,
  ...Array.from({ length: PARTIALS }, () => 420),
];

/** The streaming draft at `step`: a growing prefix of the sentence. */
export function partialText(sentence: string, step: number): string {
  const chars = Array.from(sentence);
  const shown = step - HERO_STEP.firstPartial + 1;
  if (shown <= 0) return "";
  return chars.slice(0, Math.ceil((chars.length * shown) / (PARTIALS + 1))).join("");
}
