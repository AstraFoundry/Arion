# UI assets

| File | Source | Regenerate |
| --- | --- | --- |
| `mark.png` | `resources/icon.png`, rendered by `cmd/arion-icon` | `go run ./cmd/arion-icon resources/icon.png && sips -Z 128 resources/icon.png --out ui/shared/assets/mark.png` (from `apps/desktop`) |
| `backdrop.jpg` | "Flowers falling" by Roma Kaiuk, [Unsplash](https://unsplash.com/photos/a-group-of-pink-flowers-z51SGO5m5ec), Unsplash License | Downscaled to 800×1000, JPEG quality 78: `sips -s format jpeg -s formatOptions 78 -Z 1000 <original> --out backdrop.jpg` |

The Unsplash License allows commercial use without permission; the credit is shown in Settings › About (`copy.AboutCredits`). Photos used as illustration backdrops must already be soft (motion blur or shallow focus) so the mock UI above them stays the subject. Keep each under 200 KB.
