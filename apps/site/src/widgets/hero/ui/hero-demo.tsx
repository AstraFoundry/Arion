"use client";

import { RotateCcw } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useCopy } from "@/shared/i18n";
import { useInViewOnce, useStepTimeline } from "@/shared/lib";
import { HERO_DELAYS, HERO_STEP, partialText } from "../config/timeline";
import { DictationPill } from "./dictation-pill";

function Caret() {
  return (
    <span
      aria-hidden="true"
      className="ml-px inline-block h-[1.1em] w-[2px] translate-y-[0.15em] animate-caret bg-foreground"
    />
  );
}

/**
 * The hero illustration: a document window and, below it, the dictation
 * pill listening, streaming the draft, then reporting the insertion. It is
 * a picture of the product, so it is inert and labelled as an image.
 */
export function HeroDemo() {
  const copy = useCopy();
  const { demo } = copy;
  const [ref, inView] = useInViewOnce<HTMLDivElement>();
  const { step, done, replay, reduce } = useStepTimeline(HERO_DELAYS, inView);

  const inserted = step >= HERO_STEP.inserted;
  const pillVisible = step >= HERO_STEP.listening;

  return (
    <div ref={ref} className="relative">
      <div role="img" aria-label={copy.hero.demoLabel}>
        <div inert className="flex flex-col items-center gap-6 select-none">
          <div className="w-full max-w-2xl overflow-hidden rounded-2xl border border-border bg-card text-left text-card-foreground shadow-4">
            <div className="flex h-10 items-center gap-1.5 border-b border-border px-4">
              <span className="size-2.5 rounded-full bg-muted" />
              <span className="size-2.5 rounded-full bg-muted" />
              <span className="size-2.5 rounded-full bg-muted" />
            </div>
            <div className="flex min-h-56 flex-col gap-3 px-6 py-6 sm:px-10 sm:py-8">
              <p className="font-display text-2xl">{demo.editorTitle}</p>
              <ul className="flex list-disc flex-col gap-1.5 pl-5 text-base text-muted-foreground">
                {demo.editorLines.map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
              <p className="text-base">
                {inserted ? (
                  <span className="relative">
                    <motion.span
                      aria-hidden="true"
                      className="absolute -inset-x-0.5 inset-y-0 rounded-sm bg-accent"
                      initial={{ opacity: 1 }}
                      animate={{ opacity: 0 }}
                      transition={{ duration: 1.2, delay: 0.2 }}
                    />
                    <span className="relative">{demo.sentence}</span>
                  </span>
                ) : null}
                <Caret />
              </p>
            </div>
          </div>

          <div className="flex h-11 w-full justify-center">
            <AnimatePresence>
              {pillVisible ? (
                <motion.div
                  key="pill"
                  className="flex w-full justify-center"
                  initial={{ opacity: 0, scale: 0.86, y: 14 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.92, y: 10 }}
                  transition={{ type: "spring", stiffness: 380, damping: 30 }}
                >
                  <DictationPill
                    state={
                      inserted
                        ? { kind: "inserted" }
                        : { kind: "listening", draft: partialText(demo.sentence, step) }
                    }
                  />
                </motion.div>
              ) : null}
            </AnimatePresence>
          </div>
        </div>
      </div>

      {done && !reduce ? (
        <button
          type="button"
          onClick={replay}
          className="absolute right-0 -bottom-12 flex items-center gap-1.5 rounded-full bg-card px-3 py-1.5 text-sm text-foreground shadow-2 transition-colors hover:bg-muted"
        >
          <RotateCcw className="size-3.5" aria-hidden="true" />
          {copy.common.replay}
        </button>
      ) : null}
    </div>
  );
}
