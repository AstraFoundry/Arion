import { TextReveal } from "@/shared/beui/components/motion/text-reveal";
import { downloadHref, siteConfig } from "@/shared/config";
import { format, useCopy } from "@/shared/i18n";
import { Container, LinkButton, Reveal } from "@/shared/ui";
import { HeroStage } from "./hero-stage";

/**
 * Hero: one left-aligned column of the headline, what Arion does in one
 * sentence, and the download and source links; below, the scroll-driven stage
 * where the demo rises out of the flowers of a photograph that widens to
 * fill the viewport. Every real control and line of copy stays outside the
 * art.
 */
export function Hero() {
  const copy = useCopy();
  return (
    <section className="pt-10">
      <Container size="wide" className="flex flex-col items-start">
        <TextReveal
          as="h1"
          text={copy.hero.title}
          stagger={0.06}
          blur={8}
          yOffset="24%"
          className="max-w-4xl text-balance font-display text-[length:var(--hero-title-size)] leading-[0.98] font-normal tracking-[-0.03em]"
        />
        <Reveal delay={0.35} className="mt-4.5 flex flex-col items-start">
          <p className="max-w-xl text-pretty text-lg leading-relaxed text-muted-foreground">
            {copy.hero.lead}
          </p>
          <div className="mt-7.5 flex flex-wrap items-center gap-3">
            <LinkButton href={downloadHref} size="lg">
              {copy.common.downloadLong}
            </LinkButton>
            <LinkButton href={siteConfig.repositoryUrl} size="lg" variant="secondary">
              {copy.common.viewSource}
            </LinkButton>
          </div>
          <p className="mt-4 text-sm text-muted-foreground">
            {format(copy.hero.note, {
              license: siteConfig.license,
              macos: siteConfig.minimumMacOS,
            })}
          </p>
        </Reveal>
      </Container>

      <div className="mt-10">
        <HeroStage />
      </div>
    </section>
  );
}
