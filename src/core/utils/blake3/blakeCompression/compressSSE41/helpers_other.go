//go:build !amd64
// +build !amd64

package compressSSE41

import "github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression/compressPure"

func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	compressPure.Compress(chain, block, counter, blen, flags, out)
}
