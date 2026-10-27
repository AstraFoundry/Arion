import { listPosts } from "@/entities/post";
import { useCopy, useLocale } from "@/shared/i18n";
import { Container, SectionHeading } from "@/shared/ui";
import { FeaturedPost, PostList } from "@/widgets/post-list";

export function BlogPage() {
  const { blog } = useCopy();
  const [latest, ...earlier] = listPosts(useLocale());
  return (
    <Container className="flex flex-col gap-16 py-16 sm:gap-20 sm:py-24">
      <SectionHeading as="h1" title={blog.title} lead={blog.lead} />
      <FeaturedPost post={latest} />
      {earlier.length > 0 ? (
        <section className="flex flex-col gap-6">
          <h2 className="font-display text-2xl leading-tight">{blog.allPosts}</h2>
          <PostList posts={earlier} />
        </section>
      ) : null}
    </Container>
  );
}
