import { getTestimonials } from "@/entities/testimonial";
import { TestimonialWall } from "./testimonial-wall";

/** Posts about Arion on X, scrolling in two rows; profiles are read at build time. */
export async function Testimonials() {
  const testimonials = await getTestimonials();
  return <TestimonialWall testimonials={testimonials} />;
}
