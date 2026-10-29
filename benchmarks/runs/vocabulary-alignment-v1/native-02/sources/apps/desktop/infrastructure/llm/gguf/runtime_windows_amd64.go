package gguf

// cpuid reports processor instruction support; xgetbv additionally verifies
// that the operating system preserves both the XMM and YMM register state.
func cpuid(leaf, subleaf uint32) (a, b, c, d uint32)
func xgetbv() uint64

func runtimeExecutable() string {
	maximum, _, _, _ := cpuid(0, 0)
	if maximum < 7 {
		return "llama-server.exe"
	}
	_, _, c, _ := cpuid(1, 0)
	_, b, _, _ := cpuid(7, 0)
	if acceleratedCPU(c, b) && xgetbv()&6 == 6 {
		return "llama-server-avx2.exe"
	}
	return "llama-server.exe"
}

func acceleratedCPU(leaf1c, leaf7b uint32) bool {
	// SSE4.2, FMA, F16C, XSAVE, OSXSAVE, AVX; AVX2 and BMI2.
	const leaf1 = 1<<20 | 1<<12 | 1<<29 | 1<<26 | 1<<27 | 1<<28
	const leaf7 = 1<<5 | 1<<8
	return leaf1c&leaf1 == leaf1 && leaf7b&leaf7 == leaf7
}
