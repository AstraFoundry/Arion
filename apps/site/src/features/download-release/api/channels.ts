import { centerReleaseUrl, type ReleaseChannel } from "@/shared/config";
import type { ChannelSummary } from "../model/choose";

/** Reads a channel summary from the Arion center; any failure rejects. */
export async function fetchChannel(
  channel: ReleaseChannel,
  signal: AbortSignal,
): Promise<ChannelSummary> {
  const response = await fetch(centerReleaseUrl(channel), { signal });
  if (!response.ok) {
    throw new Error(`Center ${channel} channel returned HTTP ${response.status}`);
  }
  const body = (await response.json()) as ChannelSummary;
  if (body.channel !== channel || typeof body.targets !== "object" || body.targets === null) {
    throw new Error(`Center ${channel} channel summary is malformed`);
  }
  return body;
}
