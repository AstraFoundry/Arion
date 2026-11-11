//go:build windows && (amd64 || arm64)

package windows

import (
	"math"
	"testing"
	"unsafe"
)

// The highlights of menu items cover their pixels inside fully, those past
// the rounded corners not at all, and those the corners cross partly.
func TestRoundedCoverage(t *testing.T) {
	r := &rect{0, 0, 40, 20}
	for _, c := range []struct {
		x, y     float64
		min, max float64
	}{
		{20.5, 10.5, 1, 1}, // the middle
		{0.5, 10.5, 1, 1},  // the left edge, between the corners
		{20.5, 0.5, 1, 1},  // the top edge
		{0.5, 0.5, 0, 0},   // past the top-left corner's curve
		{39.5, 19.5, 0, 0}, // past the bottom-right one's
		{1.5, 1.5, 0.01, 0.99},
	} {
		if got := roundedCoverage(c.x, c.y, r, 4); got < c.min || got > c.max {
			t.Errorf("coverage at (%v, %v) = %v, want %v to %v", c.x, c.y, got, c.min, c.max)
		}
	}
}

func TestMenuColors(t *testing.T) {
	if got := dibColor(0x332211); got != 0x112233 {
		t.Errorf("dibColor(0x332211) = %#x, want 0x112233", got)
	}
	if got := blend(0xFFFFFF, 0x000000, 0.5); got != 0x808080 {
		t.Errorf("blend halfway = %#x, want 0x808080", got)
	}
	if got := blend(0x102030, 0xA0B0C0, 0); got != 0xA0B0C0 {
		t.Errorf("blend with none = %#x, want the background", got)
	}
}

// Menus have the material behind them only where DWM surely draws it, and
// WinUI's solid background elsewhere.
func TestMenuGlassConditions(t *testing.T) {
	on := glassConditions{fluent: true, backdrops: true, transparency: true}
	if !on.glass() {
		t.Fatalf("%+v: no material, want it", on)
	}
	for name, off := range map[string]func(*glassConditions){
		"Windows 10":               func(c *glassConditions) { c.fluent = false },
		"Windows 11 before 22H2":   func(c *glassConditions) { c.backdrops = false },
		"a contrast theme":         func(c *glassConditions) { c.contrast = true },
		"transparency effects off": func(c *glassConditions) { c.transparency = false },
		"energy saver":             func(c *glassConditions) { c.energySaver = true },
		"a remote session":         func(c *glassConditions) { c.remote = true },
		"no GPU":                   func(c *glassConditions) { c.software = true },
	} {
		c := on
		off(&c)
		if c.glass() {
			t.Errorf("with %s: the material, want WinUI's solid background", name)
		}
	}
}

// DWM composes without a GPU on Microsoft's adapters, the Basic Render
// Driver of a virtual machine, and on software ones, and with any other.
func TestSoftwareAdapter(t *testing.T) {
	for _, c := range []struct {
		vendor, flags uint32
		want          bool
	}{
		{0x1414, 0, true}, // the Basic Render Driver
		{0x10de, 2, true}, // DXGI_ADAPTER_FLAG_SOFTWARE
		{0x10de, 0, false},
		{0x8086, 0, false},
		{0x1ab8, 0, false}, // Parallels' 3D driver
	} {
		if got := softwareAdapter(c.vendor, c.flags); got != c.want {
			t.Errorf("vendor %#x, flags %d: software %v, want %v", c.vendor, c.flags, got, c.want)
		}
	}
}

// The solid backgrounds are WinUI's fallbacks for its Acrylic, and the
// opaque highlight and separators are its translucent ones over them, as
// they show over the material.
func TestFluentPalette(t *testing.T) {
	near := func(a, b uint32) bool {
		for shift := 0; shift < 24; shift += 8 {
			if d := int(a>>shift&0xFF) - int(b>>shift&0xFF); d < -1 || d > 1 {
				return false
			}
		}
		return true
	}
	for _, c := range []struct {
		dark bool
		back uint32
	}{{false, 0xF9F9F9}, {true, 0x2C2C2C}} {
		p := fluentPalette(c.dark)
		if p.back != c.back {
			t.Errorf("dark %v: background %#06x, want %#06x", c.dark, p.back, c.back)
		}
		if got := blend(dibColor(p.wash.color), dibColor(p.back), p.wash.alpha); !near(got, dibColor(p.plate)) {
			t.Errorf("dark %v: highlight %#06x, want the wash over the background, %#06x", c.dark, p.plate, got)
		}
		if got := blend(dibColor(p.rule.color), dibColor(p.back), p.rule.alpha); !near(got, dibColor(p.line)) {
			t.Errorf("dark %v: separator %#06x, want the rule over the background, %#06x", c.dark, p.line, got)
		}
		if p.border == dwmColorDefault || p.border == p.back {
			t.Errorf("dark %v: outline %#x, want a color of its own", c.dark, p.border)
		}
	}
}

