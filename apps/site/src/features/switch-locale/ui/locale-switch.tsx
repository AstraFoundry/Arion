"use client";

import {
  LocaleList as SunbeamLocaleList,
  LocaleSwitch as SunbeamLocaleSwitch,
} from "@astrafoundry/design/ui";
import { useCopy } from "@/shared/i18n";
import { useLocaleLinks } from "../lib/use-locale-links";

/**
 * Sunbeam's language controls with this site's links. Each locale has its
 * own document (and `<html lang>`), so switching is a full page load.
 */
export function LocaleSwitch({ className }: { className?: string }) {
  const copy = useCopy();
  return (
    <SunbeamLocaleSwitch
      label={copy.common.language}
      links={useLocaleLinks()}
      className={className}
    />
  );
}

/** Inline list for the mobile menu. */
export function LocaleList({ className }: { className?: string }) {
  const copy = useCopy();
  return (
    <SunbeamLocaleList
      label={copy.common.language}
      links={useLocaleLinks()}
      className={className}
    />
  );
}
