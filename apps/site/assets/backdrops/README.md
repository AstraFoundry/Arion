# Site backdrop sources

Photographs behind the marketing-site illustrations and blog covers. `scripts/build-assets.mjs` crops these into `public/backdrops/`, `public/blog/<slug>/cover.webp` and `public/og.jpg`. The desktop app keeps its own backdrop.

| File | Source | Use |
| --- | --- | --- |
| `misty-hillside.jpg` | "A field of pink flowers on a misty hillside" by bahareh moradian, [Unsplash](https://unsplash.com/photos/a-field-of-pink-flowers-on-a-misty-hillside-WimA_ve2yA4), Unsplash License. Downloaded at 2400×1600. | Hero ground |
| `coastal-blur.jpg` | "Rolling coastal landscape with blurred grass" by T (@tanyabarrow), [Unsplash](https://unsplash.com/photos/rolling-coastal-landscape-with-blurred-grass-Zy7PAQgRswQ), Unsplash License. Downloaded at 2400×1609. | Shared ground of the highlights grid |
| `pink-motion.jpg` | "Pink flowers with motion blur effect" by Raphael Nicolas, [Unsplash](https://unsplash.com/photos/pink-flowers-with-motion-blur-effect-DjZ5xwr1cq4), Unsplash License. 2400×3600. | The Open Graph image |
| `pink-grass.jpg` | "Pink flowers and green grass with motion blur" by Eugene Golovesov, [Unsplash](https://unsplash.com/photos/pink-flowers-and-green-grass-with-motion-blur-6mV2lNqyLps), Unsplash License. 2000×3012. | Cover of `where-your-dictation-goes` |
| `soft-pink.jpg` | "Soft pink flowers with green leaves in soft focus" by Ruben Mavarez, [Unsplash](https://unsplash.com/photos/soft-pink-flowers-with-green-leaves-in-soft-focus-yi0RrmWvd-8), Unsplash License. 2000×1465. | Cover of `building-like-an-experiment` |

All are free under the Unsplash License and credited on `/credits/`. Regenerate the published crops with `pnpm site:assets` from the repository root.
