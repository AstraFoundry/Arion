import { TextReveal } from "@/shared/beui/components/motion/text-reveal";
import { downloadHref, siteConfig } from "@/shared/config";
import { format, useCopy } from "@/shared/i18n";
import { Backdrop, Container, LinkButton, Reveal } from "@/shared/ui";
import { HeroDemo } from "./hero-demo";

/**
 * Hero: headline, what Arion does in one sentence, the download and source
 * links, and the dictation demo floating over a motion-blurred photograph.
 */
export function Hero() {
  const copy = useCopy();
  const shortcut = siteConfig.shortcut.join(" ");
  return (
    <section className="relative pt-10 sm:pt-16">
      <Container size="narrow" className="flex flex-col items-center text-center">
        <TextReveal
          as="h1"
          text={copy.hero.title}
          stagger={0.06}
          blur={8}
          yOffset="24%"
          className="text-balance font-display text-[length:var(--hero-title-size)] leading-[0.98] font-normal tracking-[-0.03em]"
        />
        <Reveal delay={0.35} className="flex flex-col items-center">
          <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground sm:text-xl">
            {format(copy.hero.lead, { shortcut })}
          </p>
          <div className="mt-9 flex flex-col items-center gap-3 sm:flex-row">
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

      <Container size="wide" className="mt-12 sm:mt-16">
        <Reveal delay={0.5}>
          <div className="relative overflow-hidden rounded-3xl bg-muted px-4 pt-12 pb-24 sm:px-10 sm:pt-20 lg:px-20">
            <Backdrop priority />
            <div className="relative mx-auto max-w-3xl">
              <HeroDemo />
            </div>
          </div>
        </Reveal>
      </Container>
    </section>
  );
}
