import { format, useCopy } from "@/shared/i18n";
import { AppLink, BrandMark, BrandWord, Container } from "@/shared/ui";
import { footerColumns } from "../config/columns";

/**
 * Closing layer under the page sheet, on the `muted` tone one step off the
 * page. It sticks to the viewport bottom beneath the sheet, so the sheet
 * slides off to uncover it instead of pushing it up.
 */
export function SiteFooter() {
  const copy = useCopy();
  const year = new Date().getFullYear();

  return (
    <footer className="sticky bottom-0 z-0 bg-muted">
      <Container size="wide" className="grid gap-12 pt-16 pb-12 sm:pt-20 md:grid-cols-[1.2fr_2fr]">
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <BrandMark />
            <BrandWord className="text-xl" />
          </div>
          <p className="max-w-xs text-sm text-muted-foreground">{copy.footer.tagline}</p>
        </div>

        <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
          {footerColumns(copy).map((column) => (
            <div key={column.title}>
              <h2 className="text-xs font-medium tracking-[0.08em] text-faint uppercase">
                {column.title}
              </h2>
              <ul className="mt-4 flex flex-col gap-2.5">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <AppLink
                      href={link.href}
                      className="text-sm text-foreground/80 transition-colors hover:text-foreground"
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
