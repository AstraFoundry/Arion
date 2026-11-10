"use client";

import { AnimatePresence, motion } from "motion/react";
import type { ReactNode } from "react";
import { siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn, useStepTimeline } from "@/shared/lib";
import { CandidateBar, DictationPill, MockWindow, ModeBadge, type PillState } from "@/shared/ui";
import {
  HERO_DELAYS,
  HERO_LOOP_REST,
  HERO_STEP as S,
  halfText,
  partialText,
} from "../config/timeline";

function Caret({ children }: { children?: ReactNode }) {
  return (
    <span className="relative">
      <span
        aria-hidden="true"
        className="ml-px inline-block h-[1.1em] w-[2px] translate-y-[0.15em] animate-caret bg-foreground"
      />
      {children}
    </span>
  );
}

/**
 * One line of the document. Its finished text is laid out invisibly
 * underneath, so the document keeps its size while lines are typed.
 */
function Line({ full, children }: { full: ReactNode; children: ReactNode }) {
  return (
    <p className="grid">
      <span aria-hidden="true" className="invisible col-start-1 row-start-1">
        {full}
      </span>
      <span className="col-start-1 row-start-1">{children}</span>
    </p>
  );
}

/** Just typed: the text flashes the accent wash, then settles. */
function Typed({ children }: { children: ReactNode }) {
  return (
    <span className="relative">
      <motion.span
        aria-hidden="true"
        className="absolute -inset-x-0.5 inset-y-0 rounded-sm bg-accent"
        initial={{ opacity: 1 }}
        animate={{ opacity: 0 }}
        transition={{ duration: 1.2, delay: 0.2 }}
      />
      <span className="relative">{children}</span>
    </span>
  );
}

/**
 * The hero illustration: a document of Shakespeare quotes. Each loop shows
 * the highlights in turn — a dictation (mixed-language in zh and ja), a
 * correction Arion learns, a polished dictation and Pinyin typing with a
 * Shift switch — then rests and starts over. It is a picture of the product,
 * so it is inert and labelled as an image.
 */
export function HeroDemo({ playing }: { playing: boolean }) {
  const copy = useCopy();
  const { demo } = copy;
  const { keyboard } = copy.tour;
  const { step } = useStepTimeline(HERO_DELAYS, playing, { loopAfter: HERO_LOOP_REST });

  const [learnBefore, learnAfter] = demo.learnSentence.split("{term}");
  const learnSentence = (term: ReactNode) => (
    <>
      {learnBefore}
      {term}
      {learnAfter}
    </>
  );

  const pill: PillState | null =
    step >= S.listening && step < S.inserted
      ? { kind: "listening", draft: partialText(demo.sentence, step) }
      : step === S.inserted
        ? { kind: "inserted" }
        : step === S.learned
          ? { kind: "learned", term: demo.learnRight }
          : step === S.polishHeard
            ? { kind: "listening", draft: halfText(demo.spoken) }
            : step === S.polishHeardAll
              ? { kind: "listening", draft: demo.spoken }
              : step === S.polishing
                ? { kind: "polishing" }
                : step === S.polished
                  ? { kind: "inserted" }
                  : null;
  const typingPinyin = step >= S.pinyinStart;

  return (
    <div role="img" aria-label={copy.hero.demoLabel}>
      <div inert className="flex flex-col items-stretch select-none">
        <MockWindow>
          <div className="flex flex-col gap-2.5 px-6 pt-5 pb-12 text-sm leading-relaxed sm:px-10 sm:pt-7 sm:text-base">
            <p className="font-display text-xl sm:text-2xl">{demo.editorTitle}</p>

            <Line full={demo.sentence}>
              {step >= S.inserted ? <Typed>{demo.sentence}</Typed> : null}
              {step < S.learnTyped ? <Caret /> : null}
            </Line>

            <Line full={learnSentence(demo.learnWrong)}>
              {step >= S.learnTyped
                ? learnSentence(
                    <span
                      className={cn(
                        "-mx-0.5 rounded-sm px-0.5 transition-colors duration-200",
                        step === S.learnSelected && "bg-accent text-accent-foreground",
                      )}
                    >
                      <AnimatePresence mode="popLayout" initial={false}>
                        <motion.span
                          key={step >= S.learnCorrected ? "right" : "wrong"}
                          className="inline-block"
                          initial={{ opacity: 0, y: 6 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: -6 }}
                          transition={{ duration: 0.22 }}
                        >
                          {step >= S.learnCorrected ? demo.learnRight : demo.learnWrong}
                        </motion.span>
                      </AnimatePresence>
                    </span>,
                  )
                : null}
              {step >= S.learnTyped && step < S.polishHeard ? <Caret /> : null}
            </Line>

            <Line full={demo.polished}>
              {step >= S.polished ? <Typed>{demo.polished}</Typed> : null}
              {step >= S.polishHeard && step < S.pinyinStart ? <Caret /> : null}
            </Line>

            <Line full={demo.committed}>
              {typingPinyin ? (
                <span className="relative">
                  {step < S.pinyinCommitted ? (
                    <span className="underline decoration-foreground/60 underline-offset-4">
                      {step >= S.pinyin ? demo.pinyin : demo.pinyinStart}
                    </span>
                  ) : (
                    <span>{demo.committed}</span>
                  )}
                  {step === S.pinyin ? (
                    <CandidateBar
                      candidates={demo.candidates}
                      className="absolute top-full left-0 z-10 mt-2"
                    />
                  ) : null}
                </span>
              ) : null}
              {typingPinyin ? (
                <Caret>
                  {step >= S.switched ? (
                    <span className="absolute top-full left-0 mt-1">
                      <ModeBadge from={keyboard.chinese} to={keyboard.english} />
                    </span>
                  ) : null}
                </Caret>
              ) : null}
            </Line>
          </div>
        </MockWindow>

        <div className="mt-3 flex h-11 justify-center">
          <AnimatePresence mode="popLayout">
            {pill ? (
              <motion.div
                key="pill"
                className="flex w-full justify-center"
                initial={{ opacity: 0, scale: 0.86, y: 14 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.92, y: 10 }}
                transition={{ type: "spring", stiffness: 380, damping: 30 }}
              >
                <DictationPill state={pill} />
              </motion.div>
            ) : typingPinyin ? (
              <motion.kbd
                key="shift"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 6 }}
                className={cn(
                  "self-center rounded-lg border border-border bg-card px-4 py-2 font-sans text-sm text-foreground transition-transform duration-150",
                  step >= S.switched && "translate-y-px",
                )}
              >
                {siteConfig.languageSwitchKey}
              </motion.kbd>
            ) : null}
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
