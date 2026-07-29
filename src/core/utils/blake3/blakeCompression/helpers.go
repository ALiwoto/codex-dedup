package blakeCompression

import (
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression/compressPure"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression/compressSSE41"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	if blakeValues.HasSSE41 {
		compressSSE41.Compress(chain, block, counter, blen, flags, out)
	} else {
		compressPure.Compress(chain, block, counter, blen, flags, out)
	}
}
