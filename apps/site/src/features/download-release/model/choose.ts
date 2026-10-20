import type { DownloadTarget, ReleaseChannel } from "@/shared/config";

/** An installer as the center's channel summary describes it. */
export interface Installer {
  name: string;
  size: number;
  sha256: string;
  url: string;
}

/** The release a channel offers for one target. */
export interface TargetRelease {
  version: string;
  prerelease: boolean;
  publishedAt: string;
  notes: string;
  installer: Installer;
}

/** `GET /v1/releases/{channel}`; a channel without releases has no targets. */
export interface ChannelSummary {
  channel: ReleaseChannel;
  targets: Partial<Record<string, TargetRelease>>;
}

export type DownloadChoice =
  | { kind: "stable" | "preview"; release: TargetRelease }
  | { kind: "unavailable" };

/**
 * The build offered for a target: the stable release when there is one,
 * otherwise the newest preview, which the beta channel holds.
 */
export function chooseDownload(
  target: DownloadTarget,
  stable: ChannelSummary,
  beta: ChannelSummary,
): DownloadChoice {
  const release = stable.targets[target];
  if (release) return { kind: "stable", release };
  const preview = beta.targets[target];
  if (preview) return { kind: "preview", release: preview };
  return { kind: "unavailable" };
}
