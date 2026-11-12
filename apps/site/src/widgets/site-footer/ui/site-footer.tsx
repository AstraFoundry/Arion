import { format, useCopy } from "@/shared/i18n";
import { AppLink, BrandMark, BrandWord, Container } from "@/shared/ui";
import { footerColumns } from "../config/columns";

/**
 * Full-viewport closing layer on the always-dark substrate. The footer box
 * clips its content (`clip-path`), and on viewports tall enough to hold it the
 * content is pinned to the viewport bottom, so the page sheet above slides
 * away to uncover a stationary layer instead of pushing the footer up.
 */
export function SiteFooter() {
  const copy = useCopy();
  const year = new Date().getFullYear();

  return (
    <footer className="site-footer dark relative min-h-dvh bg-substrate text-foreground [clip-path:inset(0)]">
      <div className="flex min-h-dvh flex-col footer-pinned:fixed footer-pinned:inset-x-0 footer-pinned:bottom-0 footer-pinned:h-dvh">
        <div className="site-footer-rise flex flex-1 flex-col">
          <Container size="wide" className="grid gap-12 pt-20 sm:pt-28 md:grid-cols-[1.2fr_2fr]">
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

          <div aria-hidden="true" className="pointer-events-none mt-auto select-none">
            <BrandWord className="block text-center text-[min(26vw,34dvh)] leading-[1.1] tracking-[-0.03em] text-border" />
          </div>

          <Container size="wide">
            <p className="border-t border-border py-6 text-sm text-muted-foreground">
              {format(copy.footer.copyright, { year })}
            </p>
          </Container>
        </div>
      </div>
    </footer>
  );
}
