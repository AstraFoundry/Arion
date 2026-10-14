import type { ComponentPropsWithoutRef, ElementType, ReactNode } from "react";
import { cn } from "@/shared/lib";

type ContainerProps<T extends ElementType> = {
  as?: T;
  size?: "narrow" | "default" | "wide";
  className?: string;
  children: ReactNode;
} & Omit<ComponentPropsWithoutRef<T>, "as" | "className" | "children">;

const CONTAINER_SIZE = {
  narrow: "max-w-3xl",
  default: "max-w-6xl",
  wide: "max-w-7xl",
} as const;

export function Container<T extends ElementType = "div">({
  as,
  size = "default",
  className,
  children,
  ...rest
}: ContainerProps<T>) {
  const Component = as ?? "div";
  return (
    <Component
      className={cn("mx-auto w-full px-5 sm:px-8", CONTAINER_SIZE[size], className)}
      {...rest}
    >
      {children}
    </Component>
  );
}

interface SectionHeadingProps {
  title: ReactNode;
  lead?: ReactNode;
  align?: "start" | "center";
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
        <p className="max-w-xl text-pretty text-lg leading-relaxed text-muted-foreground">
          {lead}
        </p>
      ) : null}
    </div>
  );
}
