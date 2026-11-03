import Script from "next/script";
import { googleAnalyticsMeasurementId } from "@/shared/config";

/** Google Analytics 4. Renders nothing when no measurement id is configured. */
export function GoogleAnalytics() {
  const measurementId = googleAnalyticsMeasurementId();
  if (!measurementId) return null;
  return (
    <>
      <Script
        src={`https://www.googletagmanager.com/gtag/js?id=${measurementId}`}
        strategy="afterInteractive"
      />
      <Script id="google-analytics" strategy="afterInteractive">
        {`window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config',${JSON.stringify(measurementId)});`}
      </Script>
    </>
  );
}
