import { History, Keyboard, Languages, PenLine, WifiOff, type LucideIcon } from "lucide-react";

/**
 * The tour's chapters, in reading order. Each id names its copy under
 * `tour.<id>`, its mock and its backdrop crop (scripts/build-assets.mjs).
 */
export const CHAPTERS = ["offline", "mixed", "learning", "polish", "keyboard"] as const;

export type ChapterId = (typeof CHAPTERS)[number];

export const CHAPTER_ICONS: Record<ChapterId, LucideIcon> = {
  offline: WifiOff,
  mixed: Languages,
  learning: History,
  polish: PenLine,
  keyboard: Keyboard,
};

export const backdropSrc = (id: ChapterId) => `/backdrops/tour-${id}-800.webp`;

/**
 * Milliseconds each mock spends in a step before the next; a mock has
 * `length + 1` steps and replays whenever its chapter becomes active.
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
