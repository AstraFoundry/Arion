import { DownloadCta } from "@/widgets/download-cta";
import { FeatureTour } from "@/widgets/feature-tour";
import { Hero } from "@/widgets/hero";
import { OpenSource } from "@/widgets/open-source";
import { Testimonials } from "@/widgets/testimonials";

export function HomePage() {
  return (
    <>
      <Hero />
      <FeatureTour />
      <OpenSource />
      <Testimonials />
      <DownloadCta />
    </>
  );
}
