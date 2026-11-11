package interp_test

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/shader"
	"github.com/egoist/mygo/internal/shaderir"
	"github.com/egoist/mygo/internal/shaderir/interp"
)

func compile(t *testing.T, src string) (*shaderir.Program, *interp.Program) {
	t.Helper()
	p, err := shader.CompileFragment([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	prog, err := interp.New(p)
	if err != nil {
		t.Fatal(err)
	}
	return p, prog
}

// uniforms packs values by name as the program's uniform words.
func uniforms(p *shaderir.Program, values map[string][]uint32) []uint32 {
	var out []uint32
	for i, w := range interp.UniformWords(p) {
		v := values[p.UniformNames[i]]
		for j := range w {
			if j < len(v) {
				out = append(out, v[j])
			} else {
				out = append(out, 0)
			}
		}
	}
	return out
}

func f(v ...float32) []uint32 {
	out := make([]uint32, len(v))
	for i := range v {
		out[i] = math.Float32bits(v[i])
	}
	return out
}

func pcg(v uint32) uint32 {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

const hashSource = `package main

var Seed uint

func pcg(v uint) uint {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

func unit(h uint) float {
	return float(h>>8) * (1.0 / 16777216.0)
}

func Fragment(dst vec4, pos vec2) vec4 {
	c := uvec2(dst.xy)
	h := pcg(c.x*0x9E3779B9 ^ pcg(c.y^Seed))
	return vec4(unit(h), unit(pcg(h)), float(h%7)/7, 1)
}
`

// Unsigned arithmetic wraps around and shifts in zeros, as Go's uint32.
func TestHash(t *testing.T) {
	p, prog := compile(t, hashSource)
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{"Seed": {0x3fa1c2d9}})}
	unit := func(h uint32) float32 { return float32(h>>8) * (1.0 / 16777216.0) }
	for y := range 16 {
		for x := range 16 {
			got, ok := prog.Color(in, [4]float32{float32(x) + 0.5, float32(y) + 0.5, 0, 1}, float32(x)+0.5, float32(y)+0.5)
			h := pcg(uint32(x)*0x9E3779B9 ^ pcg(uint32(y)^0x3fa1c2d9))
			want := [4]float32{unit(h), unit(pcg(h)), float32(h%7) / 7, 1}
			if !ok || got != want {
				t.Fatalf("(%d, %d): %v, want %v", x, y, got, want)
			}
		}
	}
}

// Loops run to bounds that are not constants, functions return several
// results through output parameters, and assignments reach elements and
// swizzles.
func TestStatements(t *testing.T) {
	p, prog := compile(t, `package main

var Count int
var Weights [4]float

func split(x float) (float, float) {
	return floor(x), fract(x)
}

func Fragment(dst vec4, pos vec2) vec4 {
	sum := 0.0
	for i := 0; i < Count; i++ {
		if i == 2 {
			continue
		}
		sum += Weights[i]
	}
	var a [3]float
	for i := range 3 {
		a[i] = float(i) * 2
	}
	a[Count-3] = 10
	whole, part := split(pos.x)
	v := vec4(0)
	v.zx = vec2(whole, part)
	v.y = a[0] + a[1] + a[2]
	v.w = sum
	return v
}
`)
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{
		"Count":   {4},
		"Weights": f(1, 2, 4, 8),
	})}
	got, _ := prog.Color(in, [4]float32{}, 2.25, 0)
	// a = {0, 10, 4}; Weights without the third: 1 + 2 + 8.
	if want := [4]float32{0.25, 14, 2, 11}; got != want {
		t.Errorf("%v, want %v", got, want)
	}
}

