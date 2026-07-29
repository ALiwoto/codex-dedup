package blake3Tests

import (
	"crypto/sha256"
	"runtime"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

func BenchmarkChunkDigest(b *testing.B) {
	for _, benchmark := range []struct {
		name string
		size int
	}{
		{name: "16KiB", size: 16 * 1024},
		{name: "64KiB", size: 64 * 1024},
		{name: "256KiB", size: 256 * 1024},
		{name: "1MiB", size: 1024 * 1024},
	} {
		size := benchmark.size
		input := make([]byte, size)
		for index := range input {
			input[index] = byte(index % 251)
		}

		b.Run("BLAKE3_"+benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))
			var digest [32]byte
			for b.Loop() {
				digest = blake3.Sum256(input)
			}
			runtime.KeepAlive(digest)
		})

		b.Run("SHA256_"+benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(size))
			var digest [32]byte
			for b.Loop() {
				digest = sha256.Sum256(input)
			}
			runtime.KeepAlive(digest)
		})
	}
}

func BenchmarkIncrementalHasher(b *testing.B) {
	input := make([]byte, 256*1024)
	hasher := blake3.New()
	output := make([]byte, 0, 32)

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		hasher.Reset()
		for offset := 0; offset < len(input); offset += 4096 {
			_, _ = hasher.Write(input[offset : offset+4096])
		}
		output = hasher.Sum(output[:0])
	}
	runtime.KeepAlive(output)
}
