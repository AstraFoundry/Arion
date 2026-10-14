import type { ReactNode } from "react";
import { Container, SectionHeading } from "./layout";

export interface DocumentSection {
  title: string;
  body: readonly string[];
}

interface DocumentPageProps {
  title: string;
  lead: string;
  /** Rendered between the lead and the sections, such as a status notice. */
  notice?: ReactNode;
  sections: readonly DocumentSection[];
  children?: ReactNode;
}

/** A long-form page: a title, a lead, then titled sections of paragraphs. */
export function DocumentPage({ title, lead, notice, sections, children }: DocumentPageProps) {
  return (
    <Container size="narrow" className="py-16 sm:py-24">
      <SectionHeading as="h1" title={title} lead={lead} />
      {notice ? <div className="mt-10">{notice}</div> : null}
      <div className="mt-14 flex flex-col gap-12">
        {sections.map((section) => (
          <section key={section.title} className="flex flex-col gap-4">
            <h2 className="font-display text-2xl leading-tight">{section.title}</h2>
            {section.body.map((paragraph) => (
              <p key={paragraph} className="text-pretty leading-relaxed text-muted-foreground">
                {paragraph}
              </p>
            ))}
          </section>
        ))}
      </div>
      {children}
    </Container>
  );
}
