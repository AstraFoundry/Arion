// Regenerates the site's raster assets in public/ from repository sources:
//   apps/site/assets/backdrops/pink-motion.jpg  → hero crops and public/og.jpg
//   apps/site/assets/backdrops/*.jpg            → feature-tour crops and blog covers
//   apps/desktop/resources/icon.png             → public/brand/*.png, the mark on og.jpg
// The desktop app keeps its own backdrop. Run with
// `pnpm --filter @arion/site assets:build` and commit the output.
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const site = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const desktop = path.resolve(site, "../desktop");
const sources = path.join(site, "assets/backdrops");
const heroPhoto = path.join(sources, "pink-motion.jpg");
const source = (name) => path.join(sources, name);
const icon = path.join(desktop, "resources/icon.png");
const out = (...parts) => path.join(site, "public", ...parts);

await mkdir(out("backdrops"), { recursive: true });
await mkdir(out("brand"), { recursive: true });

// pink-motion.jpg is 2400×3600. The wide band keeps a readable blossom and the
// darker green at the bottom of the frame; both outputs are downscales.
const wideBand = { left: 0, top: 1800, width: 2400, height: 1500 };
for (const width of [1200, 1600]) {
  await sharp(heroPhoto)
    .extract(wideBand)
    .resize({ width, kernel: "lanczos3" })
    .webp({ quality: 80 })
    .toFile(out("backdrops", `flowers-wide-${width}.webp`));
}
await sharp(heroPhoto)
  .resize({ width: 800, kernel: "lanczos3" })
  .webp({ quality: 80 })
  .toFile(out("backdrops", "flowers-tall-800.webp"));

// Each chapter is its own photograph, cropped to 4:5, so the sticky stage
// changes ground as the reader moves. The two gladiolus chapters keep
// different regions of that frame (2400×3626). Keys are the chapter ids in
// src/widgets/feature-tour/config/chapters.ts.
const tourChapters = {
  offline: { file: "gladiolus.jpg", region: { left: 480, top: 180, width: 1440, height: 1800 } },
  mixed: { file: "coneflower-swirl.jpg" },
  learning: { file: "rose-bush.jpg" },
  polish: { file: "gladiolus.jpg", region: { left: 480, top: 1700, width: 1440, height: 1800 } },
  keyboard: { file: "dark-roses.jpg" },
};
for (const [chapter, spec] of Object.entries(tourChapters)) {
  let image = sharp(source(spec.file));
  if (spec.region) image = image.extract(spec.region);
  await image
    .resize({
      width: 800,
      height: 1000,
      fit: "cover",
      position: spec.region ? "centre" : "attention",
      kernel: "lanczos3",
    })
    .webp({ quality: 78 })
    .toFile(out("backdrops", `tour-${chapter}-800.webp`));
}

// Blog covers are 4:3, the shape of the index lead.
const covers = {
  "where-your-dictation-goes": "pink-grass.jpg",
  "building-like-an-experiment": "soft-pink.jpg",
};
for (const [slug, file] of Object.entries(covers)) {
  await mkdir(out("blog", slug), { recursive: true });
  await sharp(source(file))
    .resize({ width: 1600, height: 1200, fit: "cover", position: "attention", kernel: "lanczos3" })
    .webp({ quality: 80 })
    .toFile(out("blog", slug, "cover.webp"));
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
await sharp(heroPhoto)
  .extract(wideBand)
  .resize({ width: 1200, height: 630, fit: "cover", position: "centre", kernel: "lanczos3" })
  .composite([{ input: ogMark, gravity: "center" }])
  .jpeg({ quality: 82, mozjpeg: true })
  .toFile(out("og.jpg"));
