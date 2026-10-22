"use client";

import { motion } from "motion/react";
import { useCopy } from "@/shared/i18n";
import { useStepTimeline } from "@/shared/lib";
import { DictationPill, MockWindow } from "@/shared/ui";
import { MOCK_DELAYS } from "../../config/chapters";

/**
 * A chat composer receiving a mixed Chinese and English sentence. Streaming
 * hypotheses stay a revisable draft in the pill; only the final sentence is
 * typed into the field.
 */
export function MixedLanguageMock({ playing }: { playing: boolean }) {
  const { tour } = useCopy();
  const copy = tour.mixed;
  const { step, done } = useStepTimeline(MOCK_DELAYS.mixed, playing);
  const draft = step > 0 && !done ? copy.partials[step - 1] : "";

  return (
    <div className="flex w-full max-w-md flex-col items-center gap-5">
      <MockWindow title={copy.channel}>
        <div className="flex flex-col gap-5 px-5 py-5">
          <div className="flex gap-3">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-sm">
              {Array.from(copy.author)[0]}
            </span>
            <div className="flex min-w-0 flex-col gap-0.5">
              <span className="text-sm font-medium">{copy.author}</span>
              <p className="text-sm leading-relaxed text-muted-foreground">{copy.message}</p>
            </div>
          </div>
          <div className="min-h-[4.5rem] rounded-xl border border-border px-3.5 py-3 text-sm leading-relaxed">
            {done ? (
              <motion.span
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3 }}
              >
                {copy.sentence}
              </motion.span>
            ) : (
              <span className="text-muted-foreground">{copy.composer}</span>
            )}
          </div>
        </div>
      </MockWindow>
      <div className="flex h-11 w-full justify-center">
        {step > 0 ? (
          <DictationPill
            state={done ? { kind: "inserted" } : { kind: "listening", draft }}
          />
        ) : null}
      </div>
    </div>
  );
}