// Matrices are products of columns, and the built-in functions compute
// what GLSL's do.
func TestBuiltins(t *testing.T) {
	p, prog := compile(t, `package main

var M mat2
var X float

func Fragment(dst vec4, pos vec2) vec4 {
	v := M * vec2(1, 2)
	w := vec2(1, 2) * M
	s := smoothstep(0, 4, X) + mix(1.0, 3.0, 0.25) + clamp(X, 0, 1) + step(2, X)
	return vec4(v.x+v.y*10, w.x+w.y*10, s, mod(-X, 1.5) + max(1, 2, X) + min(vec2(X), 0).x + abs(-2.0))
}
`)
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{
		"M": f(1, 2, 3, 4), // columns (1, 2) and (3, 4)
		"X": f(3),
	})}
	got, _ := prog.Color(in, [4]float32{}, 0, 0)
	// M*(1,2) = (7, 10); (1,2)*M = (5, 11).
	sm := float32(0.75 * 0.75 * (3 - 1.5))
	want := [4]float32{7 + 100, 5 + 110, sm + 1.5 + 1 + 1, 0 + 3 + 0 + 2}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > 1e-5 {
			t.Errorf("component %d: %v, want %v", i, got[i], want[i])
		}
	}
}

type testImage struct {
	w, h int
	pix  [][4]float32
}

func (m *testImage) Size() (int, int)       { return m.w, m.h }
func (m *testImage) At(x, y int) [4]float32 { return m.pix[y*m.w+x] }

// Images are read pixel by pixel, transparent outside, and sampled
// between pixels' centers, clamped to their edges.
func TestImages(t *testing.T) {
	p, prog := compile(t, `package main

var At vec2

func Fragment(dst vec4, pos vec2) vec4 {
	return vec4(image0At(At).r, image0Sample(At).r, image1At(vec2(0)).g, image0Size().x)
}
`)
	img := &testImage{w: 2, h: 2, pix: [][4]float32{{0, 0, 0, 1}, {1, 0, 0, 1}, {0.5, 0, 0, 1}, {0.25, 0, 0, 1}}}
	for _, c := range []struct {
		at             [2]float32
		pixel, sampled float32
	}{
		{[2]float32{0.5, 0.5}, 0, 0},
		{[2]float32{1, 0.5}, 1, 0.5},
		{[2]float32{1, 1}, 0.25, 0.4375},
		{[2]float32{-1, 0.5}, 0, 0},
		{[2]float32{5, 5}, 0, 0.25},
	} {
		in := &interp.Inputs{
			Uniforms: uniforms(p, map[string][]uint32{"At": f(c.at[0], c.at[1]), "__imageSizes": f(2, 2)}),
			Images:   []interp.Image{img},
		}
		got, _ := prog.Color(in, [4]float32{}, 0, 0)
		if got[0] != c.pixel || math.Abs(float64(got[1]-c.sampled)) > 1e-6 || got[2] != 0 || got[3] != 2 {
			t.Errorf("at %v: %v, want pixel %v, sampled %v, no image 1, width 2", c.at, got, c.pixel, c.sampled)
		}
	}
}

// A discarded pixel has no color.
func TestDiscard(t *testing.T) {
	p, prog := compile(t, `package main

func Fragment(dst vec4, pos vec2) vec4 {
	if pos.x < 1 {
		discard()
	}
	return vec4(1)
}
`)
	in := &interp.Inputs{Uniforms: uniforms(p, nil)}
	if _, ok := prog.Color(in, [4]float32{}, 0.5, 0); ok {
		t.Error("a pixel the shader discards has a color")
	}
	if c, ok := prog.Color(in, [4]float32{}, 1.5, 0); !ok || c != [4]float32{1, 1, 1, 1} {
		t.Errorf("%v, %v, want white", c, ok)
	}
}

// A local's index names the variable of the innermost block declaring
// it: a block after a loop reuses the indices of the loop's locals, with
// types of its own.
func TestScopes(t *testing.T) {
	p, prog := compile(t, `package main

var Count int
var Taps [4]vec4

func Fragment(dst vec4, pos vec2) vec4 {
	glow := 0.0
	for i := 0; i < Count; i++ {
		d := distance(pos, Taps[i].xy)
		glow += Taps[i].z * exp(-d*d/(Taps[i].w*Taps[i].w))
	}
	c := vec3(1, 2, glow)
	c.rb = c.br
	return vec4(c, glow)
}
`)
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{
		"Count": {2},
		"Taps":  f(0, 0, 1, 10, 30, 0, 0.5, 20, 0, 0, 0, 0, 0, 0, 0, 0),
	})}
	got, _ := prog.Color(in, [4]float32{}, 30, 0)
	glow := float32(math.Exp(-9)) + 0.5
	if want := [4]float32{glow, 2, 1, glow}; math.Abs(float64(got[0]-want[0])) > 1e-6 || got[1] != 2 || got[2] != 1 {
		t.Errorf("%v, want %v", got, want)
	}
}

