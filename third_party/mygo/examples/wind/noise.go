package main

import "math"

// The texture's randomness on the CPU: PCG hashes of whole numbers and
// value noise on them, as wind.kage computes them, for how far the wind
// has carried its gusts, and the hash of the visit's seed that picks its
// wind. The shaders compute the rest.

// pcg hashes v, as PCG's RXS-M-XS output of one step from v.
func pcg(v uint32) uint32 {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

// unit returns h as a fraction from 0 to 1, from its top 24 bits.
func unit(h uint32) float32 { return float32(h>>8) * (1.0 / 16777216.0) }

// quintic is the fade of the noises between lattice points, whose first
// and second derivatives are 0 at both ends.
func quintic(f float32) float32 { return f * f * f * (f*(f*6-15) + 10) }

func mix(a, b, t float32) float32 { return a + (b-a)*t }

func floor(v float32) float32 { return float32(math.Floor(float64(v))) }

// lattice is a point of the noises' lattice: the hashes of its four rows
// in its two layers, from which a corner's hash is its column's.
type lattice struct {
	x0, x1     uint32
	a, b, c, d uint32
	f          [3]float32
	w          [3]float32
}

func (n *noises) lattice(x, y, z float32, salt uint32) lattice {
	fx, fy, fz := floor(x), floor(y), floor(z)
	ix, iy, iz := int32(fx), int32(fy), int32(fz)
	z0 := pcg(uint32(iz) ^ n.seed ^ salt)
	z1 := pcg(uint32(iz+1) ^ n.seed ^ salt)
	y0, y1 := uint32(iy), uint32(iy+1)
	l := lattice{x0: uint32(ix), x1: uint32(ix + 1), a: pcg(y0 ^ z0), b: pcg(y1 ^ z0), c: pcg(y0 ^ z1), d: pcg(y1 ^ z1)}
	l.f = [3]float32{x - fx, y - fy, z - fz}
	l.w = [3]float32{quintic(l.f[0]), quintic(l.f[1]), quintic(l.f[2])}
	return l
}

// noises are the noises of a visit, hashed with its seed.
type noises struct{ seed uint32 }

// value is value noise from -1 to 1: a random value at each lattice
// point, blended between them.
func (n *noises) value(x, y, z float32, salt uint32) float32 {
	l := n.lattice(x, y, z, salt)
	v := func(h uint32) float32 { return unit(pcg(h))*2 - 1 }
	return mix(
		mix(mix(v(l.x0^l.a), v(l.x1^l.a), l.w[0]), mix(v(l.x0^l.b), v(l.x1^l.b), l.w[0]), l.w[1]),
		mix(mix(v(l.x0^l.c), v(l.x1^l.c), l.w[0]), mix(v(l.x0^l.d), v(l.x1^l.d), l.w[0]), l.w[1]),
		l.w[2])
}

// mix32 scrambles the bits of v, as the app's integer hash.
func mix32(v uint32) uint32 {
	v = (v ^ v>>16) * 569420461
	v = (v ^ v>>15) * 1935289751
	return v ^ v>>15
}

// random returns numbers from 0 to 1, a sequence of its own for each seed
// and salt.
func random(seed uint32, salt string) func() float64 {
	n := mix32(seed)
	for i := 0; i < len(salt); i++ {
		n = mix32(n ^ uint32(salt[i]))
	}
	return func() float64 {
		n += 0x9E3779B9
		return float64(mix32(n)) / 4294967296
	}
}

func smoothstep(e0, e1, x float32) float32 {
	t := min(max((x-e0)/(e1-e0), 0), 1)
	return t * t * (3 - 2*t)
}

func exp(v float32) float32 { return float32(math.Exp(float64(v))) }
func cos(v float32) float32 { return float32(math.Cos(float64(v))) }
func abs(v float32) float32 { return float32(math.Abs(float64(v))) }
func hypot(x, y float32) float32 {
	return float32(math.Hypot(float64(x), float64(y)))
}
