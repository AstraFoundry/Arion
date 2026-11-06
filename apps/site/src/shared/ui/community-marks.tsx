interface MarkProps {
  className?: string;
}

/**
 * The Discord mark in its brand color (SVG Logos by Gil Barbara, CC0; the
 * desktop copy is `apps/desktop/ui/shared/icons/svg/logos-discord.svg`).
 */
export function DiscordMark({ className }: MarkProps) {
  return (
    <svg viewBox="0 0 256 199" aria-hidden="true" className={className}>
      <path
        fill="#5865f2"
        d="M216.856 16.597A208.5 208.5 0 0 0 164.042 0c-2.275 4.113-4.933 9.645-6.766 14.046q-29.538-4.442-58.533 0c-1.832-4.4-4.55-9.933-6.846-14.046a207.8 207.8 0 0 0-52.855 16.638C5.618 67.147-3.443 116.4 1.087 164.956c22.169 16.555 43.653 26.612 64.775 33.193A161 161 0 0 0 79.735 175.3a136.4 136.4 0 0 1-21.846-10.632a109 109 0 0 0 5.356-4.237c42.122 19.702 87.89 19.702 129.51 0a132 132 0 0 0 5.355 4.237a136 136 0 0 1-21.886 10.653c4.006 8.02 8.638 15.67 13.873 22.848c21.142-6.58 42.646-16.637 64.815-33.213c5.316-56.288-9.08-105.09-38.056-148.36M85.474 135.095c-12.645 0-23.015-11.805-23.015-26.18s10.149-26.2 23.015-26.2s23.236 11.804 23.015 26.2c.02 14.375-10.148 26.18-23.015 26.18m85.051 0c-12.645 0-23.014-11.805-23.014-26.18s10.148-26.2 23.014-26.2c12.867 0 23.236 11.804 23.015 26.2c0 14.375-10.148 26.18-23.015 26.18"
      />
    </svg>
  );
}

/** The X logo, in the current text color as X's brand guidelines allow. */
export function XMark({ className }: MarkProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" fill="currentColor" className={className}>
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
    </svg>
  );
}

/** Feishu's colored logo (`public/brand/feishu.png`, copied from the desktop app). */
export function FeishuMark({ className }: MarkProps) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- a fixed 96 px brand icon in a static export
    <img src="/brand/feishu.png" alt="" aria-hidden="true" width={96} height={96} className={className} />
  );
}
