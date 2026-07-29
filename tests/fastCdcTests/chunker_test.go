package fastCdcTests

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/ALiwoto/codex-dedup/src/core/utils/blake3"
	"github.com/ALiwoto/codex-dedup/src/core/utils/fastCdc"
)

func TestDefaultChunkingPreservesInputAndDigests(t *testing.T) {
	input := deterministicBytes(4 * 1024 * 1024)
	chunks, reconstructed, summary, err := collectChunks(bytes.NewReader(input), fastCdc.DefaultChunkerConfig())
	if err != nil {
		t.Fatalf("chunk input: %v", err)
	}
	if !bytes.Equal(reconstructed, input) {
		t.Fatal("reconstructed chunks differ from input")
	}
	if summary.TotalSize != uint64(len(input)) {
		t.Fatalf("unexpected total size: %d", summary.TotalSize)
	}
	if summary.ChunkCount != uint64(len(chunks)) {
		t.Fatalf("unexpected chunk count: %d", summary.ChunkCount)
	}
	if summary.BodyDigest != blake3.Sum256(input) {
		t.Fatal("whole-body digest differs from input digest")
	}

	config := fastCdc.DefaultChunkerConfig()
	for index, chunk := range chunks {
		if chunk.Length > config.MaximumSize {
			t.Fatalf("chunk %d exceeds maximum: %d", index, chunk.Length)
		}
		if index != len(chunks)-1 && chunk.Length < config.MinimumSize {
			t.Fatalf("non-final chunk %d is below minimum: %d", index, chunk.Length)
		}
		if chunk.Digest != blake3.Sum256(input[chunk.Offset:uint64(chunk.Offset)+uint64(chunk.Length)]) {
			t.Fatalf("chunk %d digest differs from its input range", index)
		}
	}
}

func TestChunkBoundariesDoNotDependOnReaderReadSize(t *testing.T) {
	input := deterministicBytes(2 * 1024 * 1024)
	config := fastCdc.DefaultChunkerConfig()
	expected, _, _, err := collectChunks(bytes.NewReader(input), config)
	if err != nil {
		t.Fatalf("chunk reference input: %v", err)
	}

	for _, readSize := range []int{1, 7, 4096, 65537} {
		reader := &limitedReader{reader: bytes.NewReader(input), maximumRead: readSize}
		actual, _, _, chunkErr := collectChunks(reader, config)
		if chunkErr != nil {
			t.Fatalf("chunk with %d-byte reads: %v", readSize, chunkErr)
		}
		if !slices.Equal(actual, expected) {
			t.Fatalf("boundaries changed with %d-byte reads", readSize)
		}
	}
}

func TestFastCdcResynchronizesAfterLocalChanges(t *testing.T) {
	original := deterministicBytes(6 * 1024 * 1024)
	changeOffset := 2 * 1024 * 1024
	inserted := deterministicBytes(8192)
	replacement := deterministicBytes(128 * 1024)
	for index := range replacement {
		replacement[index] ^= 0xa5
	}

	testCases := []struct {
		name     string
		modified []byte
	}{
		{
			name:     "insertion",
			modified: joinBytes(original[:changeOffset], inserted, original[changeOffset:]),
		},
		{
			name:     "deletion",
			modified: joinBytes(original[:changeOffset], original[changeOffset+len(inserted):]),
		},
		{
			name: "replacement",
			modified: joinBytes(
				original[:changeOffset],
				replacement,
				original[changeOffset+len(replacement):],
			),
		},
	}

	originalChunks, _, _, err := collectChunks(bytes.NewReader(original), fastCdc.DefaultChunkerConfig())
	if err != nil {
		t.Fatalf("chunk original input: %v", err)
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			modifiedChunks, _, _, chunkErr := collectChunks(
				bytes.NewReader(testCase.modified),
				fastCdc.DefaultChunkerConfig(),
			)
			if chunkErr != nil {
				t.Fatalf("chunk modified input: %v", chunkErr)
			}

			commonRun := longestCommonDigestRun(originalChunks, modifiedChunks)
			if commonRun < 20 {
				t.Fatalf("chunker did not resynchronize; longest shared run is %d chunks", commonRun)
			}
		})
	}
}

func TestEmptyAndSmallInputs(t *testing.T) {
	config := fastCdc.DefaultChunkerConfig()
	for _, size := range []int{0, 1, config.MinimumSize - 1, config.MinimumSize, config.MinimumSize + 1} {
		t.Run(fmt.Sprintf("%d_bytes", size), func(t *testing.T) {
			input := deterministicBytes(size)
			chunks, reconstructed, summary, err := collectChunks(bytes.NewReader(input), config)
			if err != nil {
				t.Fatalf("chunk input: %v", err)
			}
			if !bytes.Equal(reconstructed, input) {
				t.Fatal("reconstructed chunks differ from input")
			}
			if summary.BodyDigest != blake3.Sum256(input) {
				t.Fatal("whole-body digest differs from input digest")
			}

			expectedChunks := 1
			if size == 0 {
				expectedChunks = 0
			}
			if len(chunks) != expectedChunks {
				t.Fatalf("unexpected chunk count: %d", len(chunks))
			}
		})
	}
}

