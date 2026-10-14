import { DownloadCta } from "@/widgets/download-cta";
import { Hero } from "@/widgets/hero";
import { OpenSource } from "@/widgets/open-source";
import { PrivacySummary } from "@/widgets/privacy-summary";
import { Values } from "@/widgets/values";

export function HomePage() {
  return (
    <>
      <Hero />
      <Values />
      <PrivacySummary />
      <OpenSource />
      <DownloadCta />
    </>
  );
}
