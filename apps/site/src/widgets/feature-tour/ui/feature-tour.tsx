"use client";

import { sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { Container, SectionHeading } from "@/shared/ui";
import { CHAPTERS, DESKTOP_ROWS } from "../config/chapters";
import { ExpandingGrid } from "./expanding-grid";

/**
 * The product's highlights. The heading stays put. Under it, each row opens
 * as the resting backdrop-card grid and, as it scrolls up and sticks, grows
 * until the cards fill the viewport. Below `lg` each chapter is its own row.
 */
export function FeatureTour() {
  const { tour } = useCopy();
  return (
    <section id={sections.values} className="scroll-mt-20 py-24 sm:py-32">
      <Container size="wide">
        <SectionHeading title={tour.title} lead={tour.lead} align="split" />
      </Container>
      <div className="mt-14 sm:mt-16">
        <div className="lg:hidden">
          {CHAPTERS.map((id) => (
            <ExpandingGrid key={id} ids={[id]} sharedPhoto={false} />
          ))}
        </div>
        <div className="hidden lg:block">
          {DESKTOP_ROWS.map((ids) => (
            <ExpandingGrid key={ids.join("-")} ids={ids} sharedPhoto />
          ))}
        </div>
      </div>
    </section>
  );
}
