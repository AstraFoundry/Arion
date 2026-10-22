"use client";

import { Wifi, WifiOff } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useCopy } from "@/shared/i18n";
import { useStepTimeline } from "@/shared/lib";
import { BrandMark, BrandWord, DictationPill } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

const STEP = { networkOff: 1, inserted: 2 } as const;

/** The menu bar with the network off and Arion's panel still taking dictation. */
export function OfflineMock({ playing }: { playing: boolean }) {
  const { tour } = useCopy();
  const copy = tour.offline;
  const { step } = useStepTimeline(MOCK_DELAYS.offline, playing);
  const offline = step >= STEP.networkOff;
  const inserted = step >= STEP.inserted;
  const NetworkIcon = offline ? WifiOff : Wifi;

  return (
    <div className="flex w-full max-w-sm flex-col items-end gap-2">
      <div className="flex h-8 w-full items-center justify-end gap-3 rounded-lg bg-card px-3 text-card-foreground shadow-2">
        <AnimatePresence mode="popLayout" initial={false}>
          <motion.span
            key={offline ? "off" : "on"}
            initial={{ opacity: 0, scale: 0.6 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.6 }}
            transition={{ type: "spring", stiffness: 500, damping: 30 }}
            className="flex items-center gap-1.5 text-xs text-muted-foreground"
          >
            <NetworkIcon className="size-3.5 text-foreground" aria-hidden="true" />
            {offline ? copy.networkOff : copy.networkOn}
          </motion.span>
        </AnimatePresence>
        <BrandMark className="size-4 rounded-[4px]" />
      </div>

      <div className="flex w-full flex-col gap-4 rounded-xl border border-border bg-card p-4 text-card-foreground shadow-4">
        <div className="flex items-center gap-2.5">
          <BrandMark className="size-6 rounded-md" />
          <BrandWord className="text-lg" />
        </div>
        <div className="flex items-baseline justify-between gap-3 text-sm">
          <span className="text-muted-foreground">{copy.model}</span>
          <span>{copy.modelChoice}</span>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="text-xs text-muted-foreground">{copy.lastDictation}</span>
          <p className="min-h-[3lh] rounded-lg bg-muted px-3 py-2 text-sm leading-relaxed">
            {inserted ? (
              <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                {copy.sentence}
              </motion.span>
            ) : (
              <span className="text-muted-foreground">{copy.empty}</span>
            )}
          </p>
        </div>
      </div>

      <div className="flex h-11 w-full justify-center pt-2">
        {inserted ? (
          <motion.div
            initial={{ opacity: 0, scale: 0.86, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 380, damping: 30 }}
          >
            <DictationPill state={{ kind: "inserted" }} />
          </motion.div>
        ) : null}
      </div>
    </div>
  );
}
