import { type Credit, photoCredits, softwareCredits } from "@/shared/config";
import { format, useCopy } from "@/shared/i18n";
import { Container, SectionHeading } from "@/shared/ui";

function CreditList({ title, credits }: { title: string; credits: readonly Credit[] }) {
  const { creditsPage } = useCopy();
  return (
    <section className="flex flex-col gap-4">
      <h2 className="font-display text-2xl leading-tight">{title}</h2>
      <ul className="flex flex-col divide-y divide-border border-y border-border">
        {credits.map((credit) => (
          <li key={credit.id} className="flex flex-wrap items-baseline gap-x-3 gap-y-1 py-4">
            <a
              href={credit.sourceUrl}
              className="rounded-sm font-medium underline decoration-border underline-offset-4 transition-colors hover:decoration-foreground"
            >
              {credit.title}
            </a>
            <span className="text-muted-foreground">
              {format(creditsPage.by, { author: credit.author })}
            </span>
            <a
              href={credit.licenseUrl}
              className="ml-auto rounded-sm text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {credit.license}
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
}

export function CreditsPage() {
  const { creditsPage } = useCopy();
  return (
    <Container size="narrow" className="flex flex-col gap-14 py-16 sm:py-24">
      <SectionHeading as="h1" title={creditsPage.title} lead={creditsPage.lead} />
      <CreditList title={creditsPage.photos} credits={photoCredits} />
      <CreditList title={creditsPage.software} credits={softwareCredits} />
    </Container>
  );
}
