import { TextReveal } from "@/shared/beui/components/motion/text-reveal";
import { downloadHref, siteConfig } from "@/shared/config";
import { format, useCopy } from "@/shared/i18n";
import { Container, LinkButton, Reveal } from "@/shared/ui";
import { HeroStage } from "./hero-stage";

/**
 * Hero: the headline on the page, with what Arion does in one sentence and
 * the download and source links beside it; below, the scroll-driven stage
 * where the demo rises out of the flowers of a photograph that widens to
 * fill the viewport. Every real control and line of copy stays outside the
 * art.
 */
export function Hero() {
  const copy = useCopy();
  return (
    <section className="pt-10 sm:pt-16 lg:pt-20">
      <Container
        size="wide"
        className="flex flex-col gap-8 lg:grid lg:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)] lg:items-end lg:gap-16"
      >
        <TextReveal
          as="h1"
          text={copy.hero.title}
          stagger={0.06}
          blur={8}
          yOffset="24%"
          className="text-balance font-display text-[length:var(--hero-title-size)] leading-[0.98] font-normal tracking-[-0.03em]"
        />
        <Reveal delay={0.35} className="flex flex-col items-start lg:pb-2">
          <p className="max-w-xl text-pretty text-lg leading-relaxed text-muted-foreground sm:text-xl">
            {copy.hero.lead}
          </p>
          <div className="mt-8 flex flex-wrap items-center gap-3">
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

      <div className="mt-12 sm:mt-16">
        <HeroStage />
      </div>
    </section>
  );
}
