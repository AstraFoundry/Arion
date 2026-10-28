"use client";

import { Play } from "lucide-react";
import { useState } from "react";
import { format, useCopy } from "@/shared/i18n";

interface EmbedFrameProps {
  src: string;
  title: string;
  /** `16/9`; or a fixed `height` in pixels. */
  ratio?: string;
  height?: number;
  caption?: string;
}

const isLocal = (src: string) => src.startsWith("/") && !src.startsWith("//");

/**
 * `<embed-frame>`: interactive HTML in an isolated, lazily loaded frame.
 * A page from this site (`/blog/<slug>/demo.html`) runs sandboxed with an
 * opaque origin. Another site is never contacted until the reader asks: the
 * frame starts as a button naming the host.
 */
export function EmbedFrame({ src, title, ratio, height, caption }: EmbedFrameProps) {
  const { blog } = useCopy();
  const local = isLocal(src);
  const [loaded, setLoaded] = useState(local);
  const size = ratio ? { aspectRatio: ratio } : { height };

  return (
    <figure className="post-wide post-media">
      <div className="post-embed" style={size}>
        {loaded ? (
          <iframe
            src={src}
            title={title}
            loading="lazy"
            sandbox={local ? "allow-scripts" : "allow-scripts allow-same-origin allow-popups allow-presentation"}
            referrerPolicy="strict-origin-when-cross-origin"
            allow="fullscreen; picture-in-picture"
          />
        ) : (
          <button type="button" onClick={() => setLoaded(true)} className="post-embed-load">
            <span className="post-embed-title">{title}</span>
            <span className="post-embed-action">
              <Play aria-hidden="true" />
              {format(blog.embedLoad, { host: new URL(src).host })}
            </span>
          </button>
        )}
      </div>
      {caption ? <figcaption>{caption}</figcaption> : null}
    </figure>
  );
}
