import { describe, expect, it } from "vitest";
import { CHAPTER_SPANS, CHAPTERS, DESKTOP_ROWS } from "@/widgets/feature-tour/config/chapters";
import { ROW_GROW, rowFrame } from "@/widgets/feature-tour/lib/row-frame";

describe("feature tour rows", () => {
  it("keeps every chapter once and every desktop row at three columns", () => {
    expect(DESKTOP_ROWS.flat()).toEqual([...CHAPTERS]);
    for (const row of DESKTOP_ROWS) {
      const span = row.reduce((sum, id) => sum + CHAPTER_SPANS[id], 0);
      expect(span).toBe(3);
    }
  });
});

describe("rowFrame", () => {
  const wide = { trackTop: 2000, width: 1440, height: 800 };

  it("opens as the resting page column", () => {
    expect(rowFrame(wide, wide.trackTop, false)).toEqual({
      cardHeight: 38 * 16,
      gridWidth: 80 * 16 - 64,
      gap: 16,
      radius: 40,
    });
  });

  it("fills the stage after the grow distance", () => {
    const scroll = wide.trackTop + ROW_GROW * wide.height;
    const frame = rowFrame(wide, scroll, false);
    expect(frame.cardHeight).toBe(wide.height);
    expect(frame.gridWidth).toBe(wide.width);
    expect(frame.gap).toBeCloseTo(0);
    expect(frame.radius).toBeCloseTo(0);
  });

  it("is halfway through the grow at the midpoint", () => {
    const frame = rowFrame(wide, wide.trackTop + (ROW_GROW * wide.height) / 2, false);
    expect(frame.cardHeight).toBeCloseTo((38 * 16 + wide.height) / 2);
    expect(frame.gap).toBeCloseTo(8);
    expect(frame.radius).toBeCloseTo(20);
  });

  it("uses the narrow card on a phone-width stage", () => {
    expect(rowFrame({ trackTop: 100, width: 390, height: 700 }, 100, false)).toEqual({
      cardHeight: 34 * 16,
      gridWidth: 390 - 40,
      gap: 12,
      radius: 24,
    });
  });

  it("shows the filled frame under reduced motion", () => {
    expect(rowFrame(wide, 0, true)).toMatchObject({
      cardHeight: wide.height,
      gridWidth: wide.width,
      gap: 0,
      radius: 0,
    });
  });
});
