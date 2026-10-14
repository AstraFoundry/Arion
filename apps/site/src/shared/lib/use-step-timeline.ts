"use client";

import { useReducedMotion } from "motion/react";
import { useCallback, useEffect, useState } from "react";

/**
 * Drives a scripted demo through `delays.length + 1` steps. Step 0 is the
 * initial state; step `n` is reached `delays[n - 1]` ms after step `n - 1`.
 *
 * With reduced motion the timeline reports the final step as soon as it is
 * running, so the reader sees the finished state without animation.
 */
export function useStepTimeline(delays: readonly number[], running: boolean) {
  const reduce = useReducedMotion() ?? false;
  const last = delays.length;
  const [step, setStep] = useState(0);

  useEffect(() => {
    if (!running || reduce || step >= last) return;
    const timer = window.setTimeout(() => setStep((value) => value + 1), delays[step]);
    return () => window.clearTimeout(timer);
  }, [delays, last, reduce, running, step]);

  const replay = useCallback(() => setStep(0), []);

  /** Moves the timeline to `target`; it keeps playing from there. */
  const jump = useCallback(
    (target: number) => setStep(Math.max(0, Math.min(target, last))),
    [last],
  );

  const current = running && reduce ? last : step;
  return { step: current, done: current >= last, replay, jump, reduce };
}
