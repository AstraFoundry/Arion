"use client";

import {
  motion,
  useMotionValue,
  useReducedMotion,
  useScroll,
  useTransform,
  type MotionValue,
} from "motion/react";
import { useEffect, useLayoutEffect, useRef } from "react";
import { useCopy } from "@/shared/i18n";
import { cn, useInViewOnce } from "@/shared/lib";
import { CHAPTER_SPANS, type ChapterId } from "../config/chapters";
import { ROW_TRACK, type RowLayout, rowFrame } from "../lib/row-frame";
import { ChapterMock } from "./chapter-mock";
import { GridGround } from "./grid-ground";

/**
 * One highlight. The photograph is clipped to the card; on a shared row the
 * image is positioned against the row, so the cards stay windows onto one frame.
 */
function ChapterCard({
  id,
  clipPath,
  sharedPhoto,
}: {
  id: ChapterId;
  clipPath: MotionValue<string | undefined>;
  sharedPhoto: boolean;
}) {
  const { tour } = useCopy();
  const [ref, inView] = useInViewOnce<HTMLLIElement>();
  return (
    <motion.li
      ref={ref}
      style={{ clipPath }}
      className={cn(
        "relative flex h-full min-h-0 min-w-0 flex-col [clip-path:inset(0_round_1.5rem)]",
        sharedPhoto && "lg:static lg:[clip-path:inset(0_round_2.5rem)]",
      )}
    >
      <GridGround />
      <div className="flex min-h-0 flex-1 items-center justify-center px-6 pt-6 sm:px-10 sm:pt-10 lg:px-12 lg:pt-12">
        <ChapterMock id={id} playing={inView} />
      </div>
      <div className="bg-linear-to-t from-scrim/55 to-transparent px-6 pt-12 pb-6 sm:px-10 sm:pb-10 lg:px-12 lg:pb-12">
        <h3 className="max-w-[40ch] text-balance text-2xl leading-[1.35] font-normal text-substrate-tint-muted">
          {tour[id].title}
        </h3>
      </div>
    </motion.li>
  );
}

/**
 * A highlight row that sticks for a short scroll and grows from the resting
 * card grid until it fills the viewport. The static HTML is the resting grid;
 * measured frames take over after layout, the same way the hero stage does.
 */
export function ExpandingGrid({
  ids,
  sharedPhoto,
}: {
  ids: readonly ChapterId[];
  sharedPhoto: boolean;
}) {
  const track = useRef<HTMLDivElement>(null);
  const stage = useRef<HTMLDivElement>(null);
  const layout = useMotionValue<RowLayout | null>(null);
  const reduce = useReducedMotion() ?? false;
  const still = useMotionValue(false);
  useEffect(() => still.set(reduce), [reduce, still]);

  useLayoutEffect(() => {
    const measure = () => {
      if (!track.current || !stage.current) return;
      const width = stage.current.clientWidth;
      const height = stage.current.clientHeight;
      if (width === 0 || height === 0) {
        layout.set(null);
        return;
      }
      layout.set({
        trackTop: track.current.getBoundingClientRect().top + window.scrollY,
        width,
        height,
      });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(document.body);
    observer.observe(stage.current!);
    return () => observer.disconnect();
  }, [layout]);

  const { scrollY } = useScroll();
  // Read every input on the first pass. A computed transform subscribes only
  // to the motion values that pass reads then, and the layout is still empty.
  const frame = useTransform(() => {
    const scroll = scrollY.get();
    const reduced = still.get();
    const current = layout.get();
    return current ? rowFrame(current, scroll, reduced) : null;
  });
  const height = useTransform(frame, (value) => value?.cardHeight);
  const width = useTransform(frame, (value) => value?.gridWidth);
  const gap = useTransform(frame, (value) => (value ? `${value.gap}px` : undefined));
  const paddingInline = useTransform(frame, (value) => (value ? 0 : undefined));
  const maxWidth = useTransform(frame, (value) => (value ? ("none" as const) : undefined));
  const clipPath = useTransform(frame, (value) =>
    value ? `inset(0px round ${value.radius}px)` : undefined,
  );
  const template = ids.map((id) => `${CHAPTER_SPANS[id]}fr`).join(" ");

  return (
    <div ref={track} className="relative" style={{ height: `${ROW_TRACK * 100}svh` }}>
      <div ref={stage} className="sticky top-0 h-svh overflow-hidden">
        <motion.ul
          className={cn(
            "relative mx-auto grid w-full max-w-7xl",
            sharedPhoto ? "h-[38rem] gap-4 px-8" : "h-[34rem] gap-3 px-5 sm:gap-4 sm:px-8",
          )}
          style={{
            height,
            width,
            maxWidth,
            gap,
            paddingInline,
            gridTemplateColumns: template,
          }}
        >
          {ids.map((id) => (
            <ChapterCard key={id} id={id} clipPath={clipPath} sharedPhoto={sharedPhoto} />
          ))}
        </motion.ul>
      </div>
    </div>
  );
}
