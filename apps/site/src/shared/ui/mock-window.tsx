import type { ReactNode } from "react";
import { cn } from "@/shared/lib";

interface MockWindowProps {
  children: ReactNode;
  className?: string;
  /** Shown centered in the title bar, like a document or channel name. */
  title?: string;
}

/**
 * A solid app window for product illustrations: title bar with window
 * controls over a card surface (ADR 007: no glass, glow or gradients).
 */
export function MockWindow({ children, className, title }: MockWindowProps) {
  return (
    <div
      className={cn(
        "w-full overflow-hidden rounded-2xl border border-border bg-card text-left text-card-foreground shadow-4",
        className,
      )}
    >
      <div className="relative flex h-10 items-center gap-1.5 border-b border-border px-4">
        <span className="size-2.5 rounded-full bg-muted" />
        <span className="size-2.5 rounded-full bg-muted" />
        <span className="size-2.5 rounded-full bg-muted" />
        {title ? (
          <span className="absolute inset-x-16 truncate text-center text-xs text-muted-foreground">
            {title}
          </span>
        ) : null}
      </div>
      {children}
    </div>
  );
}
