package fastCdc

import (
	"io"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

// ChunkerConfig defines the minimum, target-average, and maximum chunk sizes.
type ChunkerConfig struct {
	MinimumSize int
	AverageSize int
	MaximumSize int
}

// Chunk describes one content-defined byte range and its BLAKE3-256 digest.
type Chunk struct {
	Offset uint64
	Length int
	Digest [32]byte
	Data   []byte
}

// ChunkingSummary describes a successfully consumed complete stream.
type ChunkingSummary struct {
	TotalSize  uint64
	ChunkCount uint64
	BodyDigest [32]byte
}

// Chunker incrementally divides one reader into deterministic chunks.
type Chunker struct {
	config      ChunkerConfig
	reader      io.Reader
	buffer      []byte
	bufferStart int
	bufferEnd   int
	maskSmall   uint64
	maskLarge   uint64
	bodyHasher  *blake3.Hasher
	summary     ChunkingSummary
	readEOF     bool
	readError   error
	finished    bool
}
