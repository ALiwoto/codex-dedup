package blake3Tests

import (
	"bytes"
	"encoding/hex"
	"hash"
	"io"
	"strconv"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
)

func TestOfficialVectors(t *testing.T) {
	for _, vector := range vectors[:officialVectorCount] {
		input := vector.input()
		t.Run("length_"+strconv.Itoa(vector.inputLen), func(t *testing.T) {
			assertDigest(t, blake3.New(), input, vector.hash)

			keyedHasher, err := blake3.NewKeyed([]byte(testVectorKey))
			if err != nil {
				t.Fatalf("create keyed hasher: %v", err)
			}
			assertDigest(t, keyedHasher, input, vector.keyedHash)
			assertDigest(t, blake3.NewDeriveKey(testVectorContext), input, vector.deriveKey)
		})
	}
}

func TestLargeRegressionVectors(t *testing.T) {
	for _, vector := range vectors[officialVectorCount:] {
		assertDigest(t, blake3.New(), vector.input(), vector.hash)
	}
}

func TestHasherImplementsStandardHash(t *testing.T) {
	var standardHasher hash.Hash = blake3.New()
	if standardHasher.Size() != 32 {
		t.Fatalf("unexpected digest size: %d", standardHasher.Size())
	}
	if standardHasher.BlockSize() != 64 {
		t.Fatalf("unexpected block size: %d", standardHasher.BlockSize())
	}

	_, _ = standardHasher.Write([]byte("some data"))
	expected := "b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2"
	if actual := hex.EncodeToString(standardHasher.Sum(nil)); actual != expected {
		t.Fatalf("unexpected digest: %s", actual)
	}
}

func TestSum256MatchesIncrementalHasher(t *testing.T) {
	input := make([]byte, 512*1024)
	for index := range input {
		input[index] = byte(index % 251)
	}

	for _, size := range []int{0, 1, 63, 64, 65, 1023, 1024, 1025, 8192, 8193, 16 * 1024, 64 * 1024, 256 * 1024, len(input)} {
		hasher := blake3.New()
		for offset := 0; offset < size; {
			end := offset + 997
			if end > size {
				end = size
			}
			_, _ = hasher.Write(input[offset:end])
			offset = end
		}

		expected := hasher.Sum(nil)
		actual := blake3.Sum256(input[:size])
		if !bytes.Equal(actual[:], expected) {
			t.Fatalf("one-shot and incremental digests differ for %d bytes", size)
		}
	}
}

func TestSum512MatchesDigest(t *testing.T) {
	input := make([]byte, 256*1024)
	for index := range input {
		input[index] = byte(index % 251)
	}

	hasher := blake3.New()
	_, _ = hasher.Write(input)
	expected := make([]byte, 64)
	_, _ = hasher.Digest().Read(expected)
	actual := blake3.Sum512(input)
	if !bytes.Equal(actual[:], expected) {
		t.Fatal("Sum512 and digest output differ")
	}
}

func TestHasherCloneAndDigestSeek(t *testing.T) {
	hasher := blake3.New()
	_, _ = hasher.WriteString("shared-prefix")
	clone := hasher.Clone()

	_, _ = hasher.WriteString("-left")
	_, _ = clone.WriteString("-right")
	if bytes.Equal(hasher.Sum(nil), clone.Sum(nil)) {
		t.Fatal("independently modified clones produced the same digest")
	}

	digest := hasher.Digest()
	fullOutput := make([]byte, 256)
	_, _ = digest.Read(fullOutput)

	if _, err := digest.Seek(73, io.SeekStart); err != nil {
		t.Fatalf("seek digest: %v", err)
	}
	partialOutput := make([]byte, 111)
	_, _ = digest.Read(partialOutput)
	if !bytes.Equal(partialOutput, fullOutput[73:184]) {
		t.Fatal("digest output changed after seeking")
	}
}

func TestKeyedHasherRejectsInvalidKeyLength(t *testing.T) {
	if _, err := blake3.NewKeyed(make([]byte, 31)); err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func assertDigest(t *testing.T, hasher *blake3.Hasher, input []byte, expectedHex string) {
	t.Helper()

	_, _ = hasher.Write(input)
	output := make([]byte, len(expectedHex)/2)
	_, _ = hasher.Digest().Read(output)
	if actual := hex.EncodeToString(output); actual != expectedHex {
		t.Fatalf("unexpected digest\nexpected: %s\nactual:   %s", expectedHex, actual)
	}
}
