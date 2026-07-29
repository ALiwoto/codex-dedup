package blakeHash

import (
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashAVX2"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashNEON"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashPure"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	if blakeValues.HasAVX2 && length > 2*blakeValues.ChunkLen {
		hashAVX2.HashF(input, length, counter, flags, key, out, chain)
	} else if blakeValues.HasNEON && length > 2*blakeValues.ChunkLen {
		hashNEON.HashF(input, length, counter, flags, key, out, chain)
	} else {
		hashPure.HashF(input, length, counter, flags, key, out, chain)
	}
}

func HashP(left, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int) {
	if blakeValues.HasAVX2 && n >= 2 {
		hashAVX2.HashP(left, right, flags, key, out, n)
	} else if blakeValues.HasNEON && n >= 2 {
		hashNEON.HashP(left, right, flags, key, out, n)
	} else {
		hashPure.HashP(left, right, flags, key, out, n)
	}
}
