// Command arion-icon renders resources/icon.png, Arion's app icon: five
// rounded ink voice bars on a Sunbeam yellow squircle. Rendering is deterministic so
// the committed PNG can be regenerated and diffed.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strconv"
)

const (
	size    = 1024
	samples = 4
	margin  = 100.0
)

// squircle reports coverage of a superellipse (n=5), Apple's icon shape.
func squircle(x, y float64) bool {
	half := (size - 2*margin) / 2
	cx, cy := size/2.0, size/2.0+6
	nx, ny := math.Abs(x-cx)/half, math.Abs(y-cy)/half
	return math.Pow(nx, 5)+math.Pow(ny, 5) <= 1
}

func bar(x, y float64) bool {
	heights := []float64{230, 420, 560, 360, 250}
	width, gap := 64.0, 46.0
	total := float64(len(heights))*width + float64(len(heights)-1)*gap
	left := size/2.0 - total/2
	for i, h := range heights {
		x0 := left + float64(i)*(width+gap)
		top, bottom := size/2.0-h/2, size/2.0+h/2
		r := width / 2
		if x < x0 || x > x0+width || y < top || y > bottom {
			continue
		}
		cx := x0 + r
		switch {
		case y < top+r:
			if math.Hypot(x-cx, y-(top+r)) <= r {
				return true
			}
		case y > bottom-r:
			if math.Hypot(x-cx, y-(bottom-r)) <= r {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// menuIcon renders the input-menu glyph at n pixels: the same five bars in
// black on transparent, which macOS tints as a template. The glyph is drawn
// on a 64-unit grid and supersampled so 16 and 32 pixel slices stay crisp.
func menuIcon(path string, n int) {
	const grid, samples = 64.0, 4
	scale := grid / float64(n)
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	heights := []float64{18, 34, 48, 30, 20}
	inside := func(x, y float64) bool {
		for i, h := range heights {
			x0 := 7 + float64(i)*11
			top, bottom := grid/2-h/2, grid/2+h/2
			if x >= x0 && x <= x0+6 && y >= top+3 && y <= bottom-3 || math.Hypot(x-(x0+3), y-(top+3)) <= 3 || math.Hypot(x-(x0+3), y-(bottom-3)) <= 3 {
				return true
			}
		}
		return false
	}
	for py := 0; py < n; py++ {
		for px := 0; px < n; px++ {
			hits := 0
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/samples) * scale
					y := (float64(py) + (float64(sy)+0.5)/samples) * scale
					if inside(x, y) {
						hits++
					}
				}
			}
			img.Set(px, py, color.NRGBA{A: uint8(255 * hits / (samples * samples))})
		}
	}
	write(path, img)
}

func write(path string, img image.Image) {
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) == 4 && os.Args[1] == "-menu" {
		pixels, err := strconv.Atoi(os.Args[2])
		if err != nil || pixels <= 0 {
			log.Fatalf("invalid menu icon size %q", os.Args[2])
		}
		menuIcon(os.Args[3], pixels)
		return
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var a, r, g, b float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					x := float64(px) + (float64(sx)+0.5)/samples
					y := float64(py) + (float64(sy)+0.5)/samples
					if !squircle(x, y) {
						continue
					}
					a++
					if bar(x, y) {
						// Ink 900, the Sunbeam text color on yellow.
						r, g, b = r+21, g+18, b+15
						continue
					}
					// Sunbeam yellow (#FFCC00), slightly lighter at the top.
					t := (y - margin) / (size - 2*margin)
					r += 255 - 4*t
					g += 214 - 18*t
					b += 46 - 46*t
				}
			}
			if a == 0 {
				continue
			}
			img.Set(px, py, color.NRGBA{R: uint8(r / a), G: uint8(g / a), B: uint8(b / a), A: uint8(255 * a / (samples * samples))})
		}
	}
	write(os.Args[1], img)
}
