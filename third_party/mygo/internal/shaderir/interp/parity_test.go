package interp_test

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/shaderir"
	"github.com/egoist/mygo/internal/shaderir/interp"
)

// noiseSource is a shader of what textures computed in shaders are made
// of, for the GPU renderers to draw as the interpreter does: hashes of
// unsigned integers, value noise from them, loops over uniform arrays to
// a count that is not a constant, swizzles, a matrix, and the usual
// functions.
const noiseSource = `package main

var Seed uint
var Time float
var Count int
var Taps [4]vec4
var Turn mat2

func pcg(v uint) uint {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

func unit(h uint) float {
	return float(h>>8) * (1.0 / 16777216.0)
}

func value(p vec2) float {
	i := floor(p)
	f := p - i
	w := f * f * (3 - 2*f)
	x0, y0 := uint(int(i.x)), uint(int(i.y))
	a := unit(pcg(x0 ^ pcg(y0^Seed)))
	b := unit(pcg((x0 + 1) ^ pcg(y0^Seed)))
	c := unit(pcg(x0 ^ pcg((y0+1)^Seed)))
	d := unit(pcg((x0 + 1) ^ pcg((y0+1)^Seed)))
	return mix(mix(a, b, w.x), mix(c, d, w.x), w.y)
}

func Fragment(dst vec4, pos vec2) vec4 {
	q := Turn * pos
	n := value(q/9 + Time) * 0.6 + value(q/4) * 0.4
	glow := 0.0
	for i := 0; i < Count; i++ {
		d := distance(pos, Taps[i].xy)
		glow += Taps[i].z * exp(-d*d/(Taps[i].w*Taps[i].w))
	}
	// A uint loop by twos, a uvec of one scalar, a range over a count the
	// body changes (read once, as Go reads it), and smoothstep between
	// equal edges (0, not NaN).
	k := 0.0
	for j := uint(0); j < Seed%7+uint(int(pos.x/40)); j += 2 {
		k += 0.02
	}
	m := Count + int(pos.y/30)
	for i := range m {
		m--
		k += 0.01 * float(i+1)
	}
	u := uvec3(Seed + uint(int(pos.x)))
	k += float(u.y%5)*0.01 + smoothstep(Time, Time, Time)
	// A uint's complement, an int shifted by a uint, a range over a uint
	// indexing an array, a shift of an untyped constant in a uint, and a
	// uint loop past int's range.
	var bits [4]uint
	for i := range Seed%3 + 1 {
		bits[i] = ^uint(0) >> (i * 8)
	}
	s := int(pos.x)
	s >>= Seed % 3
	sel := bits[uint(int(pos.y))%4] & (1 << (s % 31))
	k += float(sel>>(s%31)) * 0.03
	for j := uint(4294967290); j < 4294967293; j++ {
		k += float(j%3) * 0.01
	}
	c := vec3(n, smoothstep(0.2, 0.8, n), clamp(glow, 0, 1))
	c.rb = c.br * vec2(1, 0.5+0.5*sin(pos.x/7))
	c.g = fract(c.g + k)
	return vec4(c*0.9, 0.9)
}
`

// noiseUniforms returns the values of noiseSource's uniforms.
func noiseUniforms(p *shaderir.Program) []uint32 {
	return uniforms(p, map[string][]uint32{
		"Seed":  {0x3fa1c2d9},
		"Time":  f(1.75),
		"Count": {3},
		"Taps":  f(40, 30, 1, 20, 150, 60, 0.7, 35, 220, 100, 0.4, 50, 0, 0, 9, 1),
		"Turn":  f(0.8, 0.6, -0.6, 0.8),
	})
}

