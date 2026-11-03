"use client";

import { Check, Copy, Download } from "lucide-react";
import { useEffect, useState } from "react";
import {
  centerDownloadUrl,
  type DownloadTarget,
  downloadTargets,
} from "@/shared/config";
import { useCopy, type Messages } from "@/shared/i18n";
import { AppleMark, LinkButton, WindowsMark } from "@/shared/ui";
import { fetchChannel } from "../api/channels";
import { type ChannelSummary, chooseDownload, type DownloadChoice } from "../model/choose";

type Channels =
  | { status: "loading" }
  | { status: "failed" }
  | { status: "ready"; stable: ChannelSummary; beta: ChannelSummary };

// The Windows mark fills its whole box, so it is drawn smaller to match the
// Apple mark's optical weight.
const targetMark: Record<DownloadTarget, { Mark: typeof AppleMark; size: string }> = {
  "darwin-arm64": { Mark: AppleMark, size: "size-7" },
  "windows-amd64": { Mark: WindowsMark, size: "size-6" },
};

const targetLabel = (copy: Messages["download"], target: DownloadTarget) =>
  ({
    "darwin-arm64": copy.targets.darwinArm64,
    "windows-amd64": copy.targets.windowsAmd64,
  })[target];

/**
 * One card per desktop build. The static HTML links each button to the
 * center's stable download; once the channel summaries load, a target
 * without a stable release offers the newest preview instead. Each card
 * shows its version; the SHA-256 is copied, not printed.
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

  const choices = downloadTargets.map((target) => ({
    target,
    choice:
      channels.status === "ready"
        ? chooseDownload(target, channels.stable, channels.beta)
        : null,
  }));
  return (
    <div className="flex w-full flex-col items-center gap-5">
      <ul className="grid w-full max-w-2xl gap-3 sm:grid-cols-2">
        {choices.map(({ target, choice }) => (
          <TargetCard key={target} target={target} choice={choice} />
        ))}
      </ul>
      {channels.status === "failed" ? (
        <p role="status" className="text-sm text-muted-foreground">
          {copy.failed}
        </p>
      ) : null}
    </div>
  );
}

function TargetCard({
  target,
  choice,
}: {
  target: DownloadTarget;
  choice: DownloadChoice | null;
}) {
  const messages = useCopy();
  const copy = messages.download;
  const label = targetLabel(copy, target);
  const { Mark, size } = targetMark[target];
  const href =
    choice?.kind === "preview"
      ? centerDownloadUrl("beta", target)
      : centerDownloadUrl("stable", target);
  return (
    <li className="flex flex-col items-center gap-4 rounded-2xl border border-border bg-card px-6 py-6">
      <div className="flex flex-col items-center gap-0.5">
        <span className="mb-2 flex size-7 items-center justify-center">
          <Mark className={size} />
        </span>
        <span className="text-lg font-medium">{label.name}</span>
        <span className="text-sm text-muted-foreground">{label.detail}</span>
      </div>
      {choice?.kind === "unavailable" ? (
        <span className="flex h-10 items-center text-sm text-muted-foreground">
          {copy.unavailable}
        </span>
      ) : (
        <LinkButton href={href} className="w-full">
          <Download aria-hidden className="size-4" />
          {messages.common.download}
          <span className="sr-only"> {label.name}</span>
        </LinkButton>
      )}
      <div className="flex min-h-5 items-center gap-1.5 text-xs text-muted-foreground tabular-nums">
        {choice && choice.kind !== "unavailable" ? (
          <>
            <span>{choice.release.version}</span>
            {choice.kind === "preview" ? <span>· {copy.preview}</span> : null}
            <span aria-hidden>·</span>
            <ChecksumButton sum={choice.release.installer.sha256} />
          </>
        ) : null}
      </div>
    </li>
  );
}

const COPIED_MS = 1600;

/** Copies an installer's SHA-256; the full value is also the tooltip. */
function ChecksumButton({ sum }: { sum: string }) {
  const { download: copy } = useCopy();
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), COPIED_MS);
    return () => window.clearTimeout(timer);
  }, [copied]);

  return (
    <button
      type="button"
      title={sum}
      aria-label={copied ? copy.checksumCopied : copy.copyChecksum}
      onClick={() => void navigator.clipboard.writeText(sum).then(() => setCopied(true))}
      className="inline-flex items-center gap-1 rounded-sm transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      {copy.checksum}
      {copied ? <Check aria-hidden className="size-3" /> : <Copy aria-hidden className="size-3" />}
    </button>
  );
}
