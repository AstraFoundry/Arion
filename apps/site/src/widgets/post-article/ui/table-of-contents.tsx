import { useId } from "react";
import { useCopy } from "@/shared/i18n";
import type { PostHeading } from "../lib/rehype-post";

/** The post's sections, listed before the body like a printed article's. */
export function TableOfContents({ headings }: { headings: readonly PostHeading[] }) {
  const { blog } = useCopy();
  const titleId = useId();
  return (
    <nav aria-labelledby={titleId} className="post-toc">
      <h2 id={titleId}>{blog.contents}</h2>
      <ol>
        {headings.map((heading) => (
          <li key={heading.id}>
            <a href={`#${heading.id}`}>{heading.text}</a>
          </li>
        ))}
      </ol>
    </nav>
  );
}