func TestInvalidChunkerConfigurations(t *testing.T) {
	testCases := []fastCdc.ChunkerConfig{
		{MinimumSize: 0, AverageSize: 64, MaximumSize: 256},
		{MinimumSize: 1, AverageSize: 7, MaximumSize: 256},
		{MinimumSize: 128, AverageSize: 64, MaximumSize: 256},
		{MinimumSize: 64, AverageSize: 256, MaximumSize: 128},
	}

	for _, config := range testCases {
		if _, err := fastCdc.NewChunker(bytes.NewReader(nil), config); err == nil {
			t.Fatalf("configuration should be rejected: %+v", config)
		}
	}
}

func TestReaderFailureDoesNotProduceCompleteSummary(t *testing.T) {
	expectedError := errors.New("injected reader failure")
	reader := &failingReader{
		reader:    bytes.NewReader(deterministicBytes(1024 * 1024)),
		remaining: fastCdc.DefaultMaximumSize + 123,
		err:       expectedError,
	}
	chunker, err := fastCdc.NewChunker(reader, fastCdc.DefaultChunkerConfig())
	if err != nil {
		t.Fatalf("create chunker: %v", err)
	}

	for {
		_, nextErr := chunker.Next()
		if nextErr == nil {
			continue
		}
		if !errors.Is(nextErr, expectedError) {
			t.Fatalf("unexpected reader error: %v", nextErr)
		}
		break
	}
	if _, complete := chunker.Summary(); complete {
		t.Fatal("failed stream produced a complete summary")
	}
}

func TestFastCdcV1BoundaryFixture(t *testing.T) {
	input := deterministicBytes(2 * 1024 * 1024)
	chunks, _, _, err := collectChunks(bytes.NewReader(input), fastCdc.DefaultChunkerConfig())
	if err != nil {
		t.Fatalf("chunk fixture input: %v", err)
	}

	actual := make([]int, len(chunks))
	for index, chunk := range chunks {
		actual[index] = chunk.Length
	}
	expected := []int{
		66662, 73851, 72137, 74988, 72145, 71312, 79573, 22122,
		70063, 75739, 17490, 77061, 57297, 66896, 90002, 23380,
		67933, 106025, 80525, 69906, 114755, 68316, 97678, 67465,
		73676, 81798, 154194, 80488, 23675,
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("%s boundary fixture changed:\n%v", fastCdc.AlgorithmVersion, actual)
	}
}

type collectedChunk struct {
	Offset uint64
	Length int
	Digest [32]byte
}

type limitedReader struct {
	reader      *bytes.Reader
	maximumRead int
}

func (r *limitedReader) Read(output []byte) (int, error) {
	if len(output) > r.maximumRead {
		output = output[:r.maximumRead]
	}
	return r.reader.Read(output)
}

type failingReader struct {
	reader    *bytes.Reader
	remaining int
	err       error
}

func (r *failingReader) Read(output []byte) (int, error) {
	if r.remaining == 0 {
		return 0, r.err
	}
	if len(output) > r.remaining {
		output = output[:r.remaining]
	}

	readCount, _ := r.reader.Read(output)
	r.remaining -= readCount
	return readCount, nil
}

func collectChunks(
	reader io.Reader,
	config fastCdc.ChunkerConfig,
) ([]collectedChunk, []byte, fastCdc.ChunkingSummary, error) {
	chunker, err := fastCdc.NewChunker(reader, config)
	if err != nil {
		return nil, nil, fastCdc.ChunkingSummary{}, err
	}

	var chunks []collectedChunk
	var reconstructed []byte
	for {
		chunk, nextErr := chunker.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, nil, fastCdc.ChunkingSummary{}, nextErr
		}

		chunks = append(chunks, collectedChunk{
			Offset: chunk.Offset,
			Length: chunk.Length,
			Digest: chunk.Digest,
		})
		reconstructed = append(reconstructed, chunk.Data...)
	}

	summary, complete := chunker.Summary()
	if !complete {
		return nil, nil, fastCdc.ChunkingSummary{}, errors.New("chunker did not complete")
	}

	return chunks, reconstructed, summary, nil
}

func deterministicBytes(size int) []byte {
	data := make([]byte, size)
	state := uint64(0x243f6a8885a308d3)
	for index := range data {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		data[index] = byte(state >> 56)
	}
	return data
}

func joinBytes(parts ...[]byte) []byte {
	totalLength := 0
	for _, part := range parts {
		totalLength += len(part)
	}

	joined := make([]byte, 0, totalLength)
	for _, part := range parts {
		joined = append(joined, part...)
	}
	return joined
}

func longestCommonDigestRun(left, right []collectedChunk) int {
	previous := make([]int, len(right)+1)
	longest := 0

	for _, leftChunk := range left {
		current := make([]int, len(right)+1)
		for rightIndex, rightChunk := range right {
			if leftChunk.Digest != rightChunk.Digest {
				continue
			}

			current[rightIndex+1] = previous[rightIndex] + 1
			longest = max(longest, current[rightIndex+1])
		}
		previous = current
	}

	return longest
}
