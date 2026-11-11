import { Fraunces, IBM_Plex_Sans, JetBrains_Mono } from "next/font/google";

// Sunbeam type stacks: Fraunces (display), IBM Plex Sans (body),
// JetBrains Mono (code), exposed as the --font-latin-* variables that
// @astrafoundry/design's font stacks read; its stacks add the Chinese and
// Japanese fallbacks. Self-hosted at build time by next/font.
const fraunces = Fraunces({
  subsets: ["latin"],
  style: ["normal", "italic"],
  axes: ["opsz", "SOFT"],
  variable: "--font-latin-display",
  display: "swap",
});

const plex = IBM_Plex_Sans({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  variable: "--font-latin-sans",
  display: "swap",
});

const jetbrains = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-latin-mono",
  display: "swap",
});

export const fontVariables = [fraunces.variable, plex.variable, jetbrains.variable].join(
  " ",
);