// Items over the material wear WinUI's Acrylic tint over it, at the opacity
// of its luminosity layer: over any material, a menu is its fallback color
// but for the seventh of the material that shows through, which keeps it as
// light as the shell's flyouts over a dark desktop.
func TestMenuVeil(t *testing.T) {
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	for _, dark := range []bool{false, true} {
		p := fluentPalette(dark)
		img := newBitmap(screen, 8, 8)
		if img == nil {
			t.Fatal("no bitmap")
		}
		paintItem(img.dc, rect{0, 0, 8, 8}, itemFill{plate: p.wash.color, alpha: p.wash.alpha, glass: true, veil: p.veil}, nil, 0, func(*bitmap, rect) {})
		procGdiFlush.Call()
		px := img.px[0]
		img.free()
		alpha := float64(px>>24&0xFF) / 255
		want := float64(dibColor(p.back) & 0xFF) // gray, as the fallbacks are
		show := (1 - alpha) * 255                // as far as the material moves it
		if math.Abs(alpha-acrylicLuminosity) > 0.02 {
			t.Errorf("dark %v: an item is %v opaque, want the tint's %v", dark, alpha, acrylicLuminosity)
		}
		for _, bg := range []uint32{0x000000, 0xFFFFFF} {
			got := float64(px&0xFF) + (1-alpha)*float64(bg&0xFF)
			if math.Abs(got-want) > show+1 {
				t.Errorf("dark %v: an item over %#06x is %v, want %v but for %v", dark, bg, got, want, show)
			}
		}
	}
}

// The text of an item with the material behind it is drawn over the veil's
// own color, opaque, as the menus of Windows itself draw theirs: its pixels
// are the ones the same text has over the card, not thinned by grayscale
// antialiasing, and the item's background still shows the material through.
func TestMenuItemText(t *testing.T) {
	dpi := dpiOf(0)
	b := &Backend{themeSource: "light"}
	p := b.menuPalette()
	f := b.menuFace(dpi)
	const label = "按键设置 Open 打开"
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	w := textWidth(screen, f.text, label) + toPx(2*(menuInset+menuPadding), dpi)
	h := toPx(menuItemHeight, dpi)
	draw := func(img *bitmap, r rect) {
		old, _, _ := procSelectObject.Call(img.dc, f.text)
		defer procSelectObject.Call(img.dc, old)
		procSetBkMode.Call(img.dc, 1) // TRANSPARENT
		procSetTextColor.Call(img.dc, uintptr(p.text))
		at := rect{toPx(menuInset+menuPadding, dpi), 0, r.Right, h}
		procDrawTextW.Call(img.dc, uintptr(unsafe.Pointer(u16(label))), ^uintptr(0), uintptr(unsafe.Pointer(&at)), dtSingleLine|dtVCenter|dtNoClip)
	}
	// The text over the card, opaque, which the item's should be.
	card := dibColor(p.veil.color) | 0xFF000000
	plain := newBitmap(screen, w, h)
	if plain == nil {
		t.Fatal("no bitmap")
	}
	defer plain.free()
	for i := range plain.px {
		plain.px[i] = card
	}
	draw(plain, rect{0, 0, w, h})
	procGdiFlush.Call()

	item := newBitmap(screen, w, h)
	if item == nil {
		t.Fatal("no bitmap")
	}
	defer item.free()
	paintItem(item.dc, rect{0, 0, w, h}, itemFill{plate: p.wash.color, alpha: p.wash.alpha, glass: true, veil: p.veil}, nil, 0, draw)
	procGdiFlush.Call()

	ink, wrong := 0, 0
	for i, px := range item.px {
		if want := plain.px[i]; want&0xFFFFFF != card&0xFFFFFF {
			ink++
			if px>>24 != 0xFF || px&0xFFFFFF != want&0xFFFFFF {
				wrong++
			}
			continue
		}
		if px>>24 == 0xFF {
			wrong++
		}
	}
	if ink == 0 {
		t.Fatal("the label left no ink")
	}
	if wrong > 0 {
		t.Errorf("%d pixels of the item are not the text over the card, opaque, over the material", wrong)
	}
}

