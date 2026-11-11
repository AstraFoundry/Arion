"use client";

import { sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn, useInViewOnce } from "@/shared/lib";
import { Container, SectionHeading } from "@/shared/ui";
import { CHAPTER_SPANS, CHAPTERS, type ChapterId } from "../config/chapters";
import { ChapterMock } from "./chapter-mock";
import { GridGround } from "./grid-ground";

/**
 * One highlight: a Sunbeam backdrop card. The mock floats on the shared
 * photograph and the title is its caption, bottom-left at the card inset,
 * over the ink scrim.
 * `clip-path` (not `overflow`) clips the photograph, so from `lg` up, where
 * the image is positioned against the grid rather than the card, each card
 * still shows only its own part of it.
 */
function ChapterCard({ id }: { id: ChapterId }) {
  const { tour } = useCopy();
  const [ref, inView] = useInViewOnce<HTMLDivElement>();
  return (
    <li
      className={cn(
        "relative flex h-[34rem] flex-col [clip-path:inset(0_round_1.5rem)] lg:static lg:h-[38rem] lg:[clip-path:inset(0_round_2.5rem)]",
        CHAPTER_SPANS[id] === 2 && "lg:col-span-2",
      )}
    >
      <GridGround />
      <div
        ref={ref}
        className="flex min-h-0 flex-1 items-center justify-center px-6 pt-6 sm:px-10 sm:pt-10 lg:px-12 lg:pt-12"
      >
        <ChapterMock id={id} playing={inView} />
      </div>
      <div className="bg-linear-to-t from-scrim/55 to-transparent px-6 pt-12 pb-6 sm:px-10 sm:pb-10 lg:px-12 lg:pb-12">
        <h3 className="max-w-[40ch] text-balance text-2xl leading-[1.35] font-normal text-substrate-tint-muted">
          {tour[id].title}
        </h3>
      </div>
    </li>
  );
}

/**
 * The product's highlights as a grid of backdrop cards that are windows onto
 * one photograph: a narrow card beside a wide one, then three narrow ones.
 */
export function FeatureTour() {
  const { tour } = useCopy();
  return (
    <section id={sections.values} className="scroll-mt-20 py-24 sm:py-32">
      <Container size="wide">
        <SectionHeading title={tour.title} lead={tour.lead} align="split" />
        <ul className="mt-14 grid gap-3 sm:gap-4 sm:mt-16 lg:relative lg:grid-cols-3">
          {CHAPTERS.map((id) => (
            <ChapterCard key={id} id={id} />
          ))}
        </ul>
      </Container>
    </section>
  );
}
