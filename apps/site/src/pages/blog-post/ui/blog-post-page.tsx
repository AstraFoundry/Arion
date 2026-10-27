import { getPost, listPosts } from "@/entities/post";
import { useCopy, useLocale } from "@/shared/i18n";
import { Container } from "@/shared/ui";
import { PostArticle } from "@/widgets/post-article";
import { PostList } from "@/widgets/post-list";

/** How many other posts follow an article. */
const MORE_POSTS = 3;

export function BlogPostPage({ slug }: { slug: string }) {
  const { blog } = useCopy();
  const locale = useLocale();
  const more = listPosts(locale)
    .filter((post) => post.slug !== slug)
    .slice(0, MORE_POSTS);
  return (
    <>
      <PostArticle post={getPost(slug, locale)} />
      {more.length > 0 ? (
        <Container as="section" className="flex flex-col gap-6 pb-24">
          <h2 className="font-display text-2xl leading-tight">{blog.morePosts}</h2>
          <PostList posts={more} />
        </Container>
      ) : null}
    </>
  );
}
