# Image

<!-- preview image 240 -->

`ui.Image` shows a bitmap, or an SVG in its own colors, by default at its
size in DIPs, scaled to fit when given another size.

```go
//go:embed logo.png
var logoPNG []byte

var logo, _ = ui.DecodeBitmap(logoPNG)

ui.Image(c, logo).Size(64, 64).Fit(ui.Contain).Radius(12)
```

## Bitmaps

`ui.NewBitmap` makes a bitmap of an `image.Image`, and `ui.DecodeBitmap` of
PNG, JPEG, GIF (its first frame), WebP or BMP data, turning photos upright
as their camera's EXIF orientation says. Make bitmaps once, not in the
view: MyGo keeps a bitmap on the GPU as long as you use it. A bitmap shown
much smaller than its pixels, as a photo in a thumbnail, is drawn from a
copy halved as many times as that keeps it no smaller, made once, so that
it shows every pixel's part rather than shimmering.

A bitmap keeps its pixels on the CPU as well as on the GPU, which
applications full of photos notice. Once a bitmap has shown,
`ReleasePixels` drops its CPU copies, of the picture and of the halved
ones: it draws as before from the GPU, and the pixels are made again when
something needs them, as a GPU that lost its textures or let one go after
frames without it, the CPU renderer, or a halved copy for drawing it
smaller, and a GPU's upload drops them again. A bitmap `DecodeBitmap` made
decodes them again from the data it took, which costs a fraction of the
pixels. One `NewBitmap` made converts again the `image.Image` it was made
of, which it keeps from `ReleasePixels` on, without a copy: for an image
the app keeps anyway, as a screenshot it works on at full size, the
pixels are then in memory once rather than twice. Either must stay as it
was. A bitmap `NewBitmap` made of an image the app no longer keeps (or of
an image type other than package `image`'s), a canvas's, or one `Update`
changed has nothing else to make its pixels from, and keeps them. Call
`ReleasePixels` on the main thread, as drawing does:

```go
cover, _ := ui.DecodeBitmap(fileData)
// … once the cover has shown:
cover.ReleasePixels()

shot := ui.NewBitmap(capture) // the app keeps capture
// … once the shot has shown:
shot.ReleasePixels()
```

A window that closes lets go of what it showed: its view, the elements and
the scene of its last frame, and the bitmaps they drew. A bitmap then stays
only as long as the app keeps it.

`Update` shows another picture in a bitmap, for one that changes as video
does. Of the same size, it keeps the bitmap's texture on the GPU, which
uploads the new pixels, where a new bitmap each frame would make a texture
each. An `*image.RGBA` made by `image.NewRGBA` is taken without a copy:
leave its pixels as they are until the next `Update`. Frames decoded on a
goroutine reach the bitmap through `Window.Update`, which redraws the
window:

```go
go func() {
	for frame := range frames { // *image.RGBA, a new one each time
		win.Update(func() { app.video.Update(frame) })
	}
}()
```

[`ui.Video`](video.md) plays a video's frames so, decoded ahead and each
shown as it comes due, with animated GIFs out of the box.

`Pixelated` draws a bitmap shown larger than its pixels as squares of
their colors, as pixel art and game sprites are drawn, where it would
blend neighbors smoothly. At whole multiples of its size each pixel is a
block of whole screen pixels. At other sizes, such as a zoom of 2.5, the
squares keep one size, their edges blending across a single screen pixel
rather than some squares coming out a pixel wider than others, and they
move smoothly rather than in jumps, as Ebitengine's `FilterPixelated`
draws. Shown smaller, it is as smooth as any.

```go
tiles, err := ui.DecodeBitmap(tilesPNG)
if err != nil {
	log.Fatal(err)
}
tiles.Pixelated()
ui.Image(c, tiles).Size(64, 64) // 16×16 pixels, each 4×4
```

## Fitting

`Fit` says how an image fills its box: `Contain` (the default) fits it
inside, `Cover` covers the box and crops the rest, `FillBox` stretches it,
`ScaleDown` shows it at its own size unless that does not fit, and
`NaturalSize` at its own size, cropped. `Grayscale` draws it in shades of
gray, as for what is disabled. `Radius` rounds its corners, and
`AspectRatio` keeps its proportions in a box that grows.

```go
ui.Image(c, photo).AspectRatio(16.0 / 9).Fit(ui.Cover).Radius(8)
```

`ObjectPosition(x, y)` places the picture where `Fit` crops it or leaves
room around it, as CSS's `object-position`: fractions of that room across
and down, 0 by the left or top edge, 1 by the right or bottom one, and
0.5, the default, in the middle. Its edge keywords are 0 and 1: `left
top` is `ObjectPosition(0, 0)`. An avatar whose face, near the top of the
photo, stays in view, as `object-position: 50% 15%`:

```go
ui.Image(c, photo).Size(40, 40).Radius(20).Fit(ui.Cover).ObjectPosition(0.5, 0.15)
```

## SVGs

`ui.ParseSVG` parses an SVG, and `ui.MustParseSVG` one that is part of the
program, such as an embedded file, panicking when it is in error. `Image`
shows an SVG in its own colors, with the text color for its
`currentColor`, at the size the SVG gives (its `width` and `height`, or
those of its `viewBox`) unless given another:

```go
ui.Image(c, illustration).Width(240)
```

To show an SVG in the color of the text, as an icon, use [Icon](icon.md).

MyGo draws an SVG's shapes into the GPU's atlas once for each size it shows
at, the way it draws text, and the GPU draws them from there: moving an
image, scrolling it or changing its color draws nothing again. MyGo draws
paths and the basic shapes, groups, `use` and `symbol` elements,
transforms, fills and strokes (with their joins, caps and dashes) of
colors, `currentColor` and linear and radial gradients, opacity, clip
paths, masks and style sheets of simple selectors (type, class and id). It
leaves out text, embedded images, patterns, markers and filters: convert
text to paths in your editor before exporting.

In `Draw` callbacks, `Painter.Image` draws bitmaps and SVGs too.

## Pictures of any format

`ui.DecodeImage` decodes a picture whose format isn't known up front, as
one downloaded from a link: a bitmap when `DecodeBitmap` knows its format,
else an SVG of a size. `ui.ImageSize` gives the size `Image` shows a
bitmap or an SVG at by default, in DIPs, to size it to its aspect ratio:

```go
pic, err := ui.DecodeImage(data) // once, as it arrives, not in the view
if err != nil {
	return err
}
// … in the view:
if w, h := ui.ImageSize(pic); w > 0 && h > 0 {
	ui.Image(c, pic).MaxWidthPercent(100).AspectRatio(w / h)
}
```

## Accessibility

Images are decorations that assistive technology does not see, unless
`Label` names them, as a photo that matters.
