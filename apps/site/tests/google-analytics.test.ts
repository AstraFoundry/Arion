import { describe, expect, it } from "vitest";
import { googleAnalyticsMeasurementId } from "@/shared/config";

describe("googleAnalyticsMeasurementId", () => {
  it("accepts a GA4 measurement id", () => {
    expect(googleAnalyticsMeasurementId("G-B7VQ241TTC")).toBe("G-B7VQ241TTC");
    expect(googleAnalyticsMeasurementId("  G-ABC123  ")).toBe("G-ABC123");
  });

  it("ignores missing and malformed values", () => {
    expect(googleAnalyticsMeasurementId(undefined)).toBeNull();
    expect(googleAnalyticsMeasurementId("")).toBeNull();
    expect(googleAnalyticsMeasurementId(" ")).toBeNull();
    expect(googleAnalyticsMeasurementId("UA-12345-1")).toBeNull();
    expect(googleAnalyticsMeasurementId("G-bad id")).toBeNull();
    expect(googleAnalyticsMeasurementId("G-<script>")).toBeNull();
  });
});
