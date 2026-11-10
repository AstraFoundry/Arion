"use client";

import {
  motion,
  type MotionValue,
  useInView,
  useMotionValue,
  useReducedMotion,
  useScroll,
  useTransform,
} from "motion/react";
import { useEffect, useLayoutEffect, useRef } from "react";
import { HeroDemo } from "./hero-demo";
import { HeroGround } from "./hero-ground";

/** Where the demo's bottom edge rests, as a share of the photograph box's height. */
const REST_LINE = 0.6;
/** Where the flower field starts, as a share of the photograph box's height. */
const FLOWER_LINE = 0.64;
/** Where the flowers start in the opening card, as a share of its height. */
const OPENING_FLOWERS = 0.78;
/** Where the window's top edge sits in the opening card, as a share of its height. */
const OPENING_WINDOW = 0.18;
/** Scroll distance, in viewport heights, over which the demo rises once the stage sticks. */
const RISE = 0.6;

interface Layout {
  /** The track's top edge in the page: the scroll offset at which the stage sticks. */
  trackTop: number;
  width: number;
  height: number;
  demoHeight: number;
}

interface Frame {
  clipPath: string;
  groundY: number;
  demoY: number;
}

const clamp = (value: number) => Math.min(1, Math.max(0, value));

/**
 * The stage at scroll offset `scroll`. Before the stage sticks, the card's
 * bottom edge stays at the bottom of the viewport while its top rises with
 * the page, so the card grows; its gutter and corners shrink to nothing as it
 * reaches the top. Once it sticks, the demo rises out of the flowers.
 */
function frameAt(layout: Layout, scroll: number, still: boolean): Frame {
  const { trackTop, width, height, demoHeight } = layout;
  const grow = still ? 1 : clamp(trackTop > 0 ? scroll / trackTop : 1);
  const rise = still ? 1 : clamp((scroll - trackTop) / (RISE * height));
  const gutter = (width < 640 ? 8 : 12) * (1 - grow);
  const radius = (width < 1024 ? 24 : 40) * (1 - grow);
  const below = still ? 0 : Math.max(0, trackTop - scroll) + gutter;

  // The photograph box (see GROUND_BOX) and the lines in it, in stage pixels.
  const boxHeight = Math.max(height, width / 1.5);
  const flowerLine = FLOWER_LINE * boxHeight;
  // The opening card spans the viewport below the copy; the photograph moves
  // down so its flowers sit low in that card, then settles as the card grows.
  const openingCard = Math.max(0, height - trackTop - gutter);
  const groundY = (1 - grow) * (OPENING_FLOWERS * openingCard - flowerLine);
  // At first the window's top edge sits high in the card and the card's
  // bottom edge cuts it off; it rises to rest as the stage sticks.
  const restTop = REST_LINE * boxHeight - demoHeight;
  const openingGroundY = OPENING_FLOWERS * openingCard - flowerLine;
  const openingTop = OPENING_WINDOW * openingCard - openingGroundY;
  const demoY = (1 - rise) * (openingTop - restTop);

  return {
    clipPath: `inset(0px ${gutter}px ${below}px round ${radius}px)`,
    groundY,
    demoY,
  };
}

/** The static HTML's frame, before the stage is measured: an inset card. */
const OPENING: Frame = {
  clipPath: "inset(0px 12px 50svh round 40px)",
  groundY: 0,
  demoY: 0,
};

function useFrame(layout: MotionValue<Layout | null>, still: MotionValue<boolean>) {
  const { scrollY } = useScroll();
  const at = () => {
    const current = layout.get();
    return current ? frameAt(current, scrollY.get(), still.get()) : OPENING;
  };
  return {
    clipPath: useTransform(() => at().clipPath),
    groundY: useTransform(() => at().groundY),
    demoY: useTransform(() => at().demoY),
  };
}

/**
 * The hero's stage. The first screen shows it as a card under the copy: the
 * hillside photograph with the top half of the demo's window floating on it,
 * cut off by the card's bottom edge. As the reader scrolls, the card grows to
 * the full width and height of the viewport, then sticks while the demo rises
 * to rest, and holds before the page moves on. Under reduced motion it shows the final frame and
 * does not move.
 */
export function HeroStage() {
  const track = useRef<HTMLDivElement>(null);
  const stage = useRef<HTMLDivElement>(null);
  const demo = useRef<HTMLDivElement>(null);
  const layout = useMotionValue<Layout | null>(null);

  // The static HTML carries the opening frame; reduced motion moves to the
  // final frame through a motion value, so hydration never meets a
  // differing attribute.
  const reduce = useReducedMotion() ?? false;
  const still = useMotionValue(false);
  useEffect(() => still.set(reduce), [reduce, still]);

  useLayoutEffect(() => {
    const measure = () => {
      if (!track.current || !stage.current || !demo.current) return;
      layout.set({
        trackTop: track.current.getBoundingClientRect().top + window.scrollY,
        width: stage.current.clientWidth,
        height: stage.current.clientHeight,
        demoHeight: demo.current.offsetHeight,
      });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(document.body);
    observer.observe(stage.current!);
    observer.observe(demo.current!);
    void document.fonts.ready.then(measure);
    return () => observer.disconnect();
  }, [layout]);

  const { clipPath, groundY, demoY } = useFrame(layout, still);
  const inView = useInView(stage, { amount: 0.2 });

  return (
    <div ref={track} className="relative h-[200svh]">
      <div ref={stage} className="sticky top-0 h-svh overflow-hidden [container-type:size]">
        <motion.div className="absolute inset-0 isolate bg-substrate" style={{ clipPath }}>
          <motion.div className="absolute inset-0" style={{ y: groundY }}>
            <HeroGround />
            <div
              className="absolute inset-x-0 flex justify-center px-5 sm:px-8"
              style={{ bottom: `calc(100cqh - ${REST_LINE} * max(100cqh, 66.667cqw))` }}
            >
              <motion.div ref={demo} className="w-full max-w-2xl" style={{ y: demoY }}>
                <HeroDemo playing={inView} />
              </motion.div>
            </div>
          </motion.div>
        </motion.div>
      </div>
    </div>
  );
}
