package blake3

import (
	"errors"
	"unsafe"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeAlg"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeUtils"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

// New returns an unkeyed BLAKE3 hasher with a 32-byte digest size.
func New() *Hasher {
	return &Hasher{
		size: 32,
		h: hasher{
			key: blakeValues.IV,
		},
	}
}

// NewKeyed returns a keyed BLAKE3 hasher using the supplied 32-byte key.
func NewKeyed(key []byte) (*Hasher, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key size")
	}

	h := &Hasher{
		size: 32,
		h: hasher{
			flags: blakeValues.Flag_Keyed,
		},
	}
	blakeUtils.KeyFromBytes(key, &h.h.key)

	return h, nil
}

// DeriveKey derives len(out) bytes from material under the supplied context.
func DeriveKey(context string, material []byte, out []byte) {
	h := NewDeriveKey(context)
	_, _ = h.Write(material)
	_, _ = h.Digest().Read(out)
}

// NewDeriveKey returns a hasher initialized for BLAKE3 key derivation.
func NewDeriveKey(context string) *Hasher {
	h := &Hasher{
		size: 32,
		h: hasher{
			key:   blakeValues.IV,
			flags: blakeValues.Flag_DeriveKeyContext,
		},
	}

	var contextKey [32]byte
	_, _ = h.WriteString(context)
	_, _ = h.Digest().Read(contextKey[:])

	h.Reset()
	blakeUtils.KeyFromBytes(contextKey[:], &h.h.key)
	h.h.flags = blakeValues.Flag_DeriveKeyMaterial

	return h
}

// Sum256 returns the first 256 bits of the unkeyed BLAKE3 digest.
func Sum256(data []byte) (sum [32]byte) {
	if len(data) <= blakeValues.ChunkLen {
		sumSmall(data, sum[:])
	} else {
		sumLarge(data, sum[:])
	}
	return sum
}

// Sum512 returns the first 512 bits of the unkeyed BLAKE3 digest.
func Sum512(data []byte) (sum [64]byte) {
	if len(data) <= blakeValues.ChunkLen {
		sumSmall(data, sum[:])
	} else {
		sumLarge(data, sum[:])
	}
	return sum
}

func sumSmall(data []byte, out []byte) {
	var digest Digest
	compressAll(&digest, data, 0, blakeValues.IV)
	_, _ = digest.Read(out)
}

func sumLarge(data []byte, out []byte) {
	h := hasher{key: blakeValues.IV}
	h.update(data)
	h.finalize(out)
}

func copyChain(in *chainVector, inputColumn int, out *chainVector, outputColumn int) {
	inputPointer := unsafe.Add(unsafe.Pointer(in), inputColumn*4)
	outputPointer := unsafe.Add(unsafe.Pointer(out), outputColumn*4)

	*(*uint32)(unsafe.Add(outputPointer, 0*32)) = *(*uint32)(unsafe.Add(inputPointer, 0*32))
	*(*uint32)(unsafe.Add(outputPointer, 1*32)) = *(*uint32)(unsafe.Add(inputPointer, 1*32))
	*(*uint32)(unsafe.Add(outputPointer, 2*32)) = *(*uint32)(unsafe.Add(inputPointer, 2*32))
	*(*uint32)(unsafe.Add(outputPointer, 3*32)) = *(*uint32)(unsafe.Add(inputPointer, 3*32))
	*(*uint32)(unsafe.Add(outputPointer, 4*32)) = *(*uint32)(unsafe.Add(inputPointer, 4*32))
	*(*uint32)(unsafe.Add(outputPointer, 5*32)) = *(*uint32)(unsafe.Add(inputPointer, 5*32))
	*(*uint32)(unsafe.Add(outputPointer, 6*32)) = *(*uint32)(unsafe.Add(inputPointer, 6*32))
	*(*uint32)(unsafe.Add(outputPointer, 7*32)) = *(*uint32)(unsafe.Add(inputPointer, 7*32))
}

func readChain(in *chainVector, column int, out *[8]uint32) {
	inputPointer := unsafe.Add(unsafe.Pointer(in), column*4)

	out[0] = *(*uint32)(unsafe.Add(inputPointer, 0*32))
	out[1] = *(*uint32)(unsafe.Add(inputPointer, 1*32))
	out[2] = *(*uint32)(unsafe.Add(inputPointer, 2*32))
	out[3] = *(*uint32)(unsafe.Add(inputPointer, 3*32))
	out[4] = *(*uint32)(unsafe.Add(inputPointer, 4*32))
	out[5] = *(*uint32)(unsafe.Add(inputPointer, 5*32))
	out[6] = *(*uint32)(unsafe.Add(inputPointer, 6*32))
	out[7] = *(*uint32)(unsafe.Add(inputPointer, 7*32))
}

func writeChain(in *[8]uint32, out *chainVector, column int) {
	outputPointer := unsafe.Add(unsafe.Pointer(out), column*4)

	*(*uint32)(unsafe.Add(outputPointer, 0*32)) = in[0]
	*(*uint32)(unsafe.Add(outputPointer, 1*32)) = in[1]
	*(*uint32)(unsafe.Add(outputPointer, 2*32)) = in[2]
	*(*uint32)(unsafe.Add(outputPointer, 3*32)) = in[3]
	*(*uint32)(unsafe.Add(outputPointer, 4*32)) = in[4]
	*(*uint32)(unsafe.Add(outputPointer, 5*32)) = in[5]
	*(*uint32)(unsafe.Add(outputPointer, 6*32)) = in[6]
	*(*uint32)(unsafe.Add(outputPointer, 7*32)) = in[7]
}

func compressAll(digest *Digest, input []byte, flags uint32, key [8]uint32) {
	var compressed [16]uint32

	digest.chain = key
	digest.flags = flags | blakeValues.Flag_ChunkStart

	for len(input) > blakeValues.BlockLen {
		buffer := (*[blakeValues.BlockLen]byte)(input)

		var block *[16]uint32
		if blakeValues.OptimizeLittleEndian {
			block = (*[16]uint32)(unsafe.Pointer(buffer))
		} else {
			block = &digest.block
			blakeUtils.BytesToWords(buffer, block)
		}

		blakeAlg.Compress(
			&digest.chain,
			block,
			0,
			blakeValues.BlockLen,
			digest.flags,
			&compressed,
		)

		digest.chain = *(*[8]uint32)(compressed[0:8])
		digest.flags &^= blakeValues.Flag_ChunkStart
		input = input[blakeValues.BlockLen:]
	}

	if blakeValues.OptimizeLittleEndian {
		copy((*[blakeValues.BlockLen]byte)(unsafe.Pointer(&digest.block[0]))[:], input)
	} else {
		var buffer [blakeValues.BlockLen]byte
		copy(buffer[:], input)
		blakeUtils.BytesToWords(&buffer, &digest.block)
	}

	digest.blen = uint32(len(input))
	digest.flags |= blakeValues.Flag_ChunkEnd | blakeValues.Flag_Root
}
