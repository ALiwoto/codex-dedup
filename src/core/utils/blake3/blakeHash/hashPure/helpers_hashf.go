package hashPure

import (
	"unsafe"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeCompression"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeUtils"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	var tmp [16]uint32
	var block [16]uint32

	for i := uint64(0); blakeValues.ChunkLen*i < length && i < 8; i++ {
		bchain := *key
		bflags := flags | blakeValues.Flag_ChunkStart
		start := blakeValues.ChunkLen * i

		for n := uint64(0); n < 16; n++ {
			if n == 15 {
				bflags |= blakeValues.Flag_ChunkEnd
			}
			if start+64*n >= length {
				break
			}
			if start+64+64*n >= length {
				*chain = bchain
			}

			var blockPtr *[16]uint32
			if blakeValues.OptimizeLittleEndian {
				blockPtr = (*[16]uint32)(unsafe.Pointer(&input[blakeValues.ChunkLen*i+blakeValues.BlockLen*n]))
			} else {
				blakeUtils.BytesToWords((*[64]uint8)(input[blakeValues.ChunkLen*i+blakeValues.BlockLen*n:]), &block)
				blockPtr = &block
			}

			blakeCompression.Compress(&bchain, blockPtr, counter, blakeValues.BlockLen, bflags, &tmp)

			bchain = *(*[8]uint32)(tmp[0:8])
			bflags = flags
		}

		out[i+0] = bchain[0]
		out[i+8] = bchain[1]
		out[i+16] = bchain[2]
		out[i+24] = bchain[3]
		out[i+32] = bchain[4]
		out[i+40] = bchain[5]
		out[i+48] = bchain[6]
		out[i+56] = bchain[7]

		counter++
	}
}
