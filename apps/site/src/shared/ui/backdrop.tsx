import { cn } from "@/shared/lib";

interface BackdropProps {
  className?: string;
  /** The hero backdrop is the largest image above the fold. */
  priority?: boolean;
}

/**
 * The soft, motion-blurred photograph product illustrations float on
 * (public/backdrops, credited on /credits). Decorative: it carries no alt.
 */
export function Backdrop({ className, priority = false }: BackdropProps) {
  return (
    <picture>
      <source media="(max-width: 639px)" srcSet="/backdrops/flowers-tall-800.webp" />
      <source media="(max-width: 1279px)" srcSet="/backdrops/flowers-wide-1200.webp" />
      <img
        src="/backdrops/flowers-wide-1600.webp"
        alt=""
        aria-hidden="true"
        fetchPriority={priority ? "high" : "auto"}
        loading={priority ? "eager" : "lazy"}
        className={cn("absolute inset-0 size-full object-cover", className)}
      />
    </picture>
  );
}
