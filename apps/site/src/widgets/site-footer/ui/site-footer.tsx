import { format, useCopy } from "@/shared/i18n";
import { AppLink, BrandMark, BrandWord, Container } from "@/shared/ui";
import { footerColumns } from "../config/columns";

export function SiteFooter() {
  const copy = useCopy();
  const year = new Date().getFullYear();

  return (
    <footer className="border-t border-border">
      <Container size="wide" className="grid gap-12 pt-16 pb-10 md:grid-cols-[1.2fr_2fr]">
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <BrandMark />
            <BrandWord className="text-xl" />
          </div>
          <p className="max-w-xs text-sm text-muted-foreground">{copy.footer.tagline}</p>
        </div>

        <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
          {footerColumns(copy).map((column) => (
            <div key={column.title}>
              <h2 className="text-sm font-medium text-foreground">{column.title}</h2>
              <ul className="mt-3 flex flex-col gap-2">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <AppLink
                      href={link.href}
                      className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {link.label}
                    </AppLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </Container>

      <Container size="wide">
        <p className="border-t border-border py-6 text-sm text-muted-foreground">
          {format(copy.footer.copyright, { year })}
        </p>
      </Container>
    </footer>
  );
}
