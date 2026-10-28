"use client";

import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/shared/lib";

type LazyImageProps = ComponentPropsWithoutRef<"img"> & { src: string; alt: string };

const markLoaded = (image: HTMLImageElement | null) => {
  if (image?.complete && image.naturalWidth > 0) image.dataset.loaded = "";
};

/**
 * A post image: loaded when it nears the viewport, sized up front so the page
 * does not jump, and faded in once decoded. An image that finished before
 * hydration is marked from its ref, since its load event has already fired.
 */
export function LazyImage({ alt, className, ...rest }: LazyImageProps) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- static export; sized at build time
    <img
      {...rest}
      alt={alt}
      ref={markLoaded}
      loading="lazy"
      decoding="async"
      onLoad={(event) => markLoaded(event.currentTarget)}
      className={cn("post-image", className)}
    />
  );
}
