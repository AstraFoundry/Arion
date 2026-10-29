import { formatList, useLocale } from "@/shared/i18n";
import { cn } from "@/shared/lib";
import { Avatar, AvatarFallback, AvatarGroup, AvatarImage } from "./avatar";

export interface AuthorProfile {
  name: string;
  profileUrl: string;
  /** Square avatar image URL. */
  avatar: string;
}

/**
 * The avatar beside a visible name, so it stays out of the accessibility
 * tree; the initial shows until the image has loaded.
 */
function AuthorAvatar({ author }: { author: AuthorProfile }) {
  return (
    <Avatar aria-hidden="true">
      <AvatarImage src={author.avatar} alt="" />
      <AvatarFallback>{Array.from(author.name)[0].toUpperCase()}</AvatarFallback>
    </Avatar>
  );
}

/** A post byline: each author's avatar and name, linked to their profile. */
export function AuthorByline({ authors, className }: { authors: readonly AuthorProfile[]; className?: string }) {
  return (
    <address className={cn("flex flex-wrap items-center gap-x-4 gap-y-2 not-italic", className)}>
      {authors.map((author) => (
        <a
          key={author.profileUrl}
          href={author.profileUrl}
          className="inline-flex items-center gap-2 rounded-full pe-1 font-medium text-foreground transition-colors [--avatar-size:--spacing(6)] hover:text-muted-foreground"
        >
          <AuthorAvatar author={author} />
          {author.name}
        </a>
      ))}
    </address>
  );
}

/** Authors in a list row: overlapping avatars, then the names in one quiet line. */
export function AuthorSummary({ authors, className }: { authors: readonly AuthorProfile[]; className?: string }) {
  const locale = useLocale();
  return (
    <span className={cn("flex items-center gap-2 text-sm text-muted-foreground", className)}>
      <AvatarGroup className="[--avatar-size:--spacing(5)]">
        {authors.map((author) => (
          <AuthorAvatar key={author.profileUrl} author={author} />
        ))}
      </AvatarGroup>
      <span>{formatList(authors.map((author) => author.name), locale)}</span>
    </span>
  );
}
