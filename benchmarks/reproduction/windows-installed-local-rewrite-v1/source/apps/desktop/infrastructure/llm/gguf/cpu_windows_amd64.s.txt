#include "textflag.h"

TEXT ·cpuid(SB), NOSPLIT, $0-24
	MOVL leaf+0(FP), AX
	MOVL subleaf+4(FP), CX
	CPUID
	MOVL AX, a+8(FP)
	MOVL BX, b+12(FP)
	MOVL CX, c+16(FP)
	MOVL DX, d+20(FP)
	RET

TEXT ·xgetbv(SB), NOSPLIT, $0-8
	XORL CX, CX
	XGETBV
	SHLQ $32, DX
	ORQ DX, AX
	MOVQ AX, ret+0(FP)
	RET
