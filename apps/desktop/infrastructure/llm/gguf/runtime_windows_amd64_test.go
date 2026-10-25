package gguf

import "testing"

func TestAcceleratedRuntimeRequiresEveryInstructionAndOSXSAVE(t *testing.T) {
	t.Logf("Host runtime selection: %s", runtimeExecutable())
	const leaf1 = 1<<20 | 1<<12 | 1<<29 | 1<<26 | 1<<27 | 1<<28
	const leaf7 = 1<<5 | 1<<8
	if !acceleratedCPU(leaf1, leaf7) {
		t.Fatal("complete processor features rejected")
	}
	for bit := uint(0); bit < 32; bit++ {
		if leaf1&(1<<bit) != 0 && acceleratedCPU(leaf1 & ^uint32(1<<bit), leaf7) {
			t.Fatalf("missing leaf 1 feature %d accepted", bit)
		}
		if leaf7&(1<<bit) != 0 && acceleratedCPU(leaf1, leaf7 & ^uint32(1<<bit)) {
			t.Fatalf("missing leaf 7 feature %d accepted", bit)
		}
	}
}
