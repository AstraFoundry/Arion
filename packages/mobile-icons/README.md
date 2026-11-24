# Mobile icons

`icons.json` names every icon of the iOS and Android apps. Each is an SVG in `svg/` (Lucide 0.460.0 icons the desktop does not vendor, such as the Shift arrows, and the single-color Gemini mark from LobeHub Icons 1.97.0) or in the desktop's vendored set `apps/desktop/ui/shared/icons/svg` (Lucide, LobeHub provider marks, Google and GitHub). Licenses are beside the files.

`pnpm mobile:generate` runs `scripts/generate-mobile-icons.mjs`, which writes the iOS asset catalog (`ArionKit/Resources/Icons.xcassets`) with the `Icon` enum, and the Android vector drawables (`design/res/drawable/icon_*.xml`) with the `ArionIcon` enum. `pnpm mobile:check` fails when they differ. Icons are never drawn or converted by hand: add the official SVG and its name here.

Names in `icons` are tinted; names in `original` keep their colors. Views show icons on a fixed square frame (`UIImageView(icon:size:)`, `IconView`), so rows of different icons align.

Names in `filled` also get derived `-fill` (closed outlines filled, open strokes dropped) and `-cut` (only the open strokes) variants for pressed and active keys; the app knocks the cut out of the fill or draws it over it. They come from the same official SVG, so nothing is drawn by hand.