// The frame of a menu with the material behind it is the veil over it, as
// its items are: premultiplied pixels, the color of the card rather than
// the material's, for DWM's one outline around the window to be the menu's
// only border.
func TestMenuFrame(t *testing.T) {
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	img := newBitmap(screen, 4, 4)
	if img == nil {
		t.Fatal("no bitmap")
	}
	defer img.free()
	p := fluentPalette(false)
	fillTint(img.dc, &rect{0, 0, 4, 4}, p.veil)
	procGdiFlush.Call()
	px := img.px[5]
	// 0.85 of the fallback #F9F9F9, premultiplied.
	if got, want := px>>24, uint32(217); got != want {
		t.Errorf("the frame is %#x opaque, want %#x", got, want)
	}
	if got, want := px&0xFF, uint32(212); got != want {
		t.Errorf("the frame is %#x, want %#x", got, want)
	}
}

// Items over the material are the premultiplied pixels that show as they
// were drawn over black and over white: WinUI's translucent highlights
// as translucent, text as opaque as it covers, and nothing as nothing.
func TestUnblend(t *testing.T) {
	over := func(c uint32, a float64) (onBlack, onWhite uint32) {
		return blend(c, 0x000000, a), blend(c, 0xFFFFFF, a)
	}
	for _, c := range []struct {
		name  string
		color uint32
		alpha float64
		want  uint32
	}{
		{"nothing", 0x000000, 0, 0x00000000},
		{"opaque text", 0x1A1A1A, 1, 0xFF1A1A1A},
		{"a dark menu's highlight", 0xFFFFFF, 0x0F / 255.0, 0x0F0F0F0F},
		{"a light menu's highlight", 0x000000, 0x09 / 255.0, 0x09000000},
		{"half covered", 0xFFFFFF, 0.5, 0x80808080},
	} {
		k, w := over(c.color, c.alpha)
		if got := unblend(k, w); got != c.want {
			t.Errorf("%s: unblend(%#x, %#x) = %#x, want %#x", c.name, k, w, got, c.want)
		}
	}
}

// A label of a menu item drawn as wide as it measures shows all its ink,
// that past its advance included: Microsoft YaHei UI's, the menu font of
// Chinese Windows, reaches a pixel past it in 粘贴, 删除 and 全选 (#1028).
func TestMenuLabelInk(t *testing.T) {
	dpi := dpiOf(0)
	const face = "Microsoft YaHei UI"
	// In ClearType, as menus draw where it is on, whose ink reaches
	// further than grayscale's.
	const clearTypeQuality = 5
	lf := logFont{Height: -toPx(14, dpi), Weight: 400, Quality: clearTypeQuality}
	copy(lf.FaceName[:], utf16z(face))
	text, _, _ := procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	defer procDeleteObject.Call(text)
	if !fontFace(text, face) {
		t.Skip(face + " is not installed")
	}
	glyphs := createFont(-toPx(menuGlyphSize, dpi), "Segoe UI")
	defer procDeleteObject.Call(glyphs)
	b := &Backend{themeSource: "light", menuFonts: map[int]menuFonts{dpi: {text: text, glyphs: glyphs}}}
	p := b.menuPalette()
	back := dibColor(p.back)
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	for _, label := range []string{"撤销", "剪切", "复制", "粘贴", "删除", "全选", "查看歌手：房东的猫"} {
		// Its ink drawn with room around it.
		const pad = 50
		w := textWidth(screen, text, label) + 2*pad
		h := toPx(menuItemHeight, dpi)
		img := newBitmap(screen, w, h)
		for i := range img.px {
			img.px[i] = back
		}
		old, _, _ := procSelectObject.Call(img.dc, text)
		procSetBkMode.Call(img.dc, 1) // TRANSPARENT
		procSetTextColor.Call(img.dc, uintptr(p.text))
		at := rect{pad, 0, w, h}
		procDrawTextW.Call(img.dc, uintptr(unsafe.Pointer(u16(label))), ^uintptr(0), uintptr(unsafe.Pointer(&at)), dtSingleLine|dtVCenter|dtNoClip)
		procGdiFlush.Call()
		procSelectObject.Call(img.dc, old)
		want := inkLast(img, w, back)
		want -= pad
		img.free()

		// The item, as wide as it needs.
		l := &menuLook{label: label}
		w = b.itemWidth(l, dpi)
		img = newBitmap(screen, w, h)
		b.drawItem(0, l, &drawItemStruct{CtlType: odtMenu, DC: img.dc, Item: rect{0, 0, w, h}})
		procGdiFlush.Call()
		got := inkLast(img, w, back)
		got -= toPx(menuInset+menuPadding, dpi)
		img.free()
		if want < 0 || got != want {
			t.Errorf("%q: the item shows its ink up to column %d of the label, want %d", label, got, want)
		}
	}
}

