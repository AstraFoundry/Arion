"use client";

import { Pause, Play } from "lucide-react";
import { type CSSProperties, useState } from "react";
import type { Testimonial } from "@/entities/testimonial";
import { sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn } from "@/shared/lib";
import { Container } from "@astrafoundry/design/ui";
import { SectionHeading } from "@/shared/ui";
import { TestimonialCard } from "./testimonial-card";

/** Seconds one row takes to move by one copy of its cards. */
const SECONDS_PER_CARD = 9;

/** Alternate cards between two rows so neighbours differ in length. */
function rowsOf(testimonials: readonly Testimonial[]): Testimonial[][] {
  const rows: Testimonial[][] = [[], []];
  testimonials.forEach((testimonial, index) => rows[index % 2].push(testimonial));
  return rows.filter((row) => row.length > 0);
}

function Cards({ row, copy }: { row: readonly Testimonial[]; copy: "first" | "repeat" }) {
  // The repeat only closes the loop: screen readers and the tab order skip it.
  const repeat = copy === "repeat";
  return (
    <ul
      className={cn("flex shrink-0", repeat && "motion-reduce:hidden")}
      aria-hidden={repeat || undefined}
      inert={repeat || undefined}
    >
      {row.map((testimonial) => (
        <li key={testimonial.handle} className="pe-4">
          <TestimonialCard testimonial={testimonial} />
        </li>
      ))}
    </ul>
  );
}

/**
 * Two rows of posts drifting in opposite directions. Hovering or focusing a
 * row pauses it, and the button pauses both (WCAG 2.2.2). With reduced motion
 * the rows stand still and scroll sideways instead.
 */
export function TestimonialWall({ testimonials }: { testimonials: readonly Testimonial[] }) {
  const copy = useCopy();
  const [paused, setPaused] = useState(false);
  const rows = rowsOf(testimonials);

  return (
    <section
      id={sections.testimonials}
      className="scroll-mt-20 overflow-hidden py-24 sm:py-32"
      data-paused={paused || undefined}
    >
      <Container className="flex flex-wrap items-end justify-between gap-6">
        <SectionHeading title={copy.testimonials.title} />
        <button
          type="button"
          onClick={() => setPaused((value) => !value)}
          aria-label={paused ? copy.testimonials.play : copy.testimonials.pause}
          title={paused ? copy.testimonials.play : copy.testimonials.pause}
          className="grid size-10 shrink-0 place-items-center rounded-full border border-border bg-card text-foreground shadow-1 transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring motion-reduce:hidden"
        >
          {paused ? <Play className="size-4" /> : <Pause className="size-4" />}
        </button>
      </Container>

      <div className="mx-auto mt-12 flex max-w-7xl flex-col gap-4 mask-x-from-90% mask-x-to-100% motion-reduce:mask-none">
        {rows.map((row, index) => (
          <div
            key={index}
            className="group/row flex overflow-hidden motion-reduce:overflow-x-auto motion-reduce:px-5 sm:motion-reduce:px-8"
          >
            <div
              className={cn(
                "flex w-max animate-ticker items-stretch motion-reduce:animate-none",
                "group-hover/row:[animation-play-state:paused] group-focus-within/row:[animation-play-state:paused]",
                "in-data-paused:[animation-play-state:paused]",
                index % 2 === 1 && "[animation-direction:reverse]",
              )}
              style={{ "--ticker-duration": `${row.length * SECONDS_PER_CARD}s` } as CSSProperties}
            >
              <Cards row={row} copy="first" />
              <Cards row={row} copy="repeat" />
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
