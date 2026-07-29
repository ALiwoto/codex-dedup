package fastCdcTests

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/fastCdc"
)

func BenchmarkFastCdcChunking(b *testing.B) {
	for _, benchmark := range []struct {
		name string
		size int
	}{
		{name: "1MiB", size: 1024 * 1024},
		{name: "16MiB", size: 16 * 1024 * 1024},
	} {
		input := deterministicBytes(benchmark.size)
		b.Run(benchmark.name, func(b *testing.B) {
			benchmarkChunker(b, input, fastCdc.DefaultChunkerConfig())
		})
	}
}

func BenchmarkFastCdcConfigurations(b *testing.B) {
	input := deterministicBytes(16 * 1024 * 1024)
	for _, benchmark := range []struct {
		name   string
		config fastCdc.ChunkerConfig
	}{
		{
			name: "8_32_128_KiB",
			config: fastCdc.ChunkerConfig{
				MinimumSize: 8 * 1024,
				AverageSize: 32 * 1024,
				MaximumSize: 128 * 1024,
			},
		},
		{
			name:   "16_64_256_KiB",
			config: fastCdc.DefaultChunkerConfig(),
		},
		{
			name: "32_128_512_KiB",
			config: fastCdc.ChunkerConfig{
				MinimumSize: 32 * 1024,
				AverageSize: 128 * 1024,
				MaximumSize: 512 * 1024,
			},
		},
	} {
		config := benchmark.config
		b.Run(benchmark.name, func(b *testing.B) {
			benchmarkChunker(b, input, config)
		})
	}
}

func benchmarkChunker(b *testing.B, input []byte, config fastCdc.ChunkerConfig) {
	reader := bytes.NewReader(input)
	chunker, err := fastCdc.NewChunker(reader, config)
	if err != nil {
		b.Fatalf("create chunker: %v", err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	var summary fastCdc.ChunkingSummary
	for b.Loop() {
		reader.Reset(input)
		chunker.Reset(reader)
		for {
			_, nextErr := chunker.Next()
			if errors.Is(nextErr, io.EOF) {
				break
			}
			if nextErr != nil {
				b.Fatalf("chunk input: %v", nextErr)
			}
		}

		var complete bool
		summary, complete = chunker.Summary()
		if !complete {
			b.Fatal("chunker did not complete")
		}
	}

	b.ReportMetric(float64(len(input))/float64(summary.ChunkCount), "avg-chunk-B")
	b.ReportMetric(float64(summary.ChunkCount), "chunks/op")
	runtime.KeepAlive(summary)
}
