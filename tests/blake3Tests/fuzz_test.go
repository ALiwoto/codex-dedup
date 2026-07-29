package blake3Tests

import (
	"bytes"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

func FuzzSum256MatchesIncrementalHasher(f *testing.F) {
	f.Add([]byte(nil), uint16(1))
	f.Add([]byte("small input"), uint16(3))
	f.Add(make([]byte, 8193), uint16(1024))

	f.Fuzz(func(t *testing.T, input []byte, writeSize uint16) {
		if writeSize == 0 {
			writeSize = 1
		}

		hasher := blake3.New()
		for offset := 0; offset < len(input); {
			end := min(offset+int(writeSize), len(input))
			_, _ = hasher.Write(input[offset:end])
			offset = end
		}

		oneShot := blake3.Sum256(input)
		if !bytes.Equal(oneShot[:], hasher.Sum(nil)) {
			t.Fatal("one-shot and incremental digests differ")
		}
	})
}
