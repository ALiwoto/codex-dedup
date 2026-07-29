package fastCdcTests

import (
	"bytes"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
	"github.com/ALiwoto/codex-dedup/src/core/utils/fastCdc"
)

func FuzzChunkingPreservesInput(f *testing.F) {
	f.Add([]byte(nil), uint16(1))
	f.Add([]byte("small input"), uint16(3))
	f.Add(deterministicBytes(4097), uint16(257))

	config := fastCdc.ChunkerConfig{
		MinimumSize: 64,
		AverageSize: 256,
		MaximumSize: 1024,
	}
	f.Fuzz(func(t *testing.T, input []byte, maximumRead uint16) {
		if maximumRead == 0 {
			maximumRead = 1
		}

		reader := &limitedReader{
			reader:      bytes.NewReader(input),
			maximumRead: int(maximumRead),
		}
		chunks, reconstructed, summary, err := collectChunks(reader, config)
		if err != nil {
			t.Fatalf("chunk input: %v", err)
		}
		if !bytes.Equal(reconstructed, input) {
			t.Fatal("reconstructed chunks differ from input")
		}
		if summary.BodyDigest != blake3.Sum256(input) {
			t.Fatal("whole-body digest differs from input digest")
		}

		for index, chunk := range chunks {
			if chunk.Length > config.MaximumSize {
				t.Fatalf("chunk %d exceeds maximum", index)
			}
			if index != len(chunks)-1 && chunk.Length < config.MinimumSize {
				t.Fatalf("non-final chunk %d is below minimum", index)
			}
		}
	})
}