// inkLast returns the last column of a bitmap w pixels wide holding
// pixels other than back, -1 for none.
func inkLast(img *bitmap, w int32, back uint32) int32 {
	last := int32(-1)
	for i, p := range img.px {
		if p&0xFFFFFF != back {
			last = max(last, int32(i)%w)
		}
	}
	return last
}

// The shortcuts of items are WinUI's Caption, in its own font: the menu
// font's Latin letters are larger at the same size — Microsoft YaHei UI's,
// the menu font of Chinese Windows, by a sixth of their ink height, which
// made the shell's shortcuts look bigger than its own.
func TestMenuAccelFace(t *testing.T) {
	dpi := dpiOf(0)
	b := &Backend{themeSource: "light"}
	f := b.menuFace(dpi)
	if !fontFace(f.accel, "Segoe UI Variable Small") {
		t.Skip("Segoe UI Variable Small is not installed")
	}
	var ncm nonClientMetrics
	ncm.Size = uint32(unsafe.Sizeof(ncm))
	procSystemParametersInfoForDpi.Call(spiGetNonClientMetrics, uintptr(ncm.Size), uintptr(unsafe.Pointer(&ncm)), 0, uintptr(dpi))
	lf := ncm.MenuFont
	lf.Height = int32(math.Round(float64(lf.Height) * menuAccelSize / 12))
	menuFont, _, _ := procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	defer procDeleteObject.Call(menuFont)
	if fontFace(menuFont, "Segoe UI") {
		t.Skip("the menu font is Segoe UI, whose Latin letters aren't larger")
	}
	const shortcut = "Ctrl+B"
	accelW, accelH := inkBox(t, f.accel, shortcut, dpi)
	menuW, menuH := inkBox(t, menuFont, shortcut, dpi)
	if accelH >= menuH || accelW >= menuW {
		t.Errorf("a shortcut's ink is %dx%d, want smaller than the menu font's %dx%d", accelW, accelH, menuW, menuH)
	}
}

// inkBox returns the width and the height of the ink of text in a font,
// which is what makes it look as big as it does.
func inkBox(t *testing.T, font uintptr, text string, dpi int) (width, height int32) {
	t.Helper()
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	w, h := textWidth(screen, font, text)+40, toPx(menuItemHeight, dpi)
	img := newBitmap(screen, w, h)
	if img == nil {
		t.Fatal("no bitmap")
	}
	defer img.free()
	for i := range img.px {
		img.px[i] = 0x00FFFFFF
	}
	old, _, _ := procSelectObject.Call(img.dc, font)
	procSetBkMode.Call(img.dc, 1) // TRANSPARENT
	procSetTextColor.Call(img.dc, 0x000000)
	at := rect{20, 0, w, h}
	procDrawTextW.Call(img.dc, uintptr(unsafe.Pointer(u16(text))), ^uintptr(0), uintptr(unsafe.Pointer(&at)), dtSingleLine|dtVCenter|dtNoClip)
	procGdiFlush.Call()
	procSelectObject.Call(img.dc, old)
	top, bottom, left, right := int32(1<<30), int32(-1), int32(1<<30), int32(-1)
	for y := int32(0); y < h; y++ {
		for x := int32(0); x < w; x++ {
			if img.px[y*w+x]&0xFFFFFF != 0xFFFFFF {
				top, bottom = min(top, y), max(bottom, y)
				left, right = min(left, x), max(right, x)
			}
		}
	}
	return right - left + 1, bottom - top + 1
}
