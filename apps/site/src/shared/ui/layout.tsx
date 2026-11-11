import type { ReactNode } from "react";
import { cn } from "@/shared/lib";

interface SectionHeadingProps {
  title: ReactNode;
  lead?: ReactNode;
  /** `split` sets the title on the left and the lead on the right from `lg` up. */
  align?: "start" | "center" | "split";
  className?: string;
  titleClassName?: string;
  as?: "h1" | "h2";
}

/** Display-serif title with an optional lead paragraph. No kicker by design. */
export function SectionHeading({
  title,
  lead,
  align = "start",
  className,
  titleClassName,
  as: Heading = "h2",
}: SectionHeadingProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-4",
        align === "center" && "items-center text-center",
        align === "split" &&
          "lg:grid lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)] lg:items-end lg:gap-16",
        className,
      )}
    >
      <Heading
        className={cn(
          "text-balance font-display text-4xl leading-[1.05] font-normal tracking-[-0.02em] sm:text-5xl",
          titleClassName,
        )}
      >
        {title}
      </Heading>
      {lead ? (
        <p
          className={cn(
            "max-w-xl text-pretty text-lg leading-relaxed text-muted-foreground",
            align === "split" && "lg:justify-self-end lg:pb-1.5",
          )}
        >
          {lead}
        </p>
      ) : null}
    </div>
  );
}
