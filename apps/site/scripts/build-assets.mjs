// Regenerates the site's raster assets in public/ from repository sources:
//   apps/site/assets/backdrops/misty-hillside.jpg → hero ground
//   apps/site/assets/backdrops/coastal-blur.jpg   → the highlights grid's shared ground
//   apps/site/assets/backdrops/pink-motion.jpg    → public/og.jpg
//   apps/site/assets/backdrops/*.jpg              → blog covers
//   apps/desktop/resources/icon.png             → public/brand/*.png, the mark on og.jpg
//   apps/desktop/ui/shared/assets/feishu.png    → public/brand/feishu.png (community link)
// The desktop app keeps its own backdrop. Run with
// `pnpm --filter @arion/site assets:build` and commit the output.
import { copyFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const site = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const desktop = path.resolve(site, "../desktop");
const sources = path.join(site, "assets/backdrops");
const ogPhoto = path.join(sources, "pink-motion.jpg");
const source = (name) => path.join(sources, name);
const icon = path.join(desktop, "resources/icon.png");
const feishuLogo = path.join(desktop, "ui/shared/assets/feishu.png");
const out = (...parts) => path.join(site, "public", ...parts);

await mkdir(out("backdrops"), { recursive: true });
await mkdir(out("brand"), { recursive: true });

// misty-hillside.jpg is 2400×1600: mist across the upper half carries the
// hero copy, pink flowers fill the lower third. Full frame, downscaled.
for (const width of [960, 1600, 2400]) {
  await sharp(source("misty-hillside.jpg"))
    .resize({ width, kernel: "lanczos3" })
    .webp({ quality: width > 1600 ? 72 : 78 })
    .toFile(out("backdrops", `hillside-${width}.webp`));
}

// coastal-blur.jpg is 2400×1609: one photograph behind every card of the
// highlights grid, each card a window onto its own part of the frame.
for (const width of [960, 1600]) {
  await sharp(source("coastal-blur.jpg"))
    .resize({ width, kernel: "lanczos3" })
    .webp({ quality: 78 })
    .toFile(out("backdrops", `coastal-${width}.webp`));
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

// The desktop's Feishu logo (source in apps/desktop/ui/shared/assets/README.md), as is.
await copyFile(feishuLogo, out("brand", "feishu.png"));

const ogMark = await sharp(icon).resize(280).png().toBuffer();
await sharp(ogPhoto)
  .extract({ left: 0, top: 1800, width: 2400, height: 1500 })
  .resize({ width: 1200, height: 630, fit: "cover", position: "centre", kernel: "lanczos3" })
  .composite([{ input: ogMark, gravity: "center" }])
  .jpeg({ quality: 82, mozjpeg: true })
  .toFile(out("og.jpg"));
