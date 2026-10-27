import type { PostMeta } from "@/entities/post";
import { blogPostRoute } from "@/shared/config";
import { formatDate, useCopy, useLocale } from "@/shared/i18n";
import { AppLink, Dateline } from "@/shared/ui";

/**
 * The newest post, given the room of a magazine lead: its cover (when it
 * has one) beside the title and summary. The whole block is one link.
 */
export function FeaturedPost({ post }: { post: PostMeta }) {
  const { blog } = useCopy();
  return (
    <AppLink
      href={blogPostRoute(post.slug)}
      className="group grid items-center gap-8 rounded-3xl md:grid-cols-2 md:gap-12"
    >
      {post.cover ? (
        // eslint-disable-next-line @next/next/no-img-element -- static export; a fixed 4:3 box reserves the space
        <img
          src={post.cover.src}
          alt=""
          fetchPriority="high"
          referrerPolicy={post.cover.file ? undefined : "no-referrer"}
          className="aspect-[4/3] w-full rounded-2xl bg-muted object-cover outline-1 -outline-offset-1 outline-[var(--image-outline)]"
        />
      ) : null}
      <div className="flex flex-col gap-4">
        <Dateline date={post.date} label={blog.categories[post.category]} />
        <h2 className="text-balance font-display text-3xl leading-[1.1] tracking-[-0.02em] underline decoration-transparent decoration-1 underline-offset-[0.18em] transition-colors group-hover:decoration-foreground/30 sm:text-4xl">
          {post.title}
        </h2>
        <p className="max-w-xl text-pretty text-lg leading-relaxed text-muted-foreground">
          {post.summary}
        </p>
      </div>
    </AppLink>
  );
}

/** Posts as rows: date and category, then title and summary; newest first. */
export function PostList({ posts }: { posts: readonly PostMeta[] }) {
  const { blog } = useCopy();
  const locale = useLocale();
  return (
    <ul className="divide-y divide-border border-y border-border">
      {posts.map((post) => (
        <li key={post.slug}>
          <AppLink
            href={blogPostRoute(post.slug)}
            className="group grid gap-x-10 gap-y-2 py-6 sm:grid-cols-[10rem_1fr] sm:py-8"
          >
            <Dateline
              date={post.date}
              label={blog.categories[post.category]}
              className="sm:hidden"
            />
            <span className="hidden flex-col gap-1 pt-1 text-sm text-muted-foreground sm:flex">
              <time dateTime={post.date} className="tabular-nums">
                {formatDate(post.date, locale)}
              </time>
              <span>{blog.categories[post.category]}</span>
            </span>
            <span className="flex flex-col gap-2">
              <span className="text-balance font-display text-2xl leading-snug tracking-[-0.01em] underline decoration-transparent decoration-1 underline-offset-[0.2em] transition-colors group-hover:decoration-foreground/30">
                {post.title}
              </span>
              <span className="line-clamp-2 max-w-2xl text-pretty text-muted-foreground">
                {post.summary}
              </span>
            </span>
          </AppLink>
        </li>
      ))}
    </ul>
  );
}
