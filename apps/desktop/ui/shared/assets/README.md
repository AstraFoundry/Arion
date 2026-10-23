# UI assets

| File | Source | Regenerate |
| --- | --- | --- |
| `mark.png` | `resources/icon.png`, rendered by `cmd/arion-icon` | `go run ./cmd/arion-icon resources/icon.png && sips -Z 128 resources/icon.png --out ui/shared/assets/mark.png` (from `apps/desktop`) |
| `astrafoundry.png` | AstraFoundry organization avatar from GitHub (`https://avatars.githubusercontent.com/u/283552033?s=128`) | `curl -sL 'https://avatars.githubusercontent.com/u/283552033?v=4&s=128' -o astrafoundry.png` |
| `feishu.png` | Feishu's colored logo, the icon `www.feishu.cn` serves (`https://p16-hera-overseas.larksuitecdn.com/tos-mya-i-lojyj5t9n9/9b87226605154fb7b8141a9c94de22e9.png~tplv-lojyj5t9n9-origin.image`, 700 px). Feishu publishes no license for it; it is a ByteDance trademark used only to link to the Arion Feishu group | `sips -Z 96 <download> --out feishu.png` |
| `contributors/` | GitHub's contributor list for `AstraFoundry/Arion` (bots excluded, GitHub's order) and each avatar at 96 px; `contributors.json` records each avatar's SHA-256 | `pnpm contributors:update` (`GITHUB_TOKEN` optional) |
| `holdkey/` | Drawn for Arion with the Anatomy kit (`.agents/skills/anatomy`, MIT): a keyboard per platform and appearance, and each dictation key lit, at rest and pressed | `pnpm figures:holdkey` |
| `backdrop.jpg` | "Flowers falling" by Roma Kaiuk, [Unsplash](https://unsplash.com/photos/a-group-of-pink-flowers-z51SGO5m5ec), Unsplash License | Downscaled to 800×1000, JPEG quality 78: `sips -s format jpeg -s formatOptions 78 -Z 1000 <original> --out backdrop.jpg` |

The Unsplash License allows commercial use without permission or attribution. Photos used as illustration backdrops must already be soft (motion blur or shallow focus) so the mock UI above them stays the subject. Keep each under 200 KB.
