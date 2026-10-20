package scene

import (
	"math"
	"testing"
)

func TestAffineCompositionInverseAndSingular(t *testing.T) {
	m := Translation(30, 20).Mul(Rotation(90)).Mul(Scaling(2, -3))
	x, y := m.Point(4, 5)
	if math.Abs(float64(x-45)) > 1e-4 || math.Abs(float64(y-28)) > 1e-4 {
		t.Fatalf("composed point %v,%v", x, y)
	}
	i, ok := m.Inverse()
	if !ok {
		t.Fatal("invertible matrix rejected")
	}
	x, y = i.Point(x, y)
	if math.Abs(float64(x-4)) > 1e-4 || math.Abs(float64(y-5)) > 1e-4 {
		t.Fatalf("inverse point %v,%v", x, y)
	}
	if _, ok := Scaling(0, 1).Inverse(); ok {
		t.Fatal("singular matrix inverted")
	}
	b := m.Bounds(Rect{W: 4, H: 5})
	if math.Abs(float64(b.W-15)) > 1e-4 || math.Abs(float64(b.H-8)) > 1e-4 {
		t.Fatalf("bounds %v", b)
	}
	if (Affine{}).Mul(m) != m || m.Mul(Affine{}) != m {
		t.Fatal("zero value is not identity")
	}
}
