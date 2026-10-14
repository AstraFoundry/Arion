import nextCoreWebVitals from "eslint-config-next/core-web-vitals";
import nextTypescript from "eslint-config-next/typescript";

// FSD layer order: app -> pages -> widgets -> features -> entities -> shared.
// Each layer may only import from the layers below it, and slices are reached
// through their public `index.ts` only.
const LAYERS = ["app", "pages", "widgets", "features", "entities", "shared"];

const deepSliceImport = {
  group: ["@/pages/*/*", "@/widgets/*/*", "@/features/*/*", "@/entities/*/*"],
  message: "Import slices through their public index.ts (e.g. '@/widgets/site-header').",
};

function layerRule(layer) {
  const above = LAYERS.slice(0, LAYERS.indexOf(layer));
  const patterns = [deepSliceImport];
  if (above.length > 0) {
    patterns.push({
      group: above.map((name) => `@/${name}`).concat(above.map((name) => `@/${name}/**`)),
      message: `The '${layer}' layer cannot import from layers above it (${above.join(", ")}).`,
    });
  }
  return {
    files: [`src/${layer}/**/*.{ts,tsx}`],
    rules: { "no-restricted-imports": ["error", { patterns }] },
  };
}

const config = [
  {
    ignores: [".next/**", "out/**", "node_modules/**", "next-env.d.ts", "src/shared/beui/**"],
  },
  ...nextCoreWebVitals,
  ...nextTypescript,
  ...LAYERS.map(layerRule),
  {
    files: ["app/**/*.{ts,tsx}"],
    rules: { "no-restricted-imports": ["error", { patterns: [deepSliceImport] }] },
  },
];

export default config;
