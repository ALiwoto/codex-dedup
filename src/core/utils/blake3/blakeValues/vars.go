package blakeValues

import (
	"os"

	"golang.org/x/sys/cpu"
)

var IV = [...]uint32{IV0, IV1, IV2, IV3, IV4, IV5, IV6, IV7}

var (
	HasAVX2 = cpu.X86.HasAVX2 &&
		os.Getenv("BLAKE3_DISABLE_AVX2") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasSSE41 = cpu.X86.HasSSE41 &&
		os.Getenv("BLAKE3_DISABLE_SSE41") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	// ASIMD is ARM64's Advanced SIMD implementation, commonly called NEON.
	// It is part of the ARMv8-A baseline.
	HasNEON = cpu.ARM64.HasASIMD &&
		os.Getenv("BLAKE3_DISABLE_NEON") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""
)
