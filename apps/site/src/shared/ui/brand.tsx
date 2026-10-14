import { siteConfig } from "@/shared/config";
import { cn } from "@/shared/lib";

const MARK_SRC = "/brand/arion-mark-128.png";

/** Arion's app icon: ink bars on Sunbeam yellow (desktop `cmd/arion-icon`). */
export function BrandMark({ className }: { className?: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- static export, pre-sized asset
    <img
      src={MARK_SRC}
      alt=""
      width={128}
      height={128}
      draggable={false}
      className={cn("size-7 shrink-0", className)}
    />
  );
}

/** The product name as the wordmark: display face, no decoration. */
export function BrandWord({ className }: { className?: string }) {
  return (
    <span className={cn("font-display whitespace-nowrap", className)}>{siteConfig.name}</span>
  );
}
