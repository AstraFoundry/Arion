import { routes } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { Container, LinkButton, SectionHeading } from "@/shared/ui";

export function NotFoundPage() {
  const copy = useCopy();
  return (
    <Container
      size="narrow"
      className="flex flex-col items-center gap-8 py-32 text-center"
    >
      <SectionHeading
        as="h1"
        title={copy.notFound.title}
        lead={copy.notFound.body}
        align="center"
      />
      <LinkButton href={routes.home} variant="secondary">
        {copy.notFound.home}
      </LinkButton>
    </Container>
  );
}