// effectScene returns a scene of fx over a white frame.
func effectScene(fx *scene.Effect) *scene.Scene {
	s := &scene.Scene{Width: 240, Height: 120, Clear: scene.Color{R: 255, G: 255, B: 255, A: 255}}
	s.Ops = append(s.Ops, scene.Op{Kind: scene.OpEffect, Rect: scene.Rect{X: 10, Y: 6, W: 220, H: 108}})
	s.Effects = append(s.Effects, scene.EffectOp{Effect: fx})
	return s
}

// shaderPixels draws the effect with the interpreter.
type shaderPixels struct {
	prog *interp.Program
	in   *interp.Inputs
	rect scene.Rect
}

func (s *shaderPixels) Begin(op *scene.EffectOp, rect scene.Rect, radii [4]float32) { s.rect = rect }

func (s *shaderPixels) Color(x, y float32, _ *scene.BackdropImage) [4]float32 {
	c, _ := s.prog.Color(s.in, [4]float32{x, y, 0, 1}, x-s.rect.X, y-s.rect.Y)
	return c
}

// constUniforms replaces the uniform variables U0, U1, ... that the
// declarations of code (a line each, matched by decl, with the variable
// as its group) declare, with constants of their values: GLSL and HLSL
// read them as globals.
func constUniforms(code string, decl *regexp.Regexp, p *shaderir.Program, words []uint32, hlsl bool) string {
	vals := map[string]string{}
	k := 0
	for i := range p.Uniforms {
		t := &p.Uniforms[i]
		n := interp.UniformWords(p)[i]
		vals[fmt.Sprintf("U%d", i)] = literal(t, words[k:k+n], hlsl)
		k += n
	}
	return decl.ReplaceAllStringFunc(code, func(line string) string {
		m := decl.FindStringSubmatch(line)
		v, ok := vals[m[2]]
		if !ok {
			return line
		}
		if hlsl {
			return fmt.Sprintf("static const %s = %s;", m[1], v)
		}
		return fmt.Sprintf("const %s = %s;", m[1], v)
	})
}

// literal returns the value of type t in words in GLSL or HLSL.
func literal(t *shaderir.Type, words []uint32, hlsl bool) string {
	num := func(kind shaderir.BasicType, w uint32) string {
		switch kind {
		case shaderir.Float:
			s := fmt.Sprintf("%.9g", math.Float32frombits(w))
			if !strings.ContainsAny(s, ".e") {
				s += ".0"
			}
			return s
		case shaderir.Uint:
			return fmt.Sprintf("%du", w)
		case shaderir.Bool:
			return fmt.Sprint(w != 0)
		}
		return fmt.Sprint(int32(w))
	}
	if t.Main == shaderir.Array {
		n := len(words) / t.Length
		var es []string
		for i := range t.Length {
			es = append(es, literal(&t.Sub[0], words[i*n:(i+1)*n], hlsl))
		}
		if hlsl {
			return "{" + strings.Join(es, ", ") + "}"
		}
		return fmt.Sprintf("%s[%d](%s)", typeName(&t.Sub[0], hlsl), t.Length, strings.Join(es, ", "))
	}
	var es []string
	for _, w := range words {
		es = append(es, num(t.Scalar(), w))
	}
	if len(es) == 1 {
		return es[0]
	}
	// A matrix's lanes are its columns, which HLSL's constructor takes as
	// rows of the transposed matrix the HLSL backend keeps.
	return typeName(t, hlsl) + "(" + strings.Join(es, ", ") + ")"
}

func typeName(t *shaderir.Type, hlsl bool) string {
	if !hlsl {
		return t.String()
	}
	switch t.Main {
	case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4:
		return fmt.Sprintf("float%d", t.VectorElementCount())
	case shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
		return fmt.Sprintf("int%d", t.VectorElementCount())
	case shaderir.UVec2, shaderir.UVec3, shaderir.UVec4:
		return fmt.Sprintf("uint%d", t.VectorElementCount())
	case shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
		n := t.MatrixSize()
		return fmt.Sprintf("float%dx%d", n, n)
	}
	return t.String()
}
