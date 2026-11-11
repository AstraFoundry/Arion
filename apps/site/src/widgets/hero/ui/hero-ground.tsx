import type { CSSProperties } from "react";

/**
 * The 3:2 box the photograph fills. It covers the stage (a size container),
 * keeps its top edge on the stage's top edge and shows the frame from 60%
 * across, so the flowers stay at the same height in it whatever the stage's
 * shape.
 */
export const GROUND_BOX: CSSProperties = {
  top: 0,
  width: "max(100cqw, 150cqh)",
  height: "max(100cqh, 66.667cqw)",
  left: "calc((100cqw - max(100cqw, 150cqh)) * 0.6)",
};

/**
 * The hero's photograph (public/backdrops, credited on /credits), a hillside
 * of pink flowers in mist the demo floats on. Decorative: it carries no alt.
 */
export function HeroGround() {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- static export, pre-sized webp
    <img
      src="/backdrops/hillside-1600.webp"
      srcSet="/backdrops/hillside-960.webp 960w, /backdrops/hillside-1600.webp 1600w, /backdrops/hillside-2400.webp 2400w"
      sizes="max(100vw, 150svh)"
      alt=""
      aria-hidden="true"
      fetchPriority="high"
      loading="eager"
      className="pointer-events-none absolute max-w-none select-none"
      style={GROUND_BOX}
    />
  );
}
