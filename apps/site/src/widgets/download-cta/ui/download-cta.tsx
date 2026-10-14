import { sections, siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { Container, LinkButton, Reveal, SectionHeading } from "@/shared/ui";

export function DownloadCta() {
  const copy = useCopy();
  const { download } = copy;
  return (
    <section id={sections.download} className="scroll-mt-20 py-24 sm:py-32">
      <Container size="narrow">
        <Reveal className="flex flex-col items-center gap-8 text-center">
          <SectionHeading title={download.title} lead={download.lead} align="center" />
          <LinkButton href={siteConfig.releasesUrl} size="lg">
            {copy.common.downloadLong}
          </LinkButton>
          <p className="text-sm text-muted-foreground">{download.note}</p>
        </Reveal>
      </Container>
    </section>
  );
}
