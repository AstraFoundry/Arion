// Regenerates the site's raster assets in public/ from the desktop app's
// sources, so the site and the app share one photograph and one icon:
//   apps/desktop/ui/shared/assets/backdrop.jpg  → public/backdrops/*.webp (hero and
//                                                 feature-tour crops), public/og.jpg
//   apps/desktop/resources/icon.png             → public/brand/*.png
// Run with `pnpm --filter @arion/site assets:build` and commit the output.
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const site = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const desktop = path.resolve(site, "../desktop");
const backdrop = path.join(desktop, "ui/shared/assets/backdrop.jpg");
const icon = path.join(desktop, "resources/icon.png");
const out = (...parts) => path.join(site, "public", ...parts);

await mkdir(out("backdrops"), { recursive: true });
await mkdir(out("brand"), { recursive: true });

// The source is 800×1000. The wide crop keeps the middle band (the pink and
// white blossoms); the photo is motion-blurred, so upscaling it stays soft.
const wideBand = { left: 0, top: 360, width: 800, height: 500 };
for (const width of [1200, 1600]) {
  await sharp(backdrop)
    .extract(wideBand)
    .resize({ width, kernel: "lanczos3" })
    .webp({ quality: 80 })
    .toFile(out("backdrops", `flowers-wide-${width}.webp`));
}
await sharp(backdrop).webp({ quality: 80 }).toFile(out("backdrops", "flowers-tall-800.webp"));

// The feature tour gives each chapter its own 4:5 region of the photograph,
// so the sticky stage changes ground as the reader moves through it. Keys are
// the chapter ids in src/widgets/feature-tour/config/chapters.ts.
const tourRegions = {
  offline: { left: 120, top: 0, width: 480, height: 600 },
  mixed: { left: 320, top: 0, width: 480, height: 600 },
  learning: { left: 200, top: 290, width: 480, height: 600 },
  polish: { left: 0, top: 320, width: 480, height: 600 },
  keyboard: { left: 180, top: 400, width: 480, height: 600 },
};
for (const [chapter, region] of Object.entries(tourRegions)) {
  await sharp(backdrop)
    .extract(region)
    .resize({ width: 800, kernel: "lanczos3" })
    .webp({ quality: 78 })
    .toFile(out("backdrops", `tour-${chapter}-800.webp`));
}

// The icon has a transparent margin around its rounded square; web icons use
// the square itself.
const square = { left: 96, top: 100, width: 832, height: 832 };
const mark = sharp(icon).extract(square);
await mark.clone().resize(128).png().toFile(out("brand", "arion-mark-128.png"));
await mark.clone().resize(64).png().toFile(out("brand", "favicon-64.png"));
await mark
  .clone()
  .flatten({ background: "#ffcc00" })
  .resize(180)
  .png()
  .toFile(out("brand", "apple-touch-icon.png"));

const ogMark = await sharp(icon).resize(280).png().toBuffer();
await sharp(backdrop)
  .extract({ left: 0, top: 340, width: 800, height: 420 })
  .resize({ width: 1200, height: 630, fit: "cover", kernel: "lanczos3" })
  .composite([{ input: ogMark, gravity: "center" }])
  .jpeg({ quality: 82, mozjpeg: true })
  .toFile(out("og.jpg"));
