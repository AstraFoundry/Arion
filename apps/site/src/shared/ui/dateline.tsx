import { formatDate, useLocale } from "@/shared/i18n";
import { cn } from "@/shared/lib";

interface DatelineProps {
  /** `YYYY-MM-DD`. */
  date: string;
  /** Shown after the date, such as a category. */
  label: string;
  className?: string;
}

/** A publication date and a label on one quiet line: "Oct 27, 2026 · Product". */
export function Dateline({ date, label, className }: DatelineProps) {
  const locale = useLocale();
  return (
    <p className={cn("flex items-center gap-2 text-sm text-muted-foreground", className)}>
      <time dateTime={date} className="tabular-nums">
        {formatDate(date, locale)}
      </time>
      <span aria-hidden="true">·</span>
      <span>{label}</span>
    </p>
  );
}
