"use client";

import { Menu, X } from "lucide-react";
import { AnimatePresence, motion, useMotionValueEvent, useScroll } from "motion/react";
import { useState } from "react";
import { LocaleList, LocaleSwitch } from "@/features/switch-locale";
import { ThemeSwitch } from "@/features/theme-switch";
import { routes, siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { cn } from "@/shared/lib";
import { AppLink, BrandMark, BrandWord, GitHubMark, LinkButton } from "@/shared/ui";
import { navItems } from "../config/nav";

const EASE_OUT = [0.22, 1, 0.36, 1] as const;

export function SiteHeader() {
  const copy = useCopy();
  const items = navItems(copy);
  const { scrollY } = useScroll();
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  useMotionValueEvent(scrollY, "change", (value) => setScrolled(value > 8));

  return (
    <header
      className={cn(
        "sticky top-0 z-50 border-b transition-[background-color,border-color] duration-200",
        scrolled || menuOpen
          ? "border-border bg-background/90 backdrop-blur-md"
          : "border-transparent bg-transparent",
      )}
    >
      <div className="mx-auto flex h-16 w-full max-w-7xl items-center gap-6 px-5 sm:px-8">
        <AppLink
          href={routes.home}
          className="flex items-center gap-2 rounded-md text-foreground"
          aria-label={siteConfig.name}
        >
          <BrandMark />
          <BrandWord className="text-xl" />
        </AppLink>

        <nav aria-label={copy.nav.label} className="hidden lg:block">
          <ul className="flex items-center gap-1">
            {items.map((item) => (
              <li key={item.href}>
                <AppLink
                  href={item.href}
                  className="rounded-full px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                >
                  {item.label}
                </AppLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className="ml-auto flex items-center gap-1">
          <a
            href={siteConfig.repositoryUrl}
            aria-label={copy.common.githubLabel}
            className="grid size-9 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <GitHubMark className="size-4" />
          </a>
          <LocaleSwitch className="hidden lg:block" />
          <ThemeSwitch />
          <LinkButton
            href={siteConfig.accountUrl}
            variant="ghost"
            size="sm"
            title={copy.common.account}
            className="hidden sm:inline-flex"
          >
            {copy.nav.account}
          </LinkButton>
          <LinkButton
            href={siteConfig.releasesUrl}
            size="sm"
            className="ml-1 hidden sm:inline-flex"
          >
            {copy.common.download}
          </LinkButton>
          <button
            type="button"
            onClick={() => setMenuOpen((open) => !open)}
            aria-expanded={menuOpen}
            aria-controls="mobile-nav"
            aria-label={menuOpen ? copy.common.closeMenu : copy.common.openMenu}
            className="grid size-9 place-items-center rounded-full text-foreground transition-colors hover:bg-muted lg:hidden"
          >
            {menuOpen ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>
        </div>
      </div>

      <AnimatePresence initial={false}>
        {menuOpen ? (
          <motion.nav
            id="mobile-nav"
            aria-label={copy.nav.label}
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: EASE_OUT }}
            className="overflow-hidden lg:hidden"
          >
            <ul className="flex flex-col gap-1 px-5 pt-2 pb-4">
              {items.map((item) => (
                <li key={item.href}>
                  <AppLink
                    href={item.href}
                    onClick={() => setMenuOpen(false)}
                    className="block rounded-lg px-3 py-2.5 text-base text-foreground hover:bg-muted"
                  >
                    {item.label}
                  </AppLink>
                </li>
              ))}
              <li className="mt-2 px-1">
                <LocaleList />
              </li>
              <li className="mt-2 flex gap-2 px-1">
                <LinkButton href={siteConfig.accountUrl} variant="secondary" className="flex-1">
                  {copy.common.account}
                </LinkButton>
                <LinkButton href={siteConfig.releasesUrl} className="flex-1">
                  {copy.common.download}
                </LinkButton>
              </li>
            </ul>
          </motion.nav>
        ) : null}
      </AnimatePresence>
    </header>
  );
}
