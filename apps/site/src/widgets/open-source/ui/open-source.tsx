import { buildCommands, routes, sections, siteConfig } from "@/shared/config";
import { format, useCopy } from "@/shared/i18n";
import { Container, GitHubMark, LinkButton, Reveal } from "@/shared/ui";

export function OpenSource() {
  const copy = useCopy();
  const { openSource } = copy;
  return (
    <section
      id={sections.openSource}
      className="scroll-mt-20 bg-substrate py-24 text-substrate-hi sm:py-32"
    >
      <Container className="grid items-start gap-14 lg:grid-cols-2">
        <div className="flex flex-col gap-6">
          <h2 className="text-balance font-display text-4xl leading-[1.05] font-normal tracking-[-0.02em] sm:text-5xl">
            {openSource.title}
          </h2>
          <p className="max-w-xl text-pretty text-lg leading-relaxed text-substrate-muted">
            {format(openSource.lead, { license: siteConfig.license })}
          </p>
          <div className="mt-2 flex flex-wrap gap-3">
            <LinkButton href={siteConfig.repositoryUrl} variant="inverse">
              <GitHubMark className="size-4" />
              {copy.common.viewSource}
            </LinkButton>
            <LinkButton href={routes.selfHosting} variant="inverse">
              {openSource.selfHosting}
            </LinkButton>
          </div>
          <a
            href={siteConfig.hostedSyncWaitlistUrl}
            className="w-fit rounded-sm text-sm text-substrate-ink underline decoration-substrate-line underline-offset-4 transition-colors hover:decoration-substrate-ink"
          >
            {openSource.waitlist}
          </a>
        </div>

        <Reveal className="overflow-hidden rounded-2xl border border-substrate-edge bg-substrate-surface">
          <p className="border-b border-substrate-line px-5 py-3 text-sm text-substrate-muted">
            {openSource.buildTitle}
          </p>
          <pre className="overflow-x-auto px-5 py-5 font-mono text-sm leading-7 text-substrate-hi">
            <code>
              {buildCommands.map((command) => (
                <span key={command} className="block">
                  <span className="text-substrate-muted select-none">$ </span>
                  {command}
                </span>
              ))}
            </code>
          </pre>
          <p className="border-t border-substrate-line px-5 py-3 text-sm text-substrate-muted">
            {openSource.buildNote}
          </p>
        </Reveal>
      </Container>
    </section>
  );
}
