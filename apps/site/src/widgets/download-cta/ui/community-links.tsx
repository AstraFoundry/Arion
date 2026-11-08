import type { ComponentType } from "react";
import { type CommunityChannel, communityChannels, sections, siteConfig } from "@/shared/config";
import { useCopy } from "@/shared/i18n";
import { DiscordMark, FeishuMark, LinkButton, XMark } from "@/shared/ui";

const MARKS: Record<CommunityChannel, ComponentType<{ className?: string }>> = {
  discord: DiscordMark,
  feishu: FeishuMark,
  x: XMark,
};

/** Where to talk with the team and other users, under the download options. */
export function CommunityLinks() {
  const { community } = useCopy();
  return (
    <div
      id={sections.community}
      className="flex w-full scroll-mt-24 flex-col items-center gap-5 border-t border-border pt-10"
    >
      <div className="flex flex-col items-center gap-2">
        <h3 className="font-display text-2xl font-normal tracking-[-0.01em]">{community.title}</h3>
        <p className="max-w-md text-pretty text-muted-foreground">{community.lead}</p>
      </div>
      <ul className="flex flex-wrap justify-center gap-3">
        {communityChannels.map((channel) => {
          const Mark = MARKS[channel];
          return (
            <li key={channel}>
              <LinkButton href={siteConfig.community[channel]} variant="secondary">
                <Mark className="size-4" />
                {community[channel]}
              </LinkButton>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
