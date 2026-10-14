import { History, Languages, PenLine, WifiOff, type LucideIcon } from "lucide-react";
import { sections } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { Container, Reveal, SectionHeading } from "@/shared/ui";

/** One icon per entry of `values.items`, in dictionary order. */
const ICONS: readonly LucideIcon[] = [WifiOff, Languages, History, PenLine];

export function Values() {
  const { values } = useCopy();
  return (
    <section id={sections.values} className="scroll-mt-20 py-24 sm:py-32">
      <Container>
        <SectionHeading title={values.title} />
        <ul className="mt-14 grid gap-x-12 gap-y-12 sm:grid-cols-2">
          {values.items.map((item, index) => {
            const Icon = ICONS[index];
            return (
              <li key={item.title}>
                <Reveal delay={index * 0.06} className="flex flex-col gap-3">
                  {Icon ? <Icon className="size-5 text-foreground" aria-hidden="true" /> : null}
                  <h3 className="text-lg font-medium">{item.title}</h3>
                  <p className="max-w-md text-pretty leading-relaxed text-muted-foreground">
                    {item.body}
                  </p>
                </Reveal>
              </li>
            );
          })}
        </ul>
      </Container>
    </section>
  );
}
