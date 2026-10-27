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
  centerUrl: publicEnv.centerUrl,
  licenseUrl: `${REPOSITORY}/blob/main/LICENSE`,
  issuesUrl: `${REPOSITORY}/issues`,
  syncServiceUrl: `${REPOSITORY}/tree/main/services/sync`,
  organizationUrl: "https://github.com/AstraFoundry",
  license: "AGPL-3.0",
  shortcut: ["⌥", "Space"],
  /** Tapped alone, it switches the input method between Chinese and English. */
  languageSwitchKey: "Shift",
  minimumMacOS: "14",
} as const;

/** Internal routes. Trailing slashes match the static export layout. */
export const routes = {
  home: "/",
  privacy: "/privacy/",
  selfHosting: "/self-hosting/",
  credits: "/credits/",
  blog: "/blog/",
} as const;

/** A blog post's route; `slug` is its directory under `content/blog`. */
export const blogPostRoute = (slug: string) => `${routes.blog}${slug}/`;

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

/** Every "Download" link goes to the download section, which picks the build. */
export const downloadHref = anchor(sections.download);

/** Desktop builds the download section offers, in display order. */
export const downloadTargets = ["darwin-arm64", "windows-amd64"] as const;
export type DownloadTarget = (typeof downloadTargets)[number];

/** Update channels of the Arion center (services/updates). */
export type ReleaseChannel = "stable" | "beta";

/** The center's public summary of a channel. */
export const centerReleaseUrl = (channel: ReleaseChannel) =>
  `${siteConfig.centerUrl}/v1/releases/${channel}`;

/** Redirects to the channel's current installer for target. */
export const centerDownloadUrl = (channel: ReleaseChannel, target: DownloadTarget) =>
  `${siteConfig.centerUrl}/v1/downloads/${channel}/${target}`;
