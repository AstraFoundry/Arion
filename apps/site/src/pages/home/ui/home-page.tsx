import { DownloadCta } from "@/widgets/download-cta";
import { FeatureTour } from "@/widgets/feature-tour";
import { Hero } from "@/widgets/hero";
import { OpenSource } from "@/widgets/open-source";
import { PrivacySummary } from "@/widgets/privacy-summary";

export function HomePage() {
  return (
    <>
      <Hero />
      <FeatureTour />
      <PrivacySummary />
      <OpenSource />
      <DownloadCta />
    </>
  );
}
