package blake3

import (
	"math/bits"
	"unsafe"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeAlg"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeUtils"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

// Write adds bytes to the hash state. It never returns an error.
func (h *Hasher) Write(input []byte) (int, error) {
	h.h.update(input)
	return len(input), nil
}

// WriteString adds a string without first converting it to a byte slice.
func (h *Hasher) WriteString(input string) (int, error) {
	h.h.updateString(input)
	return len(input), nil
}

// Reset restores the hasher to its initial state.
func (h *Hasher) Reset() {
	h.h.reset()
}

// Clone returns an independent snapshot of the hasher.
func (h *Hasher) Clone() *Hasher {
	return &Hasher{size: h.size, h: h.h}
}

// Size returns the number of digest bytes appended by Sum.
func (h *Hasher) Size() int {
	return h.size
}

// BlockSize returns BLAKE3's compression block size.
func (h *Hasher) BlockSize() int {
	return blakeValues.BlockLen
}

// Sum appends the current 32-byte digest without modifying the hash state.
func (h *Hasher) Sum(output []byte) []byte {
	if top := len(output) + h.size; top <= cap(output) && top >= len(output) {
		h.h.finalize(output[len(output):top])
		return output[:top]
	}

	temporary := make([]byte, h.size)
	h.h.finalize(temporary)
	return append(output, temporary...)
}

// Digest returns a snapshot that can read and seek through BLAKE3 XOF output.
func (h *Hasher) Digest() *Digest {
	var digest Digest
	h.h.finalizeDigest(&digest)
	return &digest
}

func (h *hasher) reset() {
	h.len = 0
	h.chunks = 0
	h.stack.occ = 0
	h.stack.lvls = [8]uint8{}
	h.stack.bufn = 0
}

func (h *hasher) update(input []byte) {
	h.updateString(unsafe.String(unsafe.SliceData(input), len(input)))
}

func (h *hasher) updateString(input string) {
	var block *[8192]byte

	for len(input) > 0 {
		if h.len == 0 && len(input) > len(h.buf) {
			block = (*[8192]byte)(unsafe.Pointer(unsafe.StringData(input)))
			input = input[len(h.buf):]
		} else if h.len < uint64(len(h.buf)) {
			written := copy(h.buf[h.len:], input)
			h.len += uint64(written)
			input = input[written:]
			continue
		} else {
			block = &h.buf
		}

		h.consume(block)
		h.len = 0
		h.chunks += 8
	}
}

func (h *hasher) consume(input *[8192]byte) {
	var output chainVector
	var chain [8]uint32
	blakeAlg.HashF(input, 8192, h.chunks, h.flags, &h.key, &output, &chain)
	h.stack.pushN(0, &output, 8, h.flags, &h.key)
}

func (h *hasher) finalize(output []byte) {
	var digest Digest
	h.finalizeDigest(&digest)
	_, _ = digest.Read(output)
}

func (h *hasher) finalizeDigest(digest *Digest) {
	if h.chunks == 0 && h.len <= blakeValues.ChunkLen {
		compressAll(digest, h.buf[:h.len], h.flags, h.key)
		return
	}

	digest.chain = h.key
	digest.flags = h.flags | blakeValues.Flag_ChunkEnd

	if h.len > blakeValues.BlockLen {
		var buffer chainVector
		blakeAlg.HashF(&h.buf, h.len, h.chunks, h.flags, &h.key, &buffer, &digest.chain)

		if h.len > blakeValues.ChunkLen {
			complete := (h.len - 1) / blakeValues.ChunkLen
			h.stack.pushN(0, &buffer, int(complete), h.flags, &h.key)
			h.chunks += complete
			h.len = uint64(copy(h.buf[:], h.buf[complete*blakeValues.ChunkLen:h.len]))
		}
	}

	if h.len <= blakeValues.BlockLen {
		digest.flags |= blakeValues.Flag_ChunkStart
	}

	digest.counter = h.chunks
	digest.blen = uint32(h.len) % blakeValues.BlockLen

	base := h.len / blakeValues.BlockLen * blakeValues.BlockLen
	if h.len > 0 && digest.blen == 0 {
		digest.blen = blakeValues.BlockLen
		base -= blakeValues.BlockLen
	}

	if blakeValues.OptimizeLittleEndian {
		copy((*[blakeValues.BlockLen]byte)(unsafe.Pointer(&digest.block[0]))[:], h.buf[base:h.len])
	} else {
		var temporary [blakeValues.BlockLen]byte
		copy(temporary[:], h.buf[base:h.len])
		blakeUtils.BytesToWords(&temporary, &digest.block)
	}

	for h.stack.bufn > 0 {
		h.stack.flush(h.flags, &h.key)
	}

	var compressed [16]uint32
	for occupied := h.stack.occ; occupied != 0; occupied &= occupied - 1 {
		column := uint(bits.TrailingZeros64(occupied)) % 64

		blakeAlg.Compress(
			&digest.chain,
			&digest.block,
			digest.counter,
			digest.blen,
			digest.flags,
			&compressed,
		)

		*(*[8]uint32)(digest.block[0:8]) = h.stack.stack[column]
		*(*[8]uint32)(digest.block[8:16]) = *(*[8]uint32)(compressed[0:8])

		if occupied == h.stack.occ {
			digest.chain = h.key
			digest.counter = 0
			digest.blen = blakeValues.BlockLen
			digest.flags = h.flags | blakeValues.Flag_Parent
		}
	}

	digest.flags |= blakeValues.Flag_Root
}
