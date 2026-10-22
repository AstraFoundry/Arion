"use client";

import { motion } from "motion/react";
import { siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn, useStepTimeline } from "@/shared/lib";
import { MockWindow } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

const STEP = { pinyin: 1, committed: 2, switched: 3 } as const;
/** Each half of the badge flip, matching the input method's 0.12 s. */
const FLIP_HALF = 0.12;

/**
 * The language badge by the caret. Like the native one, it appears for a
 * switch and turns over around the horizontal axis: the mode just left folds
 * away, then the new one unfolds. Under reduced motion it shows the new mode.
 */
function ModeBadge({ from, to }: { from: string; to: string }) {
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

/** Full-Pinyin typing with the candidate bar, then Shift switching to English. */
export function KeyboardMock({ playing }: { playing: boolean }) {
  const { tour } = useCopy();
  const copy = tour.keyboard;
  const { step } = useStepTimeline(MOCK_DELAYS.keyboard, playing);
  const composing = step < STEP.committed;
  const switched = step >= STEP.switched;

  return (
    <div className="flex w-full max-w-md flex-col items-center gap-6">
      <MockWindow title={copy.document}>
        <div className="min-h-40 px-6 py-7 text-base">
          {/* The candidate bar opens under the composition and the badge
              under the caret, as the input method places them. */}
          <span className="relative">
            {composing ? (
              <span className="underline decoration-foreground/60 underline-offset-4">
                {step >= STEP.pinyin ? copy.pinyin : copy.pinyinStart}
              </span>
            ) : (
              <span>{copy.committed}</span>
            )}
            {step === STEP.pinyin ? (
              <motion.span
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.15 }}
                className="absolute top-full left-0 mt-2 flex h-[30px] items-center gap-4 rounded-[10px] border border-substrate-edge bg-substrate px-2 text-sm whitespace-nowrap text-substrate-ink shadow-3"
              >
                {copy.candidates.map((candidate, index) => (
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
            ) : null}
          </span>
          <span className="relative">
            <span
              aria-hidden="true"
              className="ml-px inline-block h-[1.1em] w-[2px] translate-y-[0.15em] animate-caret bg-foreground"
            />
            {switched ? (
              <span className="absolute top-full left-0 mt-1">
                <ModeBadge from={copy.chinese} to={copy.english} />
              </span>
            ) : null}
          </span>
        </div>
      </MockWindow>
      <kbd
        className={cn(
          "rounded-lg border border-border bg-card px-4 py-2 font-sans text-sm text-foreground shadow-2 transition-transform duration-150",
          switched && "translate-y-px shadow-1",
        )}
      >
        {siteConfig.languageSwitchKey}
      </kbd>
    </div>
  );
}
