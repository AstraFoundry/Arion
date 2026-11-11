package scene

import "testing"

func TestResizeReusesMemoryUnlessMuchSmaller(t *testing.T) {
	m := NewImageRGBA(100, 100, make([]byte, 4*100*100))
	big := &m.Pix[0]
	// Smaller, down to a quarter of the memory: the same memory.
	m.Resize(60, 50)
	if m.W != 60 || m.H != 50 || len(m.Pix) != 4*60*50 || &m.Pix[0] != big {
		t.Fatalf("resized to %dx%d with %d of %d bytes, want 60x50 in the same memory", m.W, m.H, len(m.Pix), cap(m.Pix))
	}
	m.Resize(100, 100)
	if len(m.Pix) != 4*100*100 || &m.Pix[0] != big {
		t.Fatal("growing back within the memory allocated anew")
	}
	// Under a quarter of it: memory of its own size (#1407).
	m.Resize(40, 40)
	if len(m.Pix) != 4*40*40 || cap(m.Pix) != 4*40*40 {
		t.Fatalf("shrank to %d bytes keeping %d", len(m.Pix), cap(m.Pix))
	}
	m.Resize(0, 0)
	if cap(m.Pix) != 0 {
		t.Fatalf("an empty image keeps %d bytes", cap(m.Pix))
	}
	// A canvas no renderer drew in memory still has no pixels.
	c := &Image{Scene: &Scene{}}
	c.Resize(10, 10)
	if c.Pix != nil || c.W != 10 || c.H != 10 {
		t.Fatal("resizing a canvas made its pixels")
	}
}
