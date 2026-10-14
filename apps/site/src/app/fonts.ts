import { Fraunces, IBM_Plex_Sans, JetBrains_Mono } from "next/font/google";

// Sunbeam type stacks: Fraunces (display), IBM Plex Sans (body),
// JetBrains Mono (code). Self-hosted at build time by next/font.
const fraunces = Fraunces({
  subsets: ["latin"],
  style: ["normal", "italic"],
  axes: ["opsz", "SOFT"],
  variable: "--font-fraunces",
  display: "swap",
});

const plex = IBM_Plex_Sans({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  variable: "--font-plex",
  display: "swap",
});

const jetbrains = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-jetbrains",
  display: "swap",
});

export const fontVariables = [fraunces.variable, plex.variable, jetbrains.variable].join(
  " ",
);
