"use client";

import { AnimatePresence, motion, useInView, type Variants } from "motion/react";
import { useRef } from "react";
import { sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn, useInViewOnce } from "@/shared/lib";
import { Container, SectionHeading } from "@/shared/ui";
import { backdropSrc, CHAPTER_ICONS, CHAPTERS, type ChapterId } from "../config/chapters";
import { useActiveChapter } from "../lib/use-active-chapter";
import { ChapterMock } from "./chapter-mock";
import { StageBackdrop, TourStage } from "./tour-stage";

/** The stage turns a page per chapter: the mock leaves the way the reader scrolls. */
const PAGE: Variants = {
  enter: (direction: number) => ({ opacity: 0, y: direction * 56, scale: 0.97 }),
  center: { opacity: 1, y: 0, scale: 1 },
  exit: (direction: number) => ({ opacity: 0, y: direction * -56, scale: 0.97 }),
};
const PAGE_TURN = { type: "spring", stiffness: 260, damping: 32 } as const;

/** Below `lg` each chapter carries its own stage above its copy. */
function InlineStage({ id }: { id: ChapterId }) {
  const [ref, inView] = useInViewOnce<HTMLDivElement>();
  return (
    <div ref={ref} className="lg:hidden">
      <TourStage className="flex min-h-[28rem] items-center justify-center px-5 py-10 sm:px-10">
        <StageBackdrop src={backdropSrc(id)} />
        <ChapterMock id={id} playing={inView} />
      </TourStage>
    </div>
  );
}

/**
 * The product's highlights as a scrolling tour: on wide screens a sticky
 * stage on the left, the photograph and the active chapter's mock, turns
 * its page as the chapter copy on the right scrolls past the middle.
 */
export function FeatureTour() {
  const { tour } = useCopy();
  const { index: active, direction, register } = useActiveChapter();
  const stageRef = useRef<HTMLDivElement>(null);
  const stageInView = useInView(stageRef, { amount: 0.4 });
  const activeId = CHAPTERS[active];

  return (
    <section id={sections.values} className="scroll-mt-20 py-24 sm:py-32">
      <Container>
        <SectionHeading title={tour.title} />
        <div className="mt-14 lg:mt-6 lg:grid lg:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)] lg:gap-16">
          <div className="hidden lg:block">
            <div ref={stageRef} className="sticky top-24 h-[min(calc(100svh-8rem),46rem)] py-2">
              <TourStage className="size-full">
                {CHAPTERS.map((id) => (
                  <StageBackdrop
                    key={id}
                    src={backdropSrc(id)}
                    className={cn(
                      "transition-opacity duration-700",
                      id === activeId ? "opacity-100" : "opacity-0",
                    )}
                  />
                ))}
                <AnimatePresence mode="popLayout" custom={direction} initial={false}>
                  <motion.div
                    key={activeId}
                    className="absolute inset-0 flex items-center justify-center p-10 xl:p-14"
                    custom={direction}
                    variants={PAGE}
                    initial="enter"
                    animate="center"
                    exit="exit"
                    transition={PAGE_TURN}
                  >
                    <ChapterMock id={activeId} playing={stageInView} />
                  </motion.div>
                </AnimatePresence>
              </TourStage>
            </div>
          </div>

          <ol className="flex flex-col gap-20 lg:gap-0">
            {CHAPTERS.map((id, index) => {
              const chapter = tour[id];
              const Icon = CHAPTER_ICONS[id];
              return (
                <li
                  key={id}
                  ref={register(index)}
                  data-chapter={index}
                  className="flex flex-col gap-8 lg:min-h-[80svh] lg:justify-center"
                >
                  <InlineStage id={id} />
                  <div
                    className={cn(
                      "flex max-w-md flex-col gap-3 transition-opacity duration-500",
                      index !== active && "lg:opacity-40",
                    )}
                  >
                    <Icon className="size-5 text-foreground" aria-hidden="true" />
                    <h3 className="text-balance font-display text-2xl leading-tight sm:text-3xl">
                      {chapter.title}
                    </h3>
                    <p className="text-pretty text-lg leading-relaxed text-muted-foreground">
                      {chapter.body}
                    </p>
                  </div>
                </li>
              );
            })}
          </ol>
        </div>
      </Container>
    </section>
  );
}
