package blake3

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeAlg"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeUtils"
	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3/blakeValues"
)

// Read fills output with bytes from the BLAKE3 extendable-output stream.
func (d *Digest) Read(output []byte) (int, error) {
	total := len(output)

	if d.bufn > 0 {
		copied := d.slowCopy(output)
		output = output[copied:]
		d.bufn -= copied
	}

	for len(output) >= blakeValues.BlockLen {
		d.fillBuf()

		if blakeValues.OptimizeLittleEndian {
			*(*[blakeValues.BlockLen]byte)(unsafe.Pointer(&output[0])) =
				*(*[blakeValues.BlockLen]byte)(unsafe.Pointer(&d.buf[0]))
		} else {
			blakeUtils.WordsToBytes(&d.buf, output)
		}

		output = output[blakeValues.BlockLen:]
		d.bufn = 0
	}

	if len(output) == 0 {
		return total, nil
	}

	d.fillBuf()
	d.bufn -= d.slowCopy(output)

	return total, nil
}

// Seek changes the output position. SeekStart and SeekCurrent are supported.
func (d *Digest) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekEnd:
		return 0, fmt.Errorf("seek from end not supported")
	case io.SeekCurrent:
		offset += int64(blakeValues.BlockLen*d.counter) - int64(d.bufn)
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	if offset < 0 {
		return 0, fmt.Errorf("seek before start")
	}

	d.setPosition(uint64(offset))
	return offset, nil
}

func (d *Digest) setPosition(position uint64) {
	d.counter = position / blakeValues.BlockLen
	d.fillBuf()
	d.bufn -= int(position % blakeValues.BlockLen)
}

func (d *Digest) slowCopy(output []byte) int {
	offset := uint(blakeValues.BlockLen-d.bufn) % blakeValues.BlockLen
	if blakeValues.OptimizeLittleEndian {
		return copy(
			output,
			(*[blakeValues.BlockLen]byte)(unsafe.Pointer(&d.buf[0]))[offset:],
		)
	}

	var temporary [blakeValues.BlockLen]byte
	blakeUtils.WordsToBytes(&d.buf, temporary[:])
	return copy(output, temporary[offset:])
}

func (d *Digest) fillBuf() {
	blakeAlg.Compress(&d.chain, &d.block, d.counter, d.blen, d.flags, &d.buf)
	d.counter++
	d.bufn = blakeValues.BlockLen
}
