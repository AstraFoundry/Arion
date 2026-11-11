"use client";

import { motion } from "motion/react";
import { cn } from "@/shared/lib";

/** Each half of the badge flip, matching the input method's 0.12 s. */
const FLIP_HALF = 0.12;

/**
 * The language badge by the caret. Like the native one, it appears for a
 * switch and turns over around the horizontal axis: the mode just left folds
 * away, then the new one unfolds. Under reduced motion it shows the new mode.
 */
export function ModeBadge({ from, to }: { from: string; to: string }) {
  const face = "absolute inset-0 flex items-center justify-center [backface-visibility:hidden]";
  return (
    <span className="relative inline-flex size-[23px] items-center justify-center rounded-[5px] border border-substrate-edge bg-substrate text-[15px] font-medium text-substrate-hi [perspective:60px]">
      <motion.span
        className={face}
        initial={{ rotateX: 0, opacity: 1 }}
        animate={{ rotateX: 90, opacity: 0 }}
        transition={{
          rotateX: { duration: FLIP_HALF, ease: "easeIn" },
          opacity: { duration: 0, delay: FLIP_HALF },
        }}
      >
        {from}
      </motion.span>
      <motion.span
        className={face}
        initial={{ rotateX: -90 }}
        animate={{ rotateX: 0 }}
        transition={{ delay: FLIP_HALF, duration: FLIP_HALF, ease: "easeOut" }}
      >
        {to}
      </motion.span>
    </span>
  );
}

/**
 * The input method's candidate bar, opened under the composition with the
 * first candidate selected. Position it from the composition it belongs to.
 */
export function CandidateBar({
  candidates,
  className,
}: {
  candidates: readonly string[];
  className?: string;
}) {
  return (
    <motion.span
      initial={{ opacity: 0, y: -4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.15 }}
      className={cn(
        "flex h-[30px] items-center gap-4 rounded-[10px] border border-substrate-edge bg-substrate px-2 text-sm whitespace-nowrap text-substrate-ink",
        className,
      )}
    >
      {candidates.map((candidate, index) => (
        <span
          key={candidate}
          className={cn(
            "flex items-center gap-1 rounded-md px-1.5 py-0.5",
            index === 0 && "bg-substrate-raised text-substrate-hi",
          )}
        >
          <span className="text-xs text-substrate-muted">{index + 1}</span>
          {candidate}
        </span>
      ))}
    </motion.span>
  );
}
