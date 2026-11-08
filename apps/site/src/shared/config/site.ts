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
  /** AstraFoundry's own landing page. */
  organizationSiteUrl: publicEnv.platformUrl,
  repositoryUrl: REPOSITORY,
  repositoryCloneUrl: `${REPOSITORY}.git`,
  centerUrl: publicEnv.centerUrl,
  licenseUrl: `${REPOSITORY}/blob/main/LICENSE`,
  issuesUrl: `${REPOSITORY}/issues`,
  syncServiceUrl: `${REPOSITORY}/tree/main/services/sync`,
  organizationUrl: "https://github.com/AstraFoundry",
  /** Where people talk about Arion; the same links as the repository README. */
  community: {
    discord: "https://discord.com/invite/72kh465mMm",
    feishu:
      "https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae",
    x: "https://x.com/AstraFoundry",
  },
  license: "AGPL-3.0",
  shortcut: ["⌥", "Space"],
  /** Tapped alone, it switches the input method between Chinese and English. */
  languageSwitchKey: "Shift",
  minimumMacOS: "14",
} as const;

/** The site-wide social preview (`public/og.jpg`); its alt text is `meta.ogAlt`. */
export const socialImage = { url: "/og.jpg", width: 1200, height: 630 } as const;

/** Internal routes. Trailing slashes match the static export layout. */
export const routes = {
  home: "/",
  privacy: "/privacy/",
  terms: "/terms/",
  usageData: "/usage-data/",
  selfHosting: "/self-hosting/",
  credits: "/credits/",
  blog: "/blog/",
} as const;

/** A blog post's route; `slug` is its directory under `content/blog`. */
export const blogPostRoute = (slug: string) => `${routes.blog}${slug}/`;

/** In-page anchors on the home page. */
export const sections = {
  values: "values",
  openSource: "open-source",
  testimonials: "testimonials",
  download: "download",
  community: "community",
} as const;

/** Commands that build the desktop app from a fresh clone (see the Makefile). */
export const buildCommands = [
  `git clone ${siteConfig.repositoryCloneUrl}`,
  "cd Arion",
  "make install",
  "make desktop-build",
] as const;

export const anchor = (id: string) => `/#${id}`;

/** Community channels in display order; labels are `community.<id>` in the dictionaries. */
export const communityChannels = ["discord", "feishu", "x"] as const;
export type CommunityChannel = (typeof communityChannels)[number];

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
