package localProxy

import (
	"errors"
	"io"

	"github.com/ALiwoto/codex-dedup/src/core/utils/fastCdc"
	"github.com/ALiwoto/codex-dedup/src/core/utils/logging"
)

func newDedupMeasurementStore() *dedupMeasurementStore {
	return &dedupMeasurementStore{
		chunks: make(map[chunkIdentity]struct{}),
	}
}

func measureRequestBody(
	reader io.Reader,
	store *dedupMeasurementStore,
) (requestDedupMeasurement, error) {
	chunker, err := fastCdc.NewChunker(reader, fastCdc.DefaultChunkerConfig())
	if err != nil {
		return requestDedupMeasurement{}, err
	}

	var chunks []chunkIdentity
	for {
		chunk, nextErr := chunker.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return requestDedupMeasurement{}, nextErr
		}

		chunks = append(chunks, chunkIdentity{
			digest: chunk.Digest,
			length: chunk.Length,
		})
	}

	summary, complete := chunker.Summary()
	if !complete {
		return requestDedupMeasurement{}, errors.New("request body measurement did not complete")
	}

	return store.recordRequest(summary.TotalSize, chunks), nil
}

func startBodyMeasurement(
	source io.Reader,
	expectedBytes int64,
	store *dedupMeasurementStore,
) (*measuredBodyStream, <-chan bodyMeasurementResult) {
	measurementReader, measurementWriter := io.Pipe()
	results := make(chan bodyMeasurementResult, 1)

	go func() {
		measurement, err := measureRequestBody(measurementReader, store)
		_ = measurementReader.Close()
		results <- bodyMeasurementResult{
			measurement: measurement,
			err:         err,
		}
	}()

	return &measuredBodyStream{
		source:            source,
		measurementWriter: measurementWriter,
		expectedBytes:     expectedBytes,
	}, results
}

func logBodyMeasurement(measurement requestDedupMeasurement) {
	reusePercent := float64(0)
	if measurement.BodyBytes != 0 {
		reusePercent = float64(measurement.ReusableChunkBytes) / float64(measurement.BodyBytes) * 100
	}

	logging.Infof(
		"dedup measurement body_bytes=%d chunks=%d new_chunks=%d new_chunk_bytes=%d reusable_chunks=%d reusable_chunk_bytes=%d reuse_percent=%.2f simulated_cache_chunks=%d simulated_cache_bytes=%d",
		measurement.BodyBytes,
		measurement.ChunkCount,
		measurement.NewChunkCount,
		measurement.NewChunkBytes,
		measurement.ReusableChunkCount,
		measurement.ReusableChunkBytes,
		reusePercent,
		measurement.SimulatedCacheChunks,
		measurement.SimulatedCacheBytes,
	)
}
