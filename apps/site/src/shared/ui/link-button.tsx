import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { AppLink } from "./app-link";
import { buttonClasses, type ButtonSize, type ButtonVariant } from "./button-styles";

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
 * A link styled as a button. Internal routes use Next's client navigation in
 * the current locale; external links (releases, source, the AstraFoundry
 * account) are plain anchors in the same tab.
 */
export function LinkButton({
  href,
  variant = "primary",
  size = "md",
  className,
  children,
  ...rest
}: LinkButtonProps) {
  const classes = buttonClasses(variant, size, className);
  if (isExternal(href)) {
    return (
      <a href={href} className={classes} {...rest}>
        {children}
      </a>
    );
  }
  return (
    <AppLink href={href} className={classes} {...rest}>
      {children}
    </AppLink>
  );
}
