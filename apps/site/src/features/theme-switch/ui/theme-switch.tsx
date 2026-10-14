"use client";

import { ThemeToggle } from "@/shared/beui/components/motion/theme-toggle";
import { useCopy } from "@/shared/i18n";
import { cn } from "@/shared/lib";

interface ThemeSwitchProps {
  className?: string;
}

/** Toggles light / dark with beUI's circular view-transition reveal. */
export function ThemeSwitch({ className }: ThemeSwitchProps) {
  const copy = useCopy();
  return (
    <ThemeToggle
      variant="circle"
      start="top-right"
      toDarkLabel={copy.common.themeToDark}
      toLightLabel={copy.common.themeToLight}
      className={cn(
        "size-9 rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
        className,
      )}
      iconClassName="size-4"
    />
  );
}
