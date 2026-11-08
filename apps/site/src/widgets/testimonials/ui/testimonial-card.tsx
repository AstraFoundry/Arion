import type { Testimonial } from "@/entities/testimonial";
import { format, useCopy } from "@/shared/i18n";
import { AuthorAvatar, XMark } from "@/shared/ui";

/**
 * One post as X's display rules ask: avatar, display name, @handle and the
 * X logo with the text as written. The whole card links to the post (or the
 * profile) through the name.
 */
export function TestimonialCard({ testimonial }: { testimonial: Testimonial }) {
  const copy = useCopy();
  return (
    <figure className="relative flex h-full w-[19rem] flex-col gap-4 rounded-2xl border border-border bg-card p-5 text-start shadow-1 transition-colors hover:bg-muted sm:w-[22rem]">
      <figcaption className="flex items-center gap-3 [--avatar-size:--spacing(10)]">
        <AuthorAvatar
          author={{ name: testimonial.name, profileUrl: testimonial.href, avatar: testimonial.avatar }}
        />
        <span className="flex min-w-0 flex-col">
          <a
            href={testimonial.href}
            aria-label={format(copy.testimonials.profile, { name: testimonial.name })}
            className="truncate text-sm font-medium text-foreground after:absolute after:inset-0 after:rounded-2xl focus-visible:outline-none focus-visible:after:outline-2 focus-visible:after:outline-offset-2 focus-visible:after:outline-ring"
          >
            {testimonial.name}
          </a>
          <span className="truncate text-sm text-muted-foreground">@{testimonial.handle}</span>
        </span>
        <XMark className="ms-auto size-4 shrink-0 text-foreground" />
      </figcaption>
      <blockquote lang={testimonial.lang} className="flex flex-col gap-3 text-pretty text-[0.9375rem] leading-relaxed text-foreground">
        {testimonial.paragraphs.map((paragraph) => (
          <p key={paragraph}>{paragraph}</p>
        ))}
      </blockquote>
    </figure>
  );
}
