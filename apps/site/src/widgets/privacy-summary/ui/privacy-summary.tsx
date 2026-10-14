import { ArrowRight } from "lucide-react";
import { routes, sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { AppLink, Container, Reveal, SectionHeading } from "@/shared/ui";

export function PrivacySummary() {
  const { privacy } = useCopy();
  return (
    <section id={sections.privacy} className="scroll-mt-20 border-t border-border py-24 sm:py-32">
      <Container className="grid gap-14 lg:grid-cols-[1fr_1.4fr]">
        <div className="flex flex-col gap-6">
          <SectionHeading title={privacy.title} lead={privacy.lead} />
          <AppLink
            href={routes.privacy}
            className="group inline-flex w-fit items-center gap-1.5 rounded-sm font-medium text-foreground underline decoration-border underline-offset-4 transition-colors hover:decoration-foreground"
          >
            {privacy.more}
            <ArrowRight
              className="size-4 transition-transform group-hover:translate-x-0.5"
              aria-hidden="true"
            />
          </AppLink>
        </div>
        <ol className="flex flex-col divide-y divide-border border-y border-border">
          {privacy.points.map((point, index) => (
            <li key={point.title}>
              <Reveal delay={index * 0.06} className="grid gap-2 py-7 sm:grid-cols-[12rem_1fr] sm:gap-8">
                <h3 className="font-medium">{point.title}</h3>
                <p className="text-pretty leading-relaxed text-muted-foreground">{point.body}</p>
              </Reveal>
            </li>
          ))}
        </ol>
      </Container>
    </section>
  );
}
