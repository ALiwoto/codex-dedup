package blakeAlg

import (
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash"
)

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	blakeHash.HashF(input, length, counter, flags, key, out, chain)
}

func HashP(left, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int) {
	blakeHash.HashP(left, right, flags, key, out, n)
}

func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	blakeCompression.Compress(chain, block, counter, blen, flags, out)
}
