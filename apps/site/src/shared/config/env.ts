import { parsePublicEnv } from "./env-schema";

// Each variable is referenced literally so Next.js can inline it into
// client bundles at build time.
export const publicEnv = parsePublicEnv({
  siteUrl: process.env.NEXT_PUBLIC_SITE_URL,
  platformUrl: process.env.NEXT_PUBLIC_PLATFORM_URL,
});
