"use client";

import { useEffect, useState } from "react";
import {
  centerDownloadUrl,
  type DownloadTarget,
  downloadTargets,
} from "@/shared/config";
import { format, useCopy, type Messages } from "@/shared/i18n";
import { LinkButton } from "@/shared/ui";
import { fetchChannel } from "../api/channels";
import { type ChannelSummary, chooseDownload } from "../model/choose";

type Channels =
  | { status: "loading" }
  | { status: "failed" }
  | { status: "ready"; stable: ChannelSummary; beta: ChannelSummary };

const targetLabel = (copy: Messages["download"], target: DownloadTarget) =>
  ({
    "darwin-arm64": copy.targets.darwinArm64,
    "windows-amd64": copy.targets.windowsAmd64,
  })[target];

/**
 * One row per desktop build. The static HTML links each button to the
 * center's stable download; once the channel summaries load, a target
 * without a stable release offers the newest preview instead, with its
 * version and checksum.
 */
export function DownloadOptions() {
  const messages = useCopy();
  const copy = messages.download;
  const [channels, setChannels] = useState<Channels>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      fetchChannel("stable", controller.signal),
      fetchChannel("beta", controller.signal),
    ]).then(
      ([stable, beta]) => setChannels({ status: "ready", stable, beta }),
      () => {
        if (!controller.signal.aborted) setChannels({ status: "failed" });
      },
    );
    return () => controller.abort();
  }, []);

  return (
    <div className="flex w-full max-w-xl flex-col gap-3">
      <ul className="flex flex-col divide-y divide-border rounded-2xl border border-border bg-card text-left">
        {downloadTargets.map((target) => {
          const choice =
            channels.status === "ready"
              ? chooseDownload(target, channels.stable, channels.beta)
              : null;
          const href =
            choice?.kind === "preview"
              ? centerDownloadUrl("beta", target)
              : centerDownloadUrl("stable", target);
          return (
            <li key={target} className="flex items-center gap-4 px-5 py-4">
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="font-medium">{targetLabel(copy, target)}</span>
                {choice && choice.kind !== "unavailable" ? (
                  <span className="text-sm text-muted-foreground">
                    {format(copy.version, { version: choice.release.version })}
                    {choice.kind === "preview" ? ` · ${copy.preview}` : null}
                  </span>
                ) : null}
                {choice && choice.kind !== "unavailable" ? (
                  <span
                    className="truncate font-mono text-xs text-muted-foreground"
                    title={choice.release.installer.sha256}
                  >
                    {format(copy.checksum, { sum: choice.release.installer.sha256 })}
                  </span>
                ) : null}
              </div>
              {choice?.kind === "unavailable" ? (
                <span className="text-sm text-muted-foreground">{copy.unavailable}</span>
              ) : (
                <LinkButton href={href} size="sm" variant={choice?.kind === "preview" ? "secondary" : "primary"}>
                  {messages.common.download}
                </LinkButton>
              )}
            </li>
          );
        })}
      </ul>
      {channels.status === "failed" ? (
        <p role="status" className="text-sm text-muted-foreground">
          {copy.failed}
        </p>
      ) : null}
    </div>
  );
}
