"use client";

import { MotionConfig } from "motion/react";
import { ThemeProvider } from "next-themes";
import type { ReactNode } from "react";

/**
 * Global providers. Theme is class-based (`.dark` on <html>) so the Sunbeam
 * tokens from `@astrafoundry/design` are the single place where light and
 * dark differ.
 */
export function RootProviders({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <MotionConfig reducedMotion="user">{children}</MotionConfig>
    </ThemeProvider>
  );
}
