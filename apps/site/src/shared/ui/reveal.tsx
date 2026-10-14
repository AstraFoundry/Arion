"use client";

import { motion } from "motion/react";
import type { ReactNode } from "react";

interface RevealProps {
  children: ReactNode;
  className?: string;
  /** Seconds to wait after the element enters the viewport. */
  delay?: number;
}

const EASE_OUT = [0.22, 1, 0.36, 1] as const;

/**
 * One-shot entrance for section content: a short rise and fade the first time
 * it scrolls into view. With reduced motion, the app's MotionConfig skips the
 * rise and keeps the fade. The element is the same either way: the static HTML
 * carries the hidden state, and hydration does not patch a differing element.
 */
export function Reveal({ children, className, delay = 0 }: RevealProps) {
  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 16 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, amount: 0.25 }}
      transition={{ duration: 0.6, ease: EASE_OUT, delay }}
    >
      {children}
    </motion.div>
  );
}
