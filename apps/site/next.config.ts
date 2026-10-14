import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
// Importing the parsed env fails the build early when the site configuration
// is missing or invalid (see .env.example).
import "./src/shared/config/env";

const withNextIntl = createNextIntlPlugin("./src/shared/i18n/request.ts");

// The site is fully static: every route is pre-rendered at build time into
// `out/`, which Cloudflare Workers serves as static assets (wrangler.jsonc).
const nextConfig: NextConfig = {
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  reactStrictMode: true,
  // English and the prefixed locales have separate root layouts (each sets
  // <html lang>), so the site-wide 404 is `app/global-not-found.tsx`.
  experimental: { globalNotFound: true },
};

export default withNextIntl(nextConfig);