func BenchmarkNoise(b *testing.B) {
	p, err := shader.CompileFragment([]byte(noiseSource))
	if err != nil {
		b.Fatal(err)
	}
	prog, err := interp.New(p)
	if err != nil {
		b.Fatal(err)
	}
	in := &interp.Inputs{Uniforms: noiseUniforms(p)}
	b.ReportAllocs()
	for i := range b.N {
		x := float32(i % 256)
		prog.Color(in, [4]float32{x, 10, 0, 1}, x, 10)
	}
}

// A range reads its count once, as Go does, though the body changes it;
// smoothstep between equal edges is 0, as GPUs clamp its NaN.
func TestRangeAndSmoothstep(t *testing.T) {
	p, prog := compile(t, `package main

var N int
var X float

func Fragment(dst vec4, pos vec2) vec4 {
	n := N
	sum := 0.0
	for i := range n {
		n--
		sum += 1
		_ = i
	}
	return vec4(sum, float(n), smoothstep(X, X, X), 0)
}
`)
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{"N": {5}, "X": f(0.5)})}
	got, _ := prog.Color(in, [4]float32{}, 0, 0)
	if want := [4]float32{5, 0, 0, 0}; got != want {
		t.Errorf("%v, want %v", got, want)
	}
}

// A uint's complement is in its 32 bits, ints shift by uints, ranges run
// over uints and index arrays with them, uint loops run past int's
// range, and a shift of an untyped constant takes the type of its
// context, as in Go.
func TestUintForms(t *testing.T) {
	p, prog := compile(t, `package main

var Seed uint
var Shift int

const N uint = 7

func top(m int) uint {
	return 1 << m
}

func Fragment(dst vec4, pos vec2) vec4 {
	var bits [4]uint
	for i := range Seed%3 + 1 {
		bits[i] = ^uint(0) >> (i * 8)
	}
	s := int(pos.x)
	s >>= Seed % 3
	sel := bits[uint(int(pos.y))%4] & (1 << (s % 31))
	sum := uint(0)
	for j := uint(4294967290); j < 4294967295; j++ {
		sum += j & N
	}
	var shifted uint = 1 << Shift
	return vec4(float(sel>>8), float(shifted|^(1<<Shift)&N), float(sum), float(top(Shift)|N&(1<<Shift)))
}
`)
	const seed, shift = 0x3fa1c2d9, 2
	in := &interp.Inputs{Uniforms: uniforms(p, map[string][]uint32{"Seed": {seed}, "Shift": {shift}})}
	for _, pos := range [][2]float32{{60, 0}, {80, 1}, {40, 2}, {60, 3}, {100, 1}} {
		var bits [4]uint32
		for i := range uint32(seed%3 + 1) {
			bits[i] = ^uint32(0) >> (i * 8)
		}
		s := int32(pos[0])
		s >>= uint32(seed % 3)
		sel := bits[uint32(int32(pos[1]))%4] & (1 << (s % 31))
		sum := uint32(0)
		for j := uint32(4294967290); j < 4294967295; j++ {
			sum += j & 7
		}
		var shifted uint32 = 1 << shift
		want := [4]float32{float32(sel >> 8), float32(shifted | ^uint32(1<<shift)&7), float32(sum), float32(uint32(1<<shift) | 7&(1<<shift))}
		got, _ := prog.Color(in, [4]float32{}, pos[0], pos[1])
		if got != want {
			t.Errorf("%v: %v, want %v", pos, got, want)
		}
	}
}
