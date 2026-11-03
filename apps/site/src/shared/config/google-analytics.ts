/** GA4 measurement ids look like `G-B7VQ241TTC`. Anything else is ignored. */
const MEASUREMENT_ID = /^G-[A-Z0-9]+$/;

/**
 * Measurement id for the marketing site's Google Analytics tag.
 *
 * `NEXT_PUBLIC_GA_MEASUREMENT_ID` is inlined at build time. Production sets it
 * to the AstraFoundry organization variable `GA_MEASUREMENT_ID`.
 */
export function googleAnalyticsMeasurementId(
  raw: string | undefined = process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID,
): string | null {
  const value = raw?.trim() ?? "";
  return MEASUREMENT_ID.test(value) ? value : null;
}
