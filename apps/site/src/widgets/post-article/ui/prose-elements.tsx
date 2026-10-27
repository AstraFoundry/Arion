import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { useCopy } from "@/shared/i18n";
import { AppLink } from "@/shared/ui";

type HeadingProps = ComponentPropsWithoutRef<"h2">;

/** Section headings carry a hover anchor so a reader can link to them. */
function anchored(Tag: "h2" | "h3") {
  return function Heading({ id, children, className, ...rest }: HeadingProps) {
    const { blog } = useCopy();
    if (!id || className === "post-footnotes-label") {
      return (
        <Tag id={id} className={className} {...rest}>
          {children}
        </Tag>
      );
    }
    return (
      <Tag id={id} {...rest}>
        {children}
        <a href={`#${id}`} aria-label={blog.headingLink} className="post-anchor">
          <span aria-hidden="true">#</span>
        </a>
      </Tag>
    );
  };
}

export const H2 = anchored("h2");
export const H3 = anchored("h3");

/** Site paths keep the reader's locale; anchors and other sites pass through. */
export function ProseLink({ href = "", children, ...rest }: ComponentPropsWithoutRef<"a">) {
  if (href.startsWith("/") && !href.startsWith("//")) {
    return (
      <AppLink href={href} {...rest}>
        {children}
      </AppLink>
    );
  }
  return (
    <a href={href} {...rest}>
      {children}
    </a>
  );
}

/** Wide tables scroll inside their own box instead of widening the page. */
export function ProseTable({ children, ...rest }: ComponentPropsWithoutRef<"table">) {
  return (
    <div className="post-table" tabIndex={0}>
      <table {...rest}>{children}</table>
    </div>
  );
}

/** `<aside>` in a post: a side note set apart from the running text. */
export function Note({ children }: { children?: ReactNode }) {
  return <aside className="post-note">{children}</aside>;
}
