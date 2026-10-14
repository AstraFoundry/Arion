import { cn } from "@/shared/lib";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "inverse";
export type ButtonSize = "sm" | "md" | "lg";

const VARIANT: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-primary-foreground shadow-1 hover:bg-[color-mix(in_oklab,var(--primary)_88%,var(--foreground))]",
  secondary: "border border-border bg-card text-foreground shadow-1 hover:bg-muted",
  ghost: "text-muted-foreground hover:bg-muted hover:text-foreground",
  inverse:
    "border border-substrate-edge bg-substrate-surface text-substrate-hi hover:bg-substrate-raised",
};

const SIZE: Record<ButtonSize, string> = {
  sm: "h-8 gap-1.5 px-3 text-sm",
  md: "h-10 gap-2 px-4 text-sm",
  lg: "h-12 gap-2 px-5 text-base",
};

/** Shared look for buttons and links that act as buttons. */
export function buttonClasses(
  variant: ButtonVariant,
  size: ButtonSize,
  className?: string,
) {
  return cn(
    "inline-flex shrink-0 select-none items-center justify-center rounded-full font-medium whitespace-nowrap",
    "transition-[background-color,color,box-shadow,scale] duration-150 ease-out active:scale-[0.97]",
    "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
    "disabled:pointer-events-none disabled:opacity-60",
    VARIANT[variant],
    SIZE[size],
    className,
  );
}
