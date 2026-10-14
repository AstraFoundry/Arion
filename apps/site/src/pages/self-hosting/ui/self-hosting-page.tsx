import { siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { DocumentPage, LinkButton } from "@/shared/ui";

export function SelfHostingPage() {
  const { selfHostingPage: page } = useCopy();
  return (
    <DocumentPage
      title={page.title}
      lead={page.lead}
      notice={
        <p role="note" className="rounded-xl border border-border bg-card px-5 py-4 text-sm leading-relaxed">
          {page.status}
        </p>
      }
      sections={page.sections}
    >
      <p className="mt-10">
        <a
          href={siteConfig.syncServiceUrl}
          className="rounded-sm font-medium underline decoration-border underline-offset-4 transition-colors hover:decoration-foreground"
        >
          {page.source}
        </a>
      </p>
      <div className="mt-16 flex flex-col items-start gap-4 rounded-2xl bg-muted p-8">
        <h2 className="font-display text-2xl leading-tight">{page.hostedTitle}</h2>
        <p className="text-pretty leading-relaxed text-muted-foreground">{page.hostedBody}</p>
        <LinkButton href={siteConfig.hostedSyncWaitlistUrl} variant="secondary">
          {page.hostedLink}
        </LinkButton>
      </div>
    </DocumentPage>
  );
}
