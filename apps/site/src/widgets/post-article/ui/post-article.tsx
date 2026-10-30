import { ArrowLeft } from "lucide-react";
import { Fragment, jsx, jsxs } from "react/jsx-runtime";
import { toJsxRuntime } from "hast-util-to-jsx-runtime";
import { getAuthors, type Post } from "@/entities/post";
import { routes } from "@/shared/config";
import { format, formatDate } from "@/shared/i18n";
import { loadMessages } from "@/shared/i18n/server";
import { mediaSize } from "@/shared/lib/server";
import { AppLink, AuthorByline, Dateline } from "@/shared/ui";
import { postComponents } from "../lib/components";
import { renderMarkdown } from "../lib/markdown";
import { TableOfContents } from "./table-of-contents";
import "./prose.css";

/** Posts with fewer sections read fine without a contents list. */
const MIN_TOC_HEADINGS = 3;

/** Authors and the revision date, when the post has them. */
async function Byline({ post }: { post: Post }) {
  const { blog } = loadMessages(post.locale);
  if (post.authors.length === 0 && !post.updated) return null;
  const authors = await getAuthors(post.authors);
  return (
    <div className="mt-8 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
      {authors.length > 0 ? <AuthorByline authors={authors} /> : null}
      {authors.length > 0 && post.updated ? <span aria-hidden="true">·</span> : null}
      {post.updated ? (
        <time dateTime={post.updated} className="tabular-nums">
          {format(blog.updated, { date: formatDate(post.updated, post.locale) })}
        </time>
      ) : null}
    </div>
  );
}

/**
 * One post: header, cover and body on a reading grid. Text keeps a ~65
 * character measure; figures, video and embeds break out wider.
 */
export async function PostArticle({ post }: { post: Post }) {
  const { blog } = loadMessages(post.locale);
  const { tree, headings } = await renderMarkdown(post.body, post.file, {
    footnotes: blog.footnotes,
    footnoteBack: (index) => format(blog.footnoteBack, { index }),
  });
  const cover = post.image && {
    ...post.image,
    ...(await mediaSize({ url: post.image.src, file: post.image.file })),
  };
  const body = toJsxRuntime(tree, { Fragment, jsx, jsxs, components: postComponents });

  return (
    <article className="post-grid pt-10 pb-24 sm:pt-16">
      <header className="flex flex-col items-start">
        <AppLink
          href={routes.blog}
          className="-ml-1 inline-flex items-center gap-1.5 rounded-full px-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          {blog.title}
        </AppLink>
        <Dateline
          date={post.date}
          label={blog.categories[post.category]}
          className="mt-10 sm:mt-14"
        />
        <h1 className="mt-4 text-balance font-display text-4xl leading-[1.08] font-normal tracking-[-0.02em] sm:text-[3.25rem]">
          {post.title}
        </h1>
        <p className="post-summary mt-6">{post.description}</p>
        <Byline post={post} />
      </header>

      {cover ? (
        // eslint-disable-next-line @next/next/no-img-element -- static export; the cover is sized at build time
        <img
          src={cover.src}
          alt={cover.alt}
          width={cover.width}
          height={cover.height}
          fetchPriority="high"
          referrerPolicy={cover.file ? undefined : "no-referrer"}
          className="post-wide post-cover"
        />
      ) : null}

      {headings.length >= MIN_TOC_HEADINGS ? <TableOfContents headings={headings} /> : null}

      <div className="post-prose">{body}</div>
    </article>
  );
}
