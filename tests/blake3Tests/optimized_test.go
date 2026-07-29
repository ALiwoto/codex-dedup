package blake3Tests

import (
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression/compressPure"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression/compressSSE41"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashAVX2"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashNEON"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashPure"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
	"github.com/ALiwoto/codex-dedup/src/core/utils/pcgUtils"
)

type hashFFunction func(*[8192]byte, uint64, uint64, uint32, *[8]uint32, *[64]uint32, *[8]uint32)
type hashPFunction func(*[64]uint32, *[64]uint32, uint32, *[8]uint32, *[64]uint32, int)

func TestSSE41CompressionMatchesPureGo(t *testing.T) {
	if !blakeValues.HasSSE41 {
		t.Skip("SSE4.1 is unavailable or disabled")
	}

	random := pcgUtils.New(1)
	for iteration := 0; iteration < 10_000; iteration++ {
		var chain [8]uint32
		var block [16]uint32
		for index := range chain {
			chain[index] = random.Uint32()
		}
		for index := range block {
			block[index] = random.Uint32()
		}

		counter := random.Uint64()
		blockLength := random.Uint32()
		flags := random.Uint32()
		var optimizedOutput [16]uint32
		var pureOutput [16]uint32
		compressSSE41.Compress(&chain, &block, counter, blockLength, flags, &optimizedOutput)
		compressPure.Compress(&chain, &block, counter, blockLength, flags, &pureOutput)
		if optimizedOutput != pureOutput {
			t.Fatalf("SSE4.1 output differs on iteration %d", iteration)
		}
	}
}

func TestAVX2HashMatchesPureGo(t *testing.T) {
	if !blakeValues.HasAVX2 {
		t.Skip("AVX2 is unavailable or disabled")
	}

	testOptimizedHash(t, hashAVX2.HashF, hashAVX2.HashP)
}

func TestNEONHashMatchesPureGo(t *testing.T) {
	if !blakeValues.HasNEON {
		t.Skip("NEON is unavailable or disabled")
	}

	testOptimizedHash(t, hashNEON.HashF, hashNEON.HashP)
}

func testOptimizedHash(t *testing.T, hashF hashFFunction, hashP hashPFunction) {
	t.Helper()

	random := pcgUtils.New(2)
	var input [8192]byte
	for index := range input {
		input[index] = byte(index % 251)
	}

	for length := 0; length <= len(input); length++ {
		var key [8]uint32
		for index := range key {
			key[index] = random.Uint32()
		}
		counter := random.Uint64()
		flags := random.Uint32()

		var optimizedChain [8]uint32
		var pureChain [8]uint32
		var optimizedOutput [64]uint32
		var pureOutput [64]uint32
		hashF(&input, uint64(length), counter, flags, &key, &optimizedOutput, &optimizedChain)
		hashPure.HashF(&input, uint64(length), counter, flags, &key, &pureOutput, &pureChain)

		completeChunks := length / blakeValues.ChunkLen
		for chunk := 0; chunk < completeChunks; chunk++ {
			for word := 0; word < 8; word++ {
				index := chunk + 8*word
				if optimizedOutput[index] != pureOutput[index] {
					t.Fatalf("optimized full-chunk output differs at input length %d", length)
				}
			}
		}
		if length%blakeValues.ChunkLen != 0 && optimizedChain != pureChain {
			t.Fatalf("optimized partial-chunk output differs at input length %d", length)
		}
	}

	var left [64]uint32
	var right [64]uint32
	for index := range left {
		left[index] = random.Uint32()
		right[index] = random.Uint32()
	}
	for count := 1; count <= 8; count++ {
		var key [8]uint32
		for index := range key {
			key[index] = random.Uint32()
		}
		var optimizedOutput [64]uint32
		var pureOutput [64]uint32
		hashP(&left, &right, 0, &key, &optimizedOutput, count)
		hashPure.HashP(&left, &right, 0, &key, &pureOutput, count)
		for column := 0; column < count; column++ {
			for word := 0; word < 8; word++ {
				index := column + 8*word
				if optimizedOutput[index] != pureOutput[index] {
					t.Fatalf("optimized parent output differs for %d columns", count)
				}
			}
		}
	}
}
