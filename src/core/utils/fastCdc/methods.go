package fastCdc

import (
	"errors"
	"io"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

// Reset prepares the chunker to read another stream with the same configuration.
func (c *Chunker) Reset(reader io.Reader) {
	c.reader = reader
	c.bufferStart = 0
	c.bufferEnd = 0
	c.summary = ChunkingSummary{}
	c.readEOF = false
	c.readError = nil
	c.finished = false
	c.bodyHasher.Reset()
}

// Next returns the next content-defined chunk. Chunk.Data remains valid until
// the next call to Next or Reset.
func (c *Chunker) Next() (Chunk, error) {
	if c.finished {
		return Chunk{}, io.EOF
	}

	if c.bufferEnd-c.bufferStart < c.config.MaximumSize && !c.readEOF && c.readError == nil {
		c.fillBuffer()
	}

	available := c.bufferEnd - c.bufferStart
	if available == 0 {
		if c.readError != nil {
			c.finished = true
			return Chunk{}, c.readError
		}

		c.finish()
		return Chunk{}, io.EOF
	}
	if available < c.config.MaximumSize && c.readError != nil {
		c.finished = true
		return Chunk{}, c.readError
	}

	chunkLength := c.findChunkLength(c.buffer[c.bufferStart:c.bufferEnd])
	chunkData := c.buffer[c.bufferStart : c.bufferStart+chunkLength]
	chunk := Chunk{
		Offset: c.summary.TotalSize,
		Length: chunkLength,
		Digest: blake3.Sum256(chunkData),
		Data:   chunkData,
	}

	_, _ = c.bodyHasher.Write(chunkData)
	c.bufferStart += chunkLength
	c.summary.TotalSize += uint64(chunkLength)
	c.summary.ChunkCount++

	return chunk, nil
}

// Summary returns the complete-stream digest and totals after Next returns
// io.EOF. It returns false when the stream is incomplete or failed.
func (c *Chunker) Summary() (ChunkingSummary, bool) {
	return c.summary, c.finished && c.readError == nil
}

func (c *Chunker) fillBuffer() {
	needed := c.config.MaximumSize - (c.bufferEnd - c.bufferStart)
	if len(c.buffer)-c.bufferEnd < needed {
		copy(c.buffer, c.buffer[c.bufferStart:c.bufferEnd])
		c.bufferEnd -= c.bufferStart
		c.bufferStart = 0
	}

	readCount, err := io.ReadFull(c.reader, c.buffer[c.bufferEnd:c.bufferEnd+needed])
	c.bufferEnd += readCount

	switch {
	case err == nil:
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		c.readEOF = true
	default:
		c.readError = err
	}
}

func (c *Chunker) findChunkLength(data []byte) int {
	limit := min(len(data), c.config.MaximumSize)
	if limit <= c.config.MinimumSize {
		return limit
	}

	normalSize := min(c.config.AverageSize, limit)
	rollingHash := uint64(0)
	index := c.config.MinimumSize

	for ; index < normalSize; index++ {
		rollingHash = rollingHash<<1 + gearTable[data[index]]
		if rollingHash&c.maskSmall == 0 {
			return index + 1
		}
	}
	for ; index < limit; index++ {
		rollingHash = rollingHash<<1 + gearTable[data[index]]
		if rollingHash&c.maskLarge == 0 {
			return index + 1
		}
	}

	return limit
}

func (c *Chunker) finish() {
	c.finished = true
	output := c.summary.BodyDigest[:0]
	c.bodyHasher.Sum(output)
}
