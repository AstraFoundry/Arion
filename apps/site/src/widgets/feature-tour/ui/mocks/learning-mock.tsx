"use client";

import { AnimatePresence, motion } from "motion/react";
import { useCopy } from "@/shared/i18n";
import { cn, useStepTimeline } from "@/shared/lib";
import { DictationPill, MockWindow } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

const STEP = { selected: 1, corrected: 2, learned: 3 } as const;

/** A misrecognized name corrected in place, then learned with an undo. */
export function LearningMock({ playing }: { playing: boolean }) {
  const { tour } = useCopy();
  const copy = tour.learning;
  const { step } = useStepTimeline(MOCK_DELAYS.learning, playing);
  const corrected = step >= STEP.corrected;
  const [before, after] = copy.sentence.split("{term}");

  return (
    <div className="flex w-full max-w-md flex-col items-center gap-5">
      <MockWindow title={copy.document}>
        <div className="px-6 py-7 text-base leading-relaxed">
          {before}
          <span
            className={cn(
              "relative -mx-0.5 inline-block rounded-sm px-0.5 transition-colors duration-200",
              step === STEP.selected && "bg-accent text-accent-foreground",
            )}
          >
            <AnimatePresence mode="popLayout" initial={false}>
              <motion.span
                key={corrected ? "right" : "wrong"}
                className="inline-block"
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -6 }}
                transition={{ duration: 0.22 }}
              >
                {corrected ? copy.right : copy.wrong}
              </motion.span>
            </AnimatePresence>
          </span>
          {after}
        </div>
      </MockWindow>
      <div className="flex h-11 w-full justify-center">
        {step >= STEP.learned ? (
          <motion.div
            initial={{ opacity: 0, scale: 0.86, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 380, damping: 30 }}
          >
            <DictationPill state={{ kind: "learned", term: copy.right }} />
          </motion.div>
        ) : null}
      </div>
    </div>
  );
}
