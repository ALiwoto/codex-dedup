//go:build !arm64

package hashNEON

import "github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeHash/hashPure"

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	hashPure.HashF(input, length, counter, flags, key, out, chain)
}

func HashP(left, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int) {
	hashPure.HashP(left, right, flags, key, out, n)
}
