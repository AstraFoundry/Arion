/** Scroll distance, in stage heights, over which a row grows after it sticks. */
export const ROW_GROW = 0.55;

/**
 * Stage heights in the row's track: one for the sticky stage, then the grow,
 * then a short hold on the filled frame. Matches the `175svh` track.
 */
export const ROW_TRACK = 1.75;

const NARROW_HEIGHT = 34 * 16;
const WIDE_HEIGHT = 38 * 16;
const WIDE = 1024;
const CONTAINER = 80 * 16;

export interface RowLayout {
  /** Scroll offset at which the stage sticks. */
  trackTop: number;
  width: number;
  height: number;
}

export interface RowFrame {
  /** Row height in the stage. Grows from the resting card height to the stage. */
  cardHeight: number;
  /** Row width. Grows from the page column to the stage. */
  gridWidth: number;
  gap: number;
  radius: number;
}

const clamp = (value: number) => Math.min(1, Math.max(0, value));

/**
 * The highlight row at scroll offset `scroll`. It opens as the resting grid
 * (page column, card height, gap, and corner radius) and, once its stage
 * sticks, grows until the row fills the stage. Under reduced motion it is
 * the filled frame.
 */
export function rowFrame(layout: RowLayout, scroll: number, still: boolean): RowFrame {
  const { trackTop, width, height } = layout;
  const grow = still ? 1 : clamp(trackTop > 0 ? (scroll - trackTop) / (ROW_GROW * height) : 1);
  const wide = width >= WIDE;
  const restHeight = Math.min(wide ? WIDE_HEIGHT : NARROW_HEIGHT, height);
  const pad = width < 640 ? 20 : 32;
  const contentWidth = Math.max(0, Math.min(width, CONTAINER) - pad * 2);
  const restGap = width < 640 ? 12 : 16;
  const restRadius = wide ? 40 : 24;

  return {
    cardHeight: restHeight + (height - restHeight) * grow,
    gridWidth: contentWidth + (width - contentWidth) * grow,
    gap: restGap * (1 - grow),
    radius: restRadius * (1 - grow),
  };
}
