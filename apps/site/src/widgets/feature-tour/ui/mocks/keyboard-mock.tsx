"use client";

import { siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn, useStepTimeline } from "@/shared/lib";
import { CandidateBar, MockWindow, ModeBadge } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

const STEP = { pinyin: 1, committed: 2, switched: 3 } as const;
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
              <CandidateBar
                candidates={copy.candidates}
                className="absolute top-full left-0 mt-2"
              />
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
          "rounded-lg border border-border bg-card px-4 py-2 font-sans text-sm text-foreground transition-transform duration-150",
          switched && "translate-y-px",
        )}
      >
        {siteConfig.languageSwitchKey}
      </kbd>
    </div>
  );
}
