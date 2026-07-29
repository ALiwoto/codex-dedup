package fastCdc

import (
	"errors"
	"io"
	"math/bits"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		MinimumSize: DefaultMinimumSize,
		AverageSize: DefaultAverageSize,
		MaximumSize: DefaultMaximumSize,
	}
}

func NewChunker(reader io.Reader, config ChunkerConfig) (*Chunker, error) {
	if err := validateChunkerConfig(config); err != nil {
		return nil, err
	}

	averageBits := bits.Len(uint(config.AverageSize)) - 1
	chunker := &Chunker{
		config:     config,
		buffer:     make([]byte, config.MaximumSize*2),
		maskSmall:  maskForBits(averageBits + normalizationLevel),
		maskLarge:  maskForBits(averageBits - normalizationLevel),
		bodyHasher: blake3.New(),
	}
	chunker.Reset(reader)

	return chunker, nil
}

func validateChunkerConfig(config ChunkerConfig) error {
	if config.MinimumSize <= 0 {
		return errors.New("minimum chunk size must be positive")
	}
	if config.AverageSize < 8 {
		return errors.New("average chunk size must be at least 8 bytes for two-bit normalization")
	}
	if config.AverageSize < config.MinimumSize {
		return errors.New("average chunk size must not be smaller than minimum chunk size")
	}
	if config.MaximumSize < config.AverageSize {
		return errors.New("maximum chunk size must not be smaller than average chunk size")
	}
	if config.MaximumSize > int(^uint(0)>>1)/2 {
		return errors.New("maximum chunk size is too large")
	}

	return nil
}

func maskForBits(bitCount int) uint64 {
	if bitCount < 1 {
		bitCount = 1
	}
	if bitCount >= 64 {
		return ^uint64(0)
	}

	return uint64(1)<<bitCount - 1
}

func generateGearTable() [256]uint64 {
	var table [256]uint64
	state := gearTableSeed

	for index := range table {
		state += splitMixIncrement
		value := state
		value = (value ^ value>>30) * splitMixFactor1
		value = (value ^ value>>27) * splitMixFactor2
		table[index] = value ^ value>>31
	}

	return table
}
