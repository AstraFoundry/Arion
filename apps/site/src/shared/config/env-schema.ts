/**
 * Build-time configuration for the static site. Values come from
 * `NEXT_PUBLIC_*` environment variables and are inlined into the bundle, so
 * changing them requires a new build.
 */
export interface PublicEnv {
  /** Origin of this site, without a trailing slash. */
  siteUrl: string;
  /** Origin of the Astra Foundry platform, without a trailing slash. */
  platformUrl: string;
}

export interface RawPublicEnv {
  siteUrl: string | undefined;
  platformUrl: string | undefined;
}

export class PublicEnvError extends Error {
  constructor(message: string) {
    super(`Invalid site configuration: ${message}`);
    this.name = "PublicEnvError";
  }
}

/** Accepts a bare http(s) origin: no path, query or fragment. */
function parseOrigin(name: string, raw: string | undefined): string {
  const value = raw?.trim();
  if (!value) throw new PublicEnvError(`${name} is required (see .env.example).`);
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    throw new PublicEnvError(`${name} must be an absolute URL (got ${JSON.stringify(raw)}).`);
  }
  if (url.protocol !== "https:" && url.protocol !== "http:") {
    throw new PublicEnvError(`${name} must use https or http (got ${JSON.stringify(raw)}).`);
  }
  if (url.pathname !== "/" || url.search || url.hash) {
    throw new PublicEnvError(`${name} must be an origin without a path (got ${JSON.stringify(raw)}).`);
  }
  return url.origin;
}

export function parsePublicEnv(raw: RawPublicEnv): PublicEnv {
  return {
    siteUrl: parseOrigin("NEXT_PUBLIC_SITE_URL", raw.siteUrl),
    platformUrl: parseOrigin("NEXT_PUBLIC_PLATFORM_URL", raw.platformUrl),
  };
}
