import type { ReactNode } from "react";
import { cn } from "@/shared/lib";

interface TourStageProps {
  children: ReactNode;
  className?: string;
}

/**
 * The ground a chapter mock floats on: a rounded crop of the motion-blurred
 * photograph. Its backdrop layers are passed in as children before the mock.
 */
export function TourStage({ children, className }: TourStageProps) {
  return (
    <div className={cn("relative isolate overflow-hidden rounded-3xl bg-muted", className)}>
      {children}
    </div>
  );
}

/** One chapter's backdrop crop. Decorative: it carries no alt. */
export function StageBackdrop({ src, className }: { src: string; className?: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- static export, pre-sized webp
    <img
      src={src}
      alt=""
      aria-hidden="true"
      loading="lazy"
      className={cn("absolute inset-0 -z-10 size-full object-cover", className)}
    />
  );
}
