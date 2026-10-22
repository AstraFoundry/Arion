"use client";

import { ArrowDown } from "lucide-react";
import { motion } from "motion/react";
import { useCopy } from "@/shared/i18n";
import { useStepTimeline } from "@/shared/lib";
import { MockWindow } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

const STEP = { polishing: 1, typed: 2 } as const;

/** What was said, with fillers and a spoken correction, and what was typed. */
export function PolishMock({ playing }: { playing: boolean }) {
  const { tour } = useCopy();
  const copy = tour.polish;
  const { step } = useStepTimeline(MOCK_DELAYS.polish, playing);

  return (
    <div className="flex w-full max-w-md flex-col items-center gap-3">
      <div className="w-full rounded-2xl border border-substrate-edge bg-substrate px-5 py-4 text-substrate-ink shadow-4">
        <span className="text-xs text-substrate-muted">{copy.heard}</span>
        <p className="mt-1.5 text-sm leading-relaxed">{copy.spoken}</p>
      </div>
      <motion.span
        aria-hidden="true"
        className="flex size-8 items-center justify-center rounded-full bg-card text-foreground shadow-2"
        animate={step === STEP.polishing ? { y: [0, 3, 0] } : { y: 0 }}
        transition={step === STEP.polishing ? { duration: 0.8, repeat: Infinity } : undefined}
      >
        <ArrowDown className="size-4" />
      </motion.span>
      <MockWindow>
        <div className="min-h-24 px-5 py-4">
          <span className="text-xs text-muted-foreground">{copy.typed}</span>
          {step >= STEP.typed ? (
            <motion.p
              className="mt-1.5 text-base leading-relaxed"
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              {copy.polished}
            </motion.p>
          ) : null}
        </div>
      </MockWindow>
    </div>
  );
}
