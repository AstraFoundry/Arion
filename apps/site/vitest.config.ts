import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  test: {
    include: ["tests/**/*.test.ts"],
    environment: "node",
    env: {
      NEXT_PUBLIC_SITE_URL: "https://arion.astrafoundry.si",
      NEXT_PUBLIC_PLATFORM_URL: "https://astrafoundry.si",
      NEXT_PUBLIC_CENTER_URL: "https://api.arion.astrafoundry.si",
    },
  },
});
