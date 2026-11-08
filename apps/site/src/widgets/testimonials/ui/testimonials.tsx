import { getLocale } from "next-intl/server";
import { getTestimonials } from "@/entities/testimonial";
import { toLocale } from "@/shared/i18n/server";
import { TestimonialWall } from "./testimonial-wall";

/** Posts about Arion on X, scrolling in two rows; profiles are read at build time. */
export async function Testimonials() {
  const testimonials = await getTestimonials(toLocale(await getLocale()));
  return <TestimonialWall testimonials={testimonials} />;
}
