"use client";

import { Check } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";

const SETTLE = { type: "spring", stiffness: 420, damping: 34 } as const;
const BAR_HEIGHTS = [6, 10, 16, 12, 20, 14, 9, 17, 11, 7, 5];

function LevelBars() {
  return (
    <span className="flex h-5 items-center gap-[3px]" aria-hidden="true">
      {BAR_HEIGHTS.map((height, index) => (
        <motion.span
          key={index}
          className="w-[3px] rounded-full bg-substrate-ink"
          initial={{ height: 3 }}
          animate={{ height: [3, height, 4, height * 0.7, 3] }}
          transition={{
            duration: 1.1,
            repeat: Infinity,
            ease: "easeInOut",
            delay: index * 0.06,
          }}
        />
      ))}
    </span>
  );
}

function Keycap({ children }: { children: string }) {
  return (
    <kbd className="rounded-md border border-substrate-edge bg-substrate-raised px-1.5 py-0.5 font-sans text-xs text-substrate-ink">
      {children}
    </kbd>
  );
}

export type PillState =
  | { kind: "listening"; draft: string }
  | { kind: "inserted" };

/** The floating dictation pill, drawn like the desktop overlay's. */
export function DictationPill({ state }: { state: PillState }) {
  const { demo } = useCopy();
  const live = state.kind === "listening";

  return (
    <motion.div
      layout
      transition={SETTLE}
      className="flex h-11 items-center gap-2.5 rounded-full border border-substrate-edge bg-substrate px-4 text-sm text-substrate-ink shadow-4"
      style={{ width: live ? 380 : "auto", maxWidth: "100%" }}
    >
      <AnimatePresence mode="popLayout" initial={false}>
        {live ? (
          <motion.span
            key="listening"
            className="flex min-w-0 flex-1 items-center gap-2.5"
            initial={{ opacity: 0, scale: 0.94 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.97 }}
          >
            <span className="flex shrink-0 items-center gap-2.5">
              <span className="size-2 animate-pulse rounded-full bg-danger" />
              <LevelBars />
            </span>
            {state.draft ? (
              <span className="min-w-0 flex-1 truncate text-right [direction:rtl]">
                <bdi>{state.draft}</bdi>
              </span>
            ) : (
              <span className="ml-auto flex items-center gap-1.5 text-xs text-substrate-muted">
                {siteConfig.shortcut.map((key) => (
                  <Keycap key={key}>{key}</Keycap>
                ))}
                {demo.finishHint}
              </span>
            )}
          </motion.span>
        ) : (
          <motion.span
            key="inserted"
            className="flex items-center gap-2"
            initial={{ opacity: 0, scale: 0.94 }}
            animate={{ opacity: 1, scale: 1 }}
          >
            <Check className="size-4 text-success" aria-hidden="true" />
            {demo.inserted}
          </motion.span>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
