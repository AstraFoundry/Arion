"use client";

import { Play } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { format, useCopy } from "@/shared/i18n";

interface EmbedFrameProps {
  src: string;
  title: string;
  /** `16/9`; or a fixed `height` in pixels. */
  ratio?: string;
  height?: number;
  /** A still shown in the frame's place until the page has loaded. */
  poster?: string;
  caption?: string;
}

const isLocal = (src: string) => src.startsWith("/") && !src.startsWith("//");

/** How far ahead of the viewport a local frame starts loading. */
const LOAD_AHEAD = "100% 0px";

/**
 * `<embed-frame>`: interactive HTML in an isolated, lazily loaded frame.
 * The box is sized up front and shows the poster, if any. A page from this
 * site (`/blog/<slug>/demo.html`) is created when the frame nears the
 * viewport, runs sandboxed with an opaque origin, and fades in over the
 * poster once it has loaded. Creating the frame after hydration means its
 * load event is never missed. Another site is never contacted until the
 * reader asks: the frame starts as a button naming the host.
 */
export function EmbedFrame({ src, title, ratio, height, poster, caption }: EmbedFrameProps) {
  const { blog } = useCopy();
  const local = isLocal(src);
  const box = useRef<HTMLDivElement>(null);
  const [requested, setRequested] = useState(false);
  const [near, setNear] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const size = ratio ? { aspectRatio: ratio } : { height };

  useEffect(() => {
    const element = box.current;
    if (!local || !element) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting) return;
        setNear(true);
        observer.disconnect();
      },
      { rootMargin: LOAD_AHEAD },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [local]);

  const show = local ? near : requested;

  return (
    <figure className="post-wide post-media">
      <div ref={box} className="post-embed" style={size}>
        {poster ? (
          // eslint-disable-next-line @next/next/no-img-element -- static export; the frame sets the size
          <img src={poster} alt="" aria-hidden="true" loading="lazy" decoding="async" className="post-embed-poster" />
        ) : null}
        {show ? (
          <iframe
            src={src}
            title={title}
            data-loaded={loaded ? "" : undefined}
            onLoad={() => setLoaded(true)}
            sandbox={local ? "allow-scripts" : "allow-scripts allow-same-origin allow-popups allow-presentation"}
            referrerPolicy="strict-origin-when-cross-origin"
            allow="fullscreen; picture-in-picture"
          />
        ) : local ? null : (
          <button type="button" onClick={() => setRequested(true)} className="post-embed-load">
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
