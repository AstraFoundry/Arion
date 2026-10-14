"use client";

import { useInView } from "motion/react";
import { useRef } from "react";

/**
 * Reports once when the element has been at least `amount` visible. Product
 * demos use it so their scripted timelines start when the reader arrives.
 */
export function useInViewOnce<T extends Element>(amount = 0.35) {
  const ref = useRef<T>(null);
  const inView = useInView(ref, { once: true, amount });
  return [ref, inView] as const;
}
