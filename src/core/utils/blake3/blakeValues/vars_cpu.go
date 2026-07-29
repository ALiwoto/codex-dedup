//go:build !purego

package blakeValues

import (
	"os"

	"golang.org/x/sys/cpu"
)

var (
	HasAVX2 = cpu.X86.HasAVX2 &&
		os.Getenv("BLAKE3_DISABLE_AVX2") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasSSE41 = cpu.X86.HasSSE41 &&
		os.Getenv("BLAKE3_DISABLE_SSE41") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasNEON = cpu.ARM64.HasASIMD &&
		os.Getenv("BLAKE3_DISABLE_NEON") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""
)
