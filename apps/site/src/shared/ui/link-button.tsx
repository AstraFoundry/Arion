import {
  LinkButton as SunbeamLinkButton,
  type ButtonSize,
  type ButtonVariant,
} from "@astrafoundry/design/ui";
import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { AppLink } from "./app-link";

interface LinkButtonProps extends Omit<ComponentPropsWithoutRef<"a">, "href"> {
  href: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  children: ReactNode;
}

function isExternal(href: string) {
  return /^https?:\/\//.test(href);
}

/**
 * Sunbeam's `LinkButton`. Internal routes use Next's client navigation in the
 * current locale; external links (releases, source, the AstraFoundry account)
 * are plain anchors in the same tab.
 */
export function LinkButton({ href, ...rest }: LinkButtonProps) {
  if (isExternal(href)) return <SunbeamLinkButton href={href} {...rest} />;
  return <SunbeamLinkButton as={AppLink} href={href} {...rest} />;
}
