"use client";

import { useReducedMotion } from "motion/react";
import { useEffect, useRef } from "react";

interface VideoClipProps {
  src: string;
  poster?: string;
  width: number;
  height: number;
  caption?: string;
  title?: string;
  /** Muted, looping clip that plays only while on screen. */
  autoPlay?: boolean;
  loop?: boolean;
}

/**
 * `<video-clip>`: nothing but the poster loads until the clip is played or,
 * for an autoplaying clip, scrolled half into view. Autoplaying clips pause
 * off screen and never play under reduced motion; they get controls instead.
 */
export function VideoClip({ src, poster, width, height, caption, title, autoPlay, loop }: VideoClipProps) {
  const video = useRef<HTMLVideoElement>(null);
  const reduceMotion = useReducedMotion();
  const ambient = Boolean(autoPlay) && reduceMotion === false;

  useEffect(() => {
    const element = video.current;
    if (!ambient || !element) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting) {
          element.pause();
          return;
        }
        // Pausing before playback starts rejects play() with AbortError.
        element.play().catch((error: DOMException) => {
          if (error.name !== "AbortError") throw error;
        });
      },
      { threshold: 0.5 },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [ambient]);

  return (
    <figure className="post-wide post-media">
      <video
        ref={video}
        src={src}
        poster={poster}
        width={width}
        height={height}
        title={title}
        preload="none"
        playsInline
        muted={Boolean(autoPlay)}
        loop={loop || Boolean(autoPlay)}
        controls={!ambient}
      />
      {caption ? <figcaption>{caption}</figcaption> : null}
    </figure>
  );
}
