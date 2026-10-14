"use client";

import { Check, Languages } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useId, useRef, useState } from "react";
import { useCopy } from "@/shared/i18n";
import { cn } from "@/shared/lib";
import { type LocaleLink, useLocaleLinks } from "../lib/use-locale-links";

const EASE_OUT = [0.22, 1, 0.36, 1] as const;

/**
 * Switching language is a full page load: each locale has its own document
 * (and `<html lang>`), so the options are plain anchors.
 */
function LocaleOption({ link, className }: { link: LocaleLink; className: string }) {
  return (
    <a
      href={link.href}
      hrefLang={link.tag}
      lang={link.tag}
      aria-current={link.current ? "true" : undefined}
      className={className}
    >
      {link.name}
      {link.current ? <Check className="ml-auto size-4" aria-hidden="true" /> : null}
    </a>
  );
}

/** Header control: a button that discloses the list of languages. */
export function LocaleSwitch({ className }: { className?: string }) {
  const copy = useCopy();
  const links = useLocaleLinks();
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const listId = useId();

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={root} className={cn("relative", className)}>
      <button
        type="button"
        aria-label={copy.common.language}
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => setOpen((value) => !value)}
        className="grid size-9 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <Languages className="size-4" aria-hidden="true" />
      </button>
      <AnimatePresence>
        {open ? (
          <motion.ul
            id={listId}
            initial={{ opacity: 0, y: -4, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.97 }}
            transition={{ duration: 0.16, ease: EASE_OUT }}
            className="absolute top-full right-0 mt-2 flex w-44 origin-top-right flex-col rounded-xl border border-border bg-popover p-1.5 text-popover-foreground shadow-3"
          >
            {links.map((link) => (
              <li key={link.locale}>
                <LocaleOption
                  link={link}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors hover:bg-muted aria-[current]:font-medium"
                />
              </li>
            ))}
          </motion.ul>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

/** Inline list for the mobile menu. */
export function LocaleList({ className }: { className?: string }) {
  const copy = useCopy();
  const links = useLocaleLinks();
  return (
    <nav aria-label={copy.common.language} className={className}>
      <ul className="flex flex-wrap gap-1.5">
        {links.map((link) => (
          <li key={link.locale}>
            <LocaleOption
              link={link}
              className="flex items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground aria-[current]:border-foreground/30 aria-[current]:text-foreground"
            />
          </li>
        ))}
      </ul>
    </nav>
  );
}
