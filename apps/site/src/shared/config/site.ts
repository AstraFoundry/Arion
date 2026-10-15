import { publicEnv } from "./env";

/**
 * Every outbound link on the site is built from these values. The two
 * origins come from the build environment; paths on them are fixed.
 */
const REPOSITORY = "https://github.com/AstraFoundry/Arion";

export const siteConfig = {
  name: "Arion",
  organization: "AstraFoundry",
  url: publicEnv.siteUrl,
  accountUrl: `${publicEnv.platformUrl}/account`,
  hostedSyncWaitlistUrl: `${publicEnv.platformUrl}/products/arion/waitlist/hosted-sync`,
  repositoryUrl: REPOSITORY,
  repositoryCloneUrl: `${REPOSITORY}.git`,
  releasesUrl: `${REPOSITORY}/releases`,
  licenseUrl: `${REPOSITORY}/blob/main/LICENSE`,
  issuesUrl: `${REPOSITORY}/issues`,
  syncServiceUrl: `${REPOSITORY}/tree/main/services/sync`,
  organizationUrl: "https://github.com/AstraFoundry",
  license: "AGPL-3.0",
  shortcut: ["⌥", "Space"],
  minimumMacOS: "14",
} as const;

/** Internal routes. Trailing slashes match the static export layout. */
export const routes = {
  home: "/",
  privacy: "/privacy/",
  selfHosting: "/self-hosting/",
  credits: "/credits/",
} as const;

/** In-page anchors on the home page. */
export const sections = {
  values: "values",
  privacy: "privacy",
  openSource: "open-source",
  download: "download",
} as const;

/** Commands that build the desktop app from a fresh clone (see the Makefile). */
export const buildCommands = [
  `git clone ${siteConfig.repositoryCloneUrl}`,
  "cd Arion",
  "make install",
  "make desktop-build",
] as const;

export const anchor = (id: string) => `/#${id}`;
