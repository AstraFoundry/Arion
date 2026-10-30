# Site backdrop sources

Photographs behind the marketing-site illustrations and blog covers. `scripts/build-assets.mjs` crops these into `public/backdrops/`, `public/blog/<slug>/cover.webp` and `public/og.jpg`. The desktop app keeps its own backdrop.

| File | Source | Use |
| --- | --- | --- |
| `pink-motion.jpg` | "Pink flowers with motion blur effect" by Raphael Nicolas, [Unsplash](https://unsplash.com/photos/pink-flowers-with-motion-blur-effect-DjZ5xwr1cq4), Unsplash License. 2400×3600. | Hero crops and the Open Graph image |
| `gladiolus.jpg` | "Pink gladiolus flowers with green foliage in motion" by Eugene Golovesov, [Unsplash](https://unsplash.com/photos/pink-gladiolus-flowers-with-green-foliage-in-motion-Jk_9EwoLXnY), Unsplash License. 2400×3626. | Feature-tour chapters `offline` and `polish` |
| `coneflower-swirl.jpg` | "Pink coneflowers with a swirling motion effect" by Brooke Balentine, [Unsplash](https://unsplash.com/photos/pink-coneflowers-with-a-swirling-motion-effect--Ck7iqGR1pc), Unsplash License. 2000×2999. | Feature-tour chapter `mixed` |
| `rose-bush.jpg` | "Blurred pink flowers in a vibrant, green bush" by Eugene Golovesov, [Unsplash](https://unsplash.com/photos/blurred-pink-flowers-in-a-vibrant-green-bush-cTChGLob-2k), Unsplash License. 2000×3166. | Feature-tour chapter `learning` |
| `dark-roses.jpg` | "Blurred pink flowers against dark background" by Brigitte Elsner, [Unsplash](https://unsplash.com/photos/blurred-pink-flowers-against-dark-background-_NnGJYw02Z8), Unsplash License. 2200×1650. | Feature-tour chapter `keyboard` |
| `pink-grass.jpg` | "Pink flowers and green grass with motion blur" by Eugene Golovesov, [Unsplash](https://unsplash.com/photos/pink-flowers-and-green-grass-with-motion-blur-6mV2lNqyLps), Unsplash License. 2000×3012. | Cover of `where-your-dictation-goes` |
| `soft-pink.jpg` | "Soft pink flowers with green leaves in soft focus" by Ruben Mavarez, [Unsplash](https://unsplash.com/photos/soft-pink-flowers-with-green-leaves-in-soft-focus-yi0RrmWvd-8), Unsplash License. 2000×1465. | Cover of `building-like-an-experiment` |

All are free under the Unsplash License and credited on `/credits/`. Regenerate the published crops with `pnpm site:assets` from the repository root.
